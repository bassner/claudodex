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
	"strings"
	"syscall"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/filelock"
	"github.com/bassner/claudodex/internal/modelcatalog"
	"github.com/bassner/claudodex/internal/modelconfig"
	"github.com/bassner/claudodex/internal/proxy"
	"github.com/bassner/claudodex/internal/startup"
)

type Config struct {
	Version       string
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	Interactive   bool
	Home          string
	CodexBaseURL  string
	HTTPClient    *http.Client
	TokenEndpoint string
	Models        modelconfig.Config
	ModelPins     modelconfig.Pins
	// StatusLineExecutable is an already pinned helper supplied before an update.
	StatusLineExecutable string
	Startup              *startup.Display
	RuntimeContext       context.Context
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
	progress := cfg.Startup
	if progress == nil {
		progress = startup.New(stderr, cfg.Version, cfg.Interactive && !DirectClaudeFastPath(args))
	}
	defer progress.Close()
	runtimeCtx := cfg.RuntimeContext
	if runtimeCtx == nil {
		runtimeCtx = ctx
	}
	ctx = progress.Context(ctx)
	progress.Info("🔎", "Locating Claude Code")
	claudePath, err := LookClaude()
	if err != nil {
		return fmt.Errorf("missing required claude binary: %w", err)
	}

	if DirectClaudeFastPath(args) {
		return runChild(ctx, claudePath, args, BuildClaudePrivacyEnv(os.Environ()), stdin, stdout, stderr, false)
	}
	claudePath, err = filepath.EvalSymlinks(claudePath)
	if err != nil {
		return fmt.Errorf("resolve installed Claude Code: %w", err)
	}
	progress.Info("🔧", "Verifying binary and preparing the compatibility patch")
	selected, err := selectClaudeExecutableWithRetention(ctx, cfg.Home, claudePath, cfg.Version, modelCfg, progress.Warnings(), false)
	if err != nil {
		return err
	}
	defer selected.Release()
	progress.Info("✅", "Claude Code %s compatibility ready", displayClaudeVersion(selected.Version))
	if cfg.CodexBaseURL == "" {
		cfg.CodexBaseURL = os.Getenv("CLAUDODEX_CODEX_BASE_URL")
	}
	cfg.CodexBaseURL, err = modelcatalog.NormalizeBaseURL(cfg.CodexBaseURL)
	if err != nil {
		return err
	}
	provider := auth.NewProvider(auth.NewStore(cfg.Home), cfg.HTTPClient, cfg.TokenEndpoint)
	progress.Info("🔐", "Loading Codex login")
	file, err := provider.EnsureLoggedIn(ctx, cfg.Interactive)
	if err != nil {
		return err
	}
	cfg.Stderr = progress.Warnings()
	progress.Info("📚", "Loading the Codex model catalog")
	snapshot, err := ResolveCodexModels(ctx, cfg, file, provider)
	if err != nil {
		return err
	}
	models := snapshot.Models
	if snapshot.Config != modelCfg {
		progress.Info("🔧", "Preparing compatibility for the discovered models")
		final, err := selectClaudeExecutableWithRetention(ctx, cfg.Home, selected.SourcePath, cfg.Version, snapshot.Config, progress.Warnings(), true)
		if err != nil {
			return fmt.Errorf("prepare Claude Code for discovered model labels: %w; update Claudodex or select models with --claudodex-models", err)
		}
		defer final.Release()
		selected = final
	} else if strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CLAUDE_PATCH")) != "1" {
		root, err := claudeCacheRoot(cfg.Home)
		if err != nil {
			return err
		}
		unlock, err := filelock.Acquire(ctx, filepath.Join(root, ".lock"))
		if err != nil {
			return err
		}
		retainClaudeSources(ctx, root, cfg.Version, snapshot.Config)
		unlock()
	}
	modelCfg = snapshot.Config
	progress.Info("✅", "Models ready: %s, %s, %s, %s", modelCfg.Fable, modelCfg.Opus, modelCfg.Sonnet, modelCfg.Haiku)
	claudePath = selected.SourcePath
	if requested, ok := explicitModelArg(args); ok && isDirectCodexModel(requested) {
		if _, err := modelcatalog.ValidateTarget(models, modelconfig.StripLongContext(requested), cfg.Version); err != nil {
			return err
		}
	}
	claude246Compatibility := claudeFastModeSettingsFallbackRequired(ctx, claudePath)
	claude258Compatibility := claudeThreeTierPickerArgsRequired(ctx, claudePath)
	claude259SourcePicker := claudeSourcePatchedPickerArgsRequired(ctx, claudePath)
	progress.Info("⚙️ ", "Preparing Claude Code settings and status line")
	claudeConfigDir, err := PrepareClaudeConfigSidecar(cfg.Home, modelCfg)
	if err != nil {
		return fmt.Errorf("prepare Claude Code compatibility config: %w", err)
	}
	if claude258Compatibility {
		if err := EnsureClaudeRemoteControlFeatureCache(claudeConfigDir); err != nil {
			return fmt.Errorf("prepare Claude Code Remote Control compatibility: %w", err)
		}
	}
	if err := WriteClaudeModelCapabilitiesCache(claudeConfigDir, models, modelCfg); err != nil {
		return fmt.Errorf("prepare Claude Code model capabilities: %w", err)
	}
	configMirror, err := StartClaudeConfigMirror(runtimeCtx, claudeConfigDir, modelCfg)
	if err != nil {
		return fmt.Errorf("start Claude Code config mirror: %w", err)
	}
	childArgs := RewriteClaudeModelArgsWithConfig(args, modelCfg)
	if claude258Compatibility {
		childArgs = RewriteClaudeModelArgsForThreeTierPicker(args, modelCfg)
	}
	if claude259SourcePicker {
		childArgs = RewriteClaudeModelArgsForSourcePatchedThreeTierPicker(args, modelCfg)
	}
	var compatibilitySettings map[string]any
	if claude246Compatibility {
		compatibilitySettings = claude246ModelPickerSettings(modelCfg)
	}
	childArgs, err = prepareStatusLineFlagSettingsForCatalog(claudeConfigDir, childArgs, compatibilitySettings, modelCfg, models, cfg.Version, func() (string, error) {
		if cfg.StatusLineExecutable != "" {
			return cfg.StatusLineExecutable, nil
		}
		return PinStatusLineExecutable(cfg.Home, currentExecutable())
	})
	if err != nil {
		_ = configMirror.Close()
		return fmt.Errorf("prepare Claude Code statusline compatibility: %w", err)
	}
	childArgs = DisableClaudeChrome(childArgs)

	serverCfg := proxy.Config{
		Version:       cfg.Version,
		Interactive:   cfg.Interactive,
		AuthPresent:   true,
		Home:          cfg.Home,
		CodexBaseURL:  cfg.CodexBaseURL,
		HTTPClient:    cfg.HTTPClient,
		TokenEndpoint: cfg.TokenEndpoint,
		Models:        models,
		ModelConfig:   modelCfg,
		AuthProvider:  provider,
		AuthAccountID: file.Tokens.AccountID,
		AuthUserID:    file.Tokens.UserID,
	}
	if claude246Compatibility {
		serverCfg.FastModeEnabledFallback = fastModeSettingsResolver(childArgs)
	}
	progress.Info("🔌", "Starting local compatibility proxies")
	oauthServer := proxy.New(serverCfg)
	addr, err := oauthServer.Start("127.0.0.1", 0)
	if err != nil {
		_ = configMirror.Close()
		return fmt.Errorf("start local proxy: %w", err)
	}
	defer oauthServer.Close()

	port := oauthServer.Port()
	if port == 0 {
		_ = configMirror.Close()
		return fmt.Errorf("local proxy did not expose a port at %s", addr)
	}
	// Claude's SDK uses the Unix socket, but its organization-settings Axios
	// client uses the normal HTTPS proxy. Both paths must reach the local API.
	oauthProxy, err := StartOAuthProxy(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		_ = configMirror.Close()
		return fmt.Errorf("start Claude Code OAuth compatibility proxy: %w", err)
	}
	defer oauthProxy.Close()
	apiServer, apiSocket, apiSocketDir, err := startAnthropicAPIServer(serverCfg, oauthProxy.serverTLS)
	if err != nil {
		_ = configMirror.Close()
		return fmt.Errorf("start local Anthropic API socket: %w", err)
	}
	if apiServer != nil {
		defer apiServer.Close()
	}
	if apiSocketDir != "" {
		defer os.RemoveAll(apiSocketDir)
	}
	childEnv := BuildClaudeEnv(os.Environ(), port, claudeConfigDir, apiSocket, oauthProxy.ProxyURL(), oauthProxy.CAPath(), models, modelCfg)
	if _, complete := modelcatalog.AutoCompactWindows(models, modelCfg); complete && supportsNativeModelCompaction(selected.Version) {
		nativeEnv := envMap(childEnv)
		delete(nativeEnv, "CLAUDE_CODE_AUTO_COMPACT_WINDOW")
		childEnv = flattenEnv(nativeEnv)
		progress.Info("✅", "Auto-compaction follows the active model's context window")
	} else {
		progress.Warn("Using a shared compaction ceiling; per-model compaction needs a supported Claude version and representable model limits")
	}
	if claude258Compatibility {
		childEnv = WithPrivateFeatureCache(childEnv)
	}
	childEnv = WithRealClaudeBridgeAuth(childEnv)
	if runtimeModel, ok := explicitModelArg(childArgs); ok {
		childEnv = WithFriendlyCustomModelOptionForConfig(childEnv, runtimeModel, modelCfg)
	}
	progress.Info("🚀", "Launching Claude Code")
	progress.Close()
	childErr := runChild(runtimeCtx, selected.Executable, childArgs, childEnv, stdin, stdout, stderr, !cfg.Interactive)
	return errors.Join(childErr, configMirror.Close())
}

