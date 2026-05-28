package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"github.com/bassner/claudodex/internal/launcher"
	"github.com/bassner/claudodex/internal/modelconfig"
	"github.com/bassner/claudodex/internal/proxy"
)

type Config struct {
	Version string
	Stdin   *os.File
	Stdout  io.Writer
	Stderr  io.Writer

	Home          string
	CodexBaseURL  string
	TokenEndpoint string
	HTTPClient    *http.Client
	Launcher      Launcher
	Models        modelconfig.Config
}

type Launcher interface {
	Launch(ctx context.Context, args []string, cfg launcher.Config) error
}

func Run(ctx context.Context, cfg Config, args []string) int {
	if cfg.Version == "" {
		cfg.Version = "dev"
	}
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	if cfg.Launcher == nil {
		cfg.Launcher = launcher.ProcessLauncher{}
	}

	if launcher.IsStatusLineWrapperCommand(args) {
		return launcher.RunStatusLineWrapper(args[1:], cfg.Stdin, cfg.Stdout, cfg.Stderr)
	}
	var err error
	args, cfg.Models, err = extractStartupModelConfig(args, cfg.Models)
	if err != nil {
		fmt.Fprintln(cfg.Stderr, err)
		return 2
	}

	if launcher.ForcePassThrough(args) {
		return runClaude(ctx, cfg, args[1:])
	}

	if cmd, rest, ok := launcher.ClaudodexCommand(args); ok {
		return runCommand(ctx, cfg, cmd, rest)
	}

	return runClaude(ctx, cfg, args)
}

func runCommand(ctx context.Context, cfg Config, cmd string, args []string) int {
	switch cmd {
	case "clx:version":
		fmt.Fprintf(cfg.Stdout, "claudodex %s\n", cfg.Version)
		return 0
	case "clx:auth-status":
		return authStatus(cfg)
	case "clx:auth-login":
		return authLogin(ctx, cfg)
	case "clx:auth-logout":
		return authLogout(ctx, cfg)
	case "clx:serve":
		return serve(ctx, cfg, args)
	case "clx:doctor":
		return doctor(ctx, cfg)
	case "clx:usage":
		return usage(ctx, cfg)
	case "clx:reset-installation-id":
		return resetInstallationID(cfg)
	default:
		fmt.Fprintf(cfg.Stderr, "unknown Claudodex command %q\n", cmd)
		return 2
	}
}

func runClaude(ctx context.Context, cfg Config, args []string) int {
	err := cfg.Launcher.Launch(ctx, args, launcher.Config{
		Version:      cfg.Version,
		Stdin:        cfg.Stdin,
		Stdout:       cfg.Stdout,
		Stderr:       cfg.Stderr,
		Interactive:  launcher.DetectInteractive(args, cfg.Stdin, os.Stdout),
		Home:         cfg.Home,
		CodexBaseURL: cfg.CodexBaseURL,
		HTTPClient:   cfg.HTTPClient,
		Models:       cfg.Models,
	})
	if err == nil {
		return 0
	}

	var exitErr launcher.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	fmt.Fprintln(cfg.Stderr, err)
	return 1
}

func authStatus(cfg Config) int {
	store := auth.NewStore(cfg.Home)
	file, err := store.Load()
	if err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			fmt.Fprintln(cfg.Stdout, "not logged in")
			return 1
		}
		fmt.Fprintf(cfg.Stderr, "auth status failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(cfg.Stdout, "logged in using %s\n", file.AuthMode)
	if file.Tokens.AccountID != "" {
		fmt.Fprintf(cfg.Stdout, "account: %s\n", file.Tokens.AccountID)
	}
	return 0
}

