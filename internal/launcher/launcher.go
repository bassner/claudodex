package launcher

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
	"github.com/bassner/claudodex/internal/proxy"
)

type Config struct {
	Version      string
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Interactive  bool
	Home         string
	CodexBaseURL string
	HTTPClient   *http.Client
	Models       modelconfig.Config
}

type ProcessLauncher struct{}

var childTerminateTimeout = 5 * time.Second

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("claude exited with status %d", e.Code)
}

func LookClaude() (string, error) {
	return exec.LookPath("claude")
}

func LookCodex() (string, error) {
	return exec.LookPath("codex")
}

func (ProcessLauncher) Launch(ctx context.Context, args []string, cfg Config) error {
	modelCfg := cfg.Models.Normalize()
	claudePath, err := LookClaude()
	if err != nil {
		return fmt.Errorf("missing required claude binary: %w", err)
	}
	stdin := cfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	if DirectClaudeFastPath(args) {
		return runChild(ctx, claudePath, args, BuildClaudePrivacyEnv(os.Environ()), stdin, stdout, stderr, false)
	}

	file, err := auth.EnsureLoggedIn(ctx, cfg.Home, cfg.Interactive)
	if err != nil {
		return err
	}
	models, err := FetchCodexModels(ctx, cfg, file)
	if err != nil {
		return err
	}

	serverCfg := proxy.Config{
		Version:      cfg.Version,
		Interactive:  cfg.Interactive,
		AuthPresent:  true,
		Home:         cfg.Home,
		CodexBaseURL: cfg.CodexBaseURL,
		HTTPClient:   cfg.HTTPClient,
		Models:       models,
		ModelConfig:  modelCfg,
	}
	oauthServer := proxy.New(serverCfg)
	addr, err := oauthServer.Start("127.0.0.1", 0)
	if err != nil {
		return fmt.Errorf("start local proxy: %w", err)
	}
	defer oauthServer.Close()

	port := oauthServer.Port()
	if port == 0 {
		return fmt.Errorf("local proxy did not expose a port at %s", addr)
	}
	apiServer, apiSocket, apiSocketDir, apiCAPath, err := startAnthropicAPIServer(serverCfg)
	if err != nil {
		return fmt.Errorf("start local Anthropic API socket: %w", err)
	}
	if apiServer != nil {
		defer apiServer.Close()
	}
	if apiSocketDir != "" {
		defer os.RemoveAll(apiSocketDir)
	}
	if apiCAPath != "" {
		defer os.Remove(apiCAPath)
	}
	claudeConfigDir, err := PrepareClaudeConfigSidecar(cfg.Home, modelCfg)
	if err != nil {
		return fmt.Errorf("prepare Claude Code compatibility config: %w", err)
	}
	if err := WriteClaudeModelCapabilitiesCache(claudeConfigDir, models, modelCfg); err != nil {
		return fmt.Errorf("prepare Claude Code model capabilities: %w", err)
	}
	configMirror, err := StartClaudeConfigMirror(ctx, claudeConfigDir, modelCfg)
	if err != nil {
		return fmt.Errorf("start Claude Code config mirror: %w", err)
	}
	httpsProxy := ""
	caPath := ""
	var oauthProxy *OAuthProxy
	if apiSocket == "" {
		oauthProxy, err = StartOAuthProxy(fmt.Sprintf("http://127.0.0.1:%d", port))
		if err != nil {
			_ = configMirror.Close()
			return fmt.Errorf("start Claude Code OAuth compatibility proxy: %w", err)
		}
		defer oauthProxy.Close()
		httpsProxy = oauthProxy.ProxyURL()
		caPath = oauthProxy.CAPath()
	}
	childArgs := RewriteClaudeModelArgsWithConfig(args, modelCfg)
	childArgs, err = PrepareStatusLineFlagSettings(claudeConfigDir, childArgs)
	if err != nil {
		_ = configMirror.Close()
		return fmt.Errorf("prepare Claude Code statusline compatibility: %w", err)
	}
	childArgs = DisableClaudeChrome(childArgs)
	if apiCAPath != "" {
		caPath = apiCAPath
	}
	childEnv := BuildClaudeEnv(os.Environ(), port, claudeConfigDir, apiSocket, httpsProxy, caPath, models, modelCfg)
	childEnv = WithRealClaudeBridgeAuth(childEnv)
	if runtimeModel, ok := explicitModelArg(childArgs); ok {
		childEnv = WithFriendlyCustomModelOption(childEnv, runtimeModel)
	}
	claudePath = prepareClaudeExecutable(ctx, cfg.Home, claudePath, cfg.Version, modelCfg, stderr)
	childErr := runChild(ctx, claudePath, childArgs, childEnv, stdin, stdout, stderr, !cfg.Interactive)
	return errors.Join(childErr, configMirror.Close())
}