func supportsNativeModelCompaction(version string) bool {
	return version == "2.1.322" || version == "2.1.319" || version == "2.1.318" || version == "2.1.314" || version == "2.1.316" || version == "2.1.317"
}

func isDirectCodexModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(modelconfig.StripLongContext(model)), "gpt-")
}

func startAnthropicAPIServer(serverCfg proxy.Config, cert tls.Certificate) (*proxy.Server, string, string, error) {
	if runtime.GOOS == "windows" {
		return nil, "", "", nil
	}
	socketDir, err := os.MkdirTemp("", "claudodex-api-*")
	if err != nil {
		return nil, "", "", err
	}
	socketPath := filepath.Join(socketDir, "api.sock")
	server := proxy.New(serverCfg)
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if _, err := server.StartUnixTLS(socketPath, tlsConfig); err != nil {
		_ = server.Close()
		_ = os.RemoveAll(socketDir)
		return nil, "", "", err
	}
	return server, socketPath, socketDir, nil
}

func FetchCodexModels(ctx context.Context, cfg Config, file auth.File) ([]codex.ModelInfo, error) {
	snapshot, err := ResolveCodexModels(ctx, cfg, file)
	return snapshot.Models, err
}

// Production callers share their authentication provider so an access-token
// rejection during discovery can refresh once before the proxies exist.
// Metadata-only compatibility helpers may omit it.
func ResolveCodexModels(ctx context.Context, cfg Config, file auth.File, providers ...*auth.Provider) (modelcatalog.Snapshot, error) {
	installationID, err := auth.InstallationID(cfg.Home)
	if err != nil {
		return modelcatalog.Snapshot{}, err
	}
	baseURL := cfg.CodexBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("CLAUDODEX_CODEX_BASE_URL")
	}
	options := modelcatalog.Options{
		Home: cfg.Home, BaseURL: baseURL, Version: cfg.Version,
		UserID: file.Tokens.UserID, Issuer: file.Issuer, HTTPClient: cfg.HTTPClient,
		Requested: cfg.Models, Pins: cfg.ModelPins, Warnings: cfg.Stderr,
		Credentials: codex.Credentials{
			AccessToken:    file.Tokens.AccessToken,
			AccountID:      file.Tokens.AccountID,
			InstallationID: installationID,
			FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
			AllowedOrigin:  file.AllowedOrigin,
		},
	}
	snapshot, err := modelcatalog.Load(ctx, options)
	var upstream *codex.UpstreamError
	if err == nil || len(providers) == 0 || providers[0] == nil || !errors.As(err, &upstream) || upstream.Status != http.StatusUnauthorized {
		return snapshot, err
	}
	refreshed, err := providers[0].Refresh(ctx, file.Tokens.AccessToken)
	if err != nil {
		return modelcatalog.Snapshot{}, err
	}
	if refreshed.Tokens.UserID != file.Tokens.UserID || refreshed.Tokens.AccountID != file.Tokens.AccountID {
		return modelcatalog.Snapshot{}, fmt.Errorf("login changed during model discovery; restart this session after signing in")
	}
	options.UserID, options.Issuer = refreshed.Tokens.UserID, refreshed.Issuer
	options.Credentials.AccessToken = refreshed.Tokens.AccessToken
	options.Credentials.AccountID = refreshed.Tokens.AccountID
	options.Credentials.FedRAMP = refreshed.Tokens.ChatGPTAccountIsFedRAMP
	options.Credentials.AllowedOrigin = refreshed.AllowedOrigin
	return modelcatalog.Load(ctx, options)
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