func authLogin(ctx context.Context, cfg Config) int {
	fmt.Fprintln(cfg.Stdout, "Starting OpenAI OAuth login...")
	if _, err := auth.Login(ctx, auth.LoginOptions{Home: cfg.Home}); err != nil {
		fmt.Fprintf(cfg.Stderr, "auth login failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(cfg.Stdout, "logged in")
	return 0
}

func authLogout(ctx context.Context, cfg Config) int {
	if err := auth.RevokeAndDelete(ctx, auth.NewStore(cfg.Home), nil, ""); err != nil {
		if errors.Is(err, auth.ErrRevokeFailed) {
			fmt.Fprintf(cfg.Stderr, "auth logout warning: %v\n", err)
			fmt.Fprintln(cfg.Stdout, "logged out")
			return 0
		}
		fmt.Fprintf(cfg.Stderr, "auth logout failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(cfg.Stdout, "logged out")
	return 0
}

func serve(ctx context.Context, cfg Config, args []string) int {
	host := "127.0.0.1"
	port := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "missing value for --host")
				return 2
			}
			host = args[i]
		case "--port":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "missing value for --port")
				return 2
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed < 0 || parsed > 65535 {
				fmt.Fprintf(cfg.Stderr, "invalid --port %q\n", args[i])
				return 2
			}
			port = parsed
		default:
			fmt.Fprintf(cfg.Stderr, "unknown clx:serve arg %q\n", args[i])
			return 2
		}
	}

	file, err := auth.EnsureLoggedIn(ctx, cfg.Home, launcher.DetectInteractive(nil, cfg.Stdin, os.Stdout))
	if err != nil {
		fmt.Fprintln(cfg.Stderr, err)
		return 1
	}
	models, err := launcher.FetchCodexModels(ctx, launcher.Config{
		Version:      cfg.Version,
		Home:         cfg.Home,
		CodexBaseURL: cfg.CodexBaseURL,
		HTTPClient:   cfg.HTTPClient,
		Models:       cfg.Models,
	}, file)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "fetch Codex model metadata failed: %v\n", err)
		return 1
	}

	server := proxy.New(proxy.Config{
		Version:       cfg.Version,
		Interactive:   true,
		AuthPresent:   true,
		Home:          cfg.Home,
		CodexBaseURL:  cfg.CodexBaseURL,
		TokenEndpoint: cfg.TokenEndpoint,
		HTTPClient:    cfg.HTTPClient,
		Models:        models,
		ModelConfig:   cfg.Models,
	})
	addr, err := server.Start(host, port)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "proxy start failed: %v\n", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	fmt.Fprintf(cfg.Stdout, "http://%s\n", addr)
	<-ctx.Done()
	_ = server.Close()
	return 0
}

func doctor(ctx context.Context, cfg Config) int {
	fmt.Fprintf(cfg.Stdout, "claudodex: %s\n", cfg.Version)
	if _, err := launcher.LookClaude(); err != nil {
		fmt.Fprintf(cfg.Stdout, "claude: missing (%v)\n", err)
	} else {
		fmt.Fprintln(cfg.Stdout, "claude: found")
	}
	if _, err := launcher.LookCodex(); err != nil {
		fmt.Fprintln(cfg.Stdout, "codex: missing (warning only)")
	} else {
		fmt.Fprintln(cfg.Stdout, "codex: found")
	}
	file, err := auth.NewStore(cfg.Home).Load()
	if err != nil {
		fmt.Fprintln(cfg.Stdout, "auth: not logged in")
		fmt.Fprintln(cfg.Stdout, "usage: skipped (not logged in)")
	} else {
		fmt.Fprintln(cfg.Stdout, "auth: present")
		doctorUsage(ctx, cfg, file)
	}
	return 0
}