func startAnthropicAPIServer(serverCfg proxy.Config) (*proxy.Server, string, string, string, error) {
	if runtime.GOOS == "windows" {
		return nil, "", "", "", nil
	}
	cert, caPEM, err := generateOAuthProxyCertificate()
	if err != nil {
		return nil, "", "", "", err
	}
	caPath, err := writeOAuthProxyCAFile(caPEM)
	if err != nil {
		return nil, "", "", "", err
	}
	socketDir, err := os.MkdirTemp("", "claudodex-api-*")
	if err != nil {
		_ = os.Remove(caPath)
		return nil, "", "", "", err
	}
	socketPath := filepath.Join(socketDir, "api.sock")
	server := proxy.New(serverCfg)
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if _, err := server.StartUnixTLS(socketPath, tlsConfig); err != nil {
		_ = server.Close()
		_ = os.RemoveAll(socketDir)
		_ = os.Remove(caPath)
		return nil, "", "", "", err
	}
	return server, socketPath, socketDir, caPath, nil
}

func FetchCodexModels(ctx context.Context, cfg Config, file auth.File) ([]codex.ModelInfo, error) {
	modelCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	installationID, err := auth.InstallationID(cfg.Home)
	if err != nil {
		return nil, err
	}
	client := codex.Client{BaseURL: cfg.CodexBaseURL, HTTPClient: cfg.HTTPClient, Version: cfg.Version}
	models, err := client.FetchModels(modelCtx, codex.Credentials{
		AccessToken:    file.Tokens.AccessToken,
		AccountID:      file.Tokens.AccountID,
		InstallationID: installationID,
		FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch Codex model metadata: %w", err)
	}
	if !requiredCodexModelsHaveContextWindows(models, cfg.Models.Normalize()) {
		return nil, fmt.Errorf("Codex model metadata missing context windows for required models")
	}
	return models, nil
}

func requiredCodexModelsHaveContextWindows(models []codex.ModelInfo, modelCfg modelconfig.Config) bool {
	for _, slug := range modelCfg.RequiredModels() {
		if _, ok := catalogContextWindow(models, slug); !ok {
			return false
		}
	}
	return true
}

func runChild(ctx context.Context, path string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer, forwardInterrupt bool) error {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	signals := make(chan os.Signal, 4)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	for {
		select {
		case sig := <-signals:
			if sig == os.Interrupt {
				if forwardInterrupt {
					_ = forwardSignal(cmd, sig)
				}
				continue
			}
			return signalAndWait(cmd, done, sig, childTerminateTimeout)
		case <-ctx.Done():
			return signalAndWait(cmd, done, syscall.SIGTERM, childTerminateTimeout)
		case err := <-done:
			return childExitError(err)
		}
	}
}

func childExitError(err error) error {
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				return ExitError{Code: 128 + int(status.Signal())}
			}
			return ExitError{Code: exit.ExitCode()}
		}
		return err
	}
	return nil
}

func signalAndWait(cmd *exec.Cmd, done <-chan error, sig os.Signal, timeout time.Duration) error {
	_ = forwardSignal(cmd, sig)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			return childExitError(err)
		}
		if exitErr := signalExitError(sig); exitErr != nil {
			return exitErr
		}
		return nil
	case <-timer.C:
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return childExitError(<-done)
	}
}

func signalExitError(sig os.Signal) error {
	signalValue, ok := sig.(syscall.Signal)
	if !ok {
		return nil
	}
	return ExitError{Code: 128 + int(signalValue)}
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	signalValue, ok := sig.(syscall.Signal)
	if !ok {
		return nil
	}
	return cmd.Process.Signal(signalValue)
}