func doctorUsage(ctx context.Context, cfg Config, _ auth.File) {
	mapped, err := fetchCodexUsage(ctx, cfg, 5*time.Second)
	if err != nil {
		fmt.Fprintf(cfg.Stdout, "usage: failed (%v)\n", err)
		return
	}
	fmt.Fprintln(cfg.Stdout, "usage: reachable")
	if fiveHour, _ := mapped["five_hour"].(map[string]any); fiveHour != nil {
		fmt.Fprintf(cfg.Stdout, "usage five_hour: %.0f%%\n", numericUsage(fiveHour["utilization"]))
	}
	if sevenDay, _ := mapped["seven_day"].(map[string]any); sevenDay != nil {
		fmt.Fprintf(cfg.Stdout, "usage seven_day: %.0f%%\n", numericUsage(sevenDay["utilization"]))
	}
}

func usage(ctx context.Context, cfg Config) int {
	mapped, err := fetchCodexUsage(ctx, cfg, 10*time.Second)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "usage failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(cfg.Stdout, "Codex usage")
	printUsageWindow(cfg.Stdout, "five_hour", mapped["five_hour"])
	printUsageWindow(cfg.Stdout, "seven_day", mapped["seven_day"])
	printUsageWindow(cfg.Stdout, "seven_day_opus", mapped["seven_day_opus"])
	printUsageWindow(cfg.Stdout, "seven_day_sonnet", mapped["seven_day_sonnet"])
	if tier, _ := mapped["service_tier"].(string); tier != "" {
		fmt.Fprintf(cfg.Stdout, "service_tier: %s\n", tier)
	}
	return 0
}

func fetchCodexUsage(ctx context.Context, cfg Config, timeout time.Duration) (map[string]any, error) {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	store := auth.NewStore(cfg.Home)
	refresher := auth.NewRefresher(store, client, cfg.TokenEndpoint)
	file, err := refresher.EnsureFresh(checkCtx, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("auth refresh failed: %w", err)
	}
	installationID, err := auth.InstallationID(cfg.Home)
	if err != nil {
		return nil, fmt.Errorf("installation id failed: %w", err)
	}
	baseURL := cfg.CodexBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("CLAUDODEX_CODEX_BASE_URL")
	}
	if baseURL == "" {
		baseURL = codex.DefaultBaseURL
	}
	codexClient := codex.Client{
		BaseURL:    baseURL,
		HTTPClient: client,
		Version:    cfg.Version,
	}
	credentials := codex.Credentials{
		AccessToken:    file.Tokens.AccessToken,
		AccountID:      file.Tokens.AccountID,
		InstallationID: installationID,
		FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
	}
	raw, err := codexClient.FetchUsage(checkCtx, credentials)
	if err != nil {
		var upstream *codex.UpstreamError
		if !errors.As(err, &upstream) || upstream.Status != http.StatusUnauthorized {
			return nil, err
		}
		file, err = refresher.Refresh(checkCtx)
		if err != nil {
			return nil, fmt.Errorf("auth refresh failed: %w", err)
		}
		credentials.AccessToken = file.Tokens.AccessToken
		credentials.AccountID = file.Tokens.AccountID
		credentials.FedRAMP = file.Tokens.ChatGPTAccountIsFedRAMP
		raw, err = codexClient.FetchUsage(checkCtx, credentials)
	}
	if err != nil {
		return nil, err
	}
	return convert.CodexUsageToClaude(raw), nil
}

func printUsageWindow(w io.Writer, name string, value any) {
	window, _ := value.(map[string]any)
	if window == nil {
		fmt.Fprintf(w, "%s: unavailable\n", name)
		return
	}
	utilization := numericUsage(window["utilization"])
	if reset, _ := window["resets_at"].(string); reset != "" {
		fmt.Fprintf(w, "%s: %.0f%% (resets %s)\n", name, utilization, reset)
		return
	}
	fmt.Fprintf(w, "%s: %.0f%%\n", name, utilization)
}

func numericUsage(value any) float64 {
	if n, ok := value.(float64); ok {
		return n
	}
	return 0
}

func resetInstallationID(cfg Config) int {
	id, err := auth.ResetInstallationID(cfg.Home)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "reset installation id failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(cfg.Stdout, "installation id reset: %s\n", strings.TrimSpace(id))
	return 0
}
