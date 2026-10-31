package modelcatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const loadCatalogJSON = `{"models":[
 {"slug":"gpt-8.2-sol","display_name":"Sol","supported_in_api":true,"visibility":"list","priority":0,"minimal_client_version":"0.166.0","context_window":500001,"max_context_window":900001,"default_reasoning_level":"medium","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"max"}],"service_tiers":[{"id":"priority","name":"Fast"}]},
 {"slug":"gpt-8.2-terra","display_name":"Terra","supported_in_api":true,"visibility":"list","priority":1,"context_window":500002,"max_context_window":900002,"default_reasoning_level":"low","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"}]},
 {"slug":"gpt-8-luna","display_name":"Luna","supported_in_api":true,"visibility":"list","priority":2,"context_window":500003,"max_context_window":900003,"default_reasoning_level":"medium","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"}]},
 {"slug":"gpt-8-astra","display_name":"Astra","supported_in_api":true,"visibility":"list","priority":3,"context_window":500004,"max_context_window":900004,"default_reasoning_level":"medium","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"max"},{"effort":"ultra"}]}
]}`

var loadTestTime = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

type loadRoundTripper func(*http.Request) (*http.Response, error)

func (f loadRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func loadOptions(t *testing.T, server *httptest.Server) Options {
	t.Helper()
	return Options{Home: t.TempDir(), BaseURL: server.URL + "/backend-api", Version: "0.4.0", UserID: "private-cache-user-id", Issuer: "https://auth.openai.com",
		Credentials: codex.Credentials{AccessToken: "private-cache-access-token", AccountID: "private-cache-account-id", InstallationID: "private-cache-installation-id"},
		HTTPClient:  server.Client(), Requested: modelconfig.Default()}
}

func loadFixture(t *testing.T) []codex.ModelInfo {
	t.Helper()
	var response codex.ModelsResponse
	if err := json.Unmarshal([]byte(loadCatalogJSON), &response); err != nil {
		t.Fatal(err)
	}
	return response.Models
}

func loadFixedClock(value time.Time) func() time.Time { return func() time.Time { return value } }

func loadCacheKey(t *testing.T, opts Options) string {
	t.Helper()
	backend, err := NormalizeBaseURL(opts.BaseURL)
	if err != nil {
		t.Fatal(err)
	}
	return scopeKey(opts, backend, codex.ModelsClientVersion(opts.Version))
}

func loadCachePath(t *testing.T, opts Options) string {
	t.Helper()
	return filepath.Join(opts.Home, ".claudodex", "model-catalog", loadCacheKey(t, opts)+".json")
}

func seedLoadCache(t *testing.T, opts Options, fetched time.Time) {
	t.Helper()
	key := loadCacheKey(t, opts)
	if err := writeCache(opts.Home, key, cachedCatalog{Schema: catalogCacheSchema, Scope: key, ObservedAt: fetched, FetchedAt: fetched, Models: loadFixture(t)}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFetchesEveryLaunchAndUsesActualWireProtocol(t *testing.T) {
	var calls atomic.Int32
	queries := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/backend-api/codex/models" {
			t.Errorf("metadata path = %q", r.URL.Path)
		}
		queries <- r.URL.Query().Get("client_version")
		_, _ = io.WriteString(w, loadCatalogJSON)
	}))
	defer server.Close()
	opts := loadOptions(t, server)
	for _, version := range []string{"0.4.0", "dev", "0.200.0"} {
		opts.Version = version
		got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
		if err != nil || got.FromCache {
			t.Fatalf("fresh discovery version %q: %#v, %v", version, got, err)
		}
		want := modelconfig.Config{Opus: "gpt-8.2-sol", Sonnet: "gpt-8.2-terra", Haiku: "gpt-8-luna", Fable: "gpt-8-astra"}
		if got.Config != want || !got.FetchedAt.Equal(loadTestTime) || len(got.Models) != 4 {
			t.Fatalf("fresh snapshot = %#v", got)
		}
	}
	gotQueries := []string{<-queries, <-queries, <-queries}
	if calls.Load() != 3 || strings.Join(gotQueries, ",") != "0.166.0,0.166.0,0.200.0" {
		t.Fatalf("fresh calls=%d wire versions=%v", calls.Load(), gotQueries)
	}
}

func TestLoadTransientFailuresUseRecentCacheWithAgeWarning(t *testing.T) {
	for _, status := range []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-2*time.Hour))
			var warning bytes.Buffer
			opts.Warnings = &warning
			got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
			if err != nil || !got.FromCache || got.Config.Opus != "gpt-8.2-sol" || !got.FetchedAt.Equal(loadTestTime.Add(-2*time.Hour)) {
				t.Fatalf("temporary failure = %#v, %v", got, err)
			}
			if !strings.Contains(warning.String(), "cached 2h0m0s ago") || !strings.Contains(warning.String(), "temporary network failure") {
				t.Fatalf("fallback warning = %q", warning.String())
			}
		})
	}
}

func TestLoadTransportAndOwnRequestTimeoutCanUseCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected actual network request") }))
	defer server.Close()
	for name, failure := range map[string]error{
		"transport":           &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary fixture network failure")},
		"own request timeout": context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Minute))
			opts.HTTPClient = &http.Client{Transport: loadRoundTripper(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > catalogTimeout || time.Until(deadline) <= 0 {
					t.Errorf("metadata request has no bounded independent deadline: %v", deadline)
				}
				return nil, failure
			})}
			got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
			if err != nil || !got.FromCache {
				t.Fatalf("network failure = %#v, %v", got, err)
			}
		})
	}
}

func TestLoadParentCancellationNeverFallsBackOrInvalidatesCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected actual network request") }))
	defer server.Close()
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Minute))
			var ctx context.Context
			var cancel context.CancelFunc
			if cause == context.Canceled {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			} else {
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()
			}
			got, err := loadAt(ctx, opts, loadFixedClock(loadTestTime))
			if !errors.Is(err, cause) || got.FromCache {
				t.Fatalf("parent cancellation = %#v, %v", got, err)
			}
			if _, err := readCache(opts.Home, loadCacheKey(t, opts), loadTestTime); err != nil {
				t.Fatalf("parent cancellation invalidated good metadata: %v", err)
			}
		})
	}
}

func TestLoadAuthoritativeRejectionPreventsLaterResurrection(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, `{"error":"fixture"}`},
		{"forbidden", 403, `{"error":"fixture"}`},
		{"bad request", 400, `{"error":"fixture"}`},
		{"not found", 404, `{"error":"fixture"}`},
		{"malformed JSON", 200, `{"models":`},
		{"duplicate catalog ID", 200, `{"models":[{"slug":"gpt-8-sol"},{"slug":"gpt-8-sol"}]}`},
		{"empty catalog", 200, `{"models":[]}`},
		{"missing selected family", 200, `{"models":[{"slug":"gpt-8.2-sol","supported_in_api":true,"visibility":"list","context_window":500001,"supported_reasoning_levels":[{"effort":"medium"}]}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var laterOutage atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if laterOutage.Load() {
					w.WriteHeader(503)
					return
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil || got.FromCache {
				t.Fatalf("authoritative rejection = %#v, %v", got, err)
			}
			laterOutage.Store(true)
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(time.Minute))); err == nil || got.FromCache {
				t.Fatalf("old catalog resurrected after authoritative rejection: %#v, %v", got, err)
			}
		})
	}
}

func TestLoadFreshCatalogRemovalInvalidatesExplicitPinForLaterOutage(t *testing.T) {
	var outage atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if outage.Load() {
			w.WriteHeader(503)
			return
		}
		_, _ = io.WriteString(w, strings.ReplaceAll(loadCatalogJSON, "gpt-8.2-sol", "gpt-9-sol"))
	}))
	defer server.Close()
	opts := loadOptions(t, server)
	opts.Requested.Opus, opts.Pins = "gpt-8.2-sol", modelconfig.PinOpus
	seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
	if _, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil {
		t.Fatal("removed explicit pin was accepted")
	}
	outage.Store(true)
	if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(time.Minute))); err == nil || got.FromCache {
		t.Fatalf("removed pin resurrected: %#v, %v", got, err)
	}
}

func TestLoadCacheScopeIsolation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	base := loadOptions(t, server)
	seedLoadCache(t, base, loadTestTime.Add(-time.Minute))
	for _, test := range []struct {
		name   string
		mutate func(*Options)
	}{
		{"user", func(o *Options) { o.UserID = "different-user" }},
		{"account", func(o *Options) { o.Credentials.AccountID = "different-account" }},
		{"backend", func(o *Options) { o.BaseURL += "/different-backend" }},
		{"issuer", func(o *Options) { o.Issuer = "https://different-issuer.invalid" }},
		{"region", func(o *Options) { o.Credentials.FedRAMP = true }},
		{"actual protocol", func(o *Options) { o.Version = "0.200.0" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := base
			test.mutate(&opts)
			if loadCacheKey(t, opts) == loadCacheKey(t, base) {
				t.Fatal("different cache scope had the same key")
			}
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil || got.FromCache {
				t.Fatalf("cross-scope catalog reused: %#v, %v", got, err)
			}
		})
	}
	for _, productVersion := range []string{"dev", "0.165.0", "0.166.0"} {
		opts := base
		opts.Version = productVersion
		if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err != nil || !got.FromCache {
			t.Fatalf("same actual wire protocol did not share catalog (%s): %#v, %v", productVersion, got, err)
		}
	}
}

func TestLoadWithoutIdentityNeverReadsOrWritesCache(t *testing.T) {
	var outage atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if outage.Load() {
			w.WriteHeader(503)
			return
		}
		_, _ = io.WriteString(w, loadCatalogJSON)
	}))
	defer server.Close()
	for _, missing := range []string{"user", "account", "both"} {
		t.Run(missing, func(t *testing.T) {
			outage.Store(false)
			opts := loadOptions(t, server)
			if missing != "account" {
				opts.UserID = ""
			}
			if missing != "user" {
				opts.Credentials.AccountID = ""
			}
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err != nil || got.FromCache {
				t.Fatalf("fresh anonymous metadata = %#v, %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(opts.Home, ".claudodex", "model-catalog")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unknown identity created cache state: %v", err)
			}
			outage.Store(true)
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil || got.FromCache || !strings.Contains(err.Error(), "known user and account") {
				t.Fatalf("unknown identity outage = %#v, %v", got, err)
			}
		})
	}
}

func TestCatalogCachePrivacyAndPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, loadCatalogJSON) }))
	defer server.Close()
	opts := loadOptions(t, server)
	if _, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err != nil {
		t.Fatal(err)
	}
	path := loadCachePath(t, opts)
	for _, test := range []struct {
		path string
		mode os.FileMode
	}{{path, 0o600}, {filepath.Dir(path), 0o700}} {
		info, err := os.Stat(test.path)
		if err != nil || info.Mode().Perm() != test.mode {
			t.Fatalf("cache permissions %s: %v, %v", test.path, info, err)
		}
	}
	if err := filepath.WalkDir(filepath.Dir(path), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range []string{opts.Credentials.AccessToken, opts.Credentials.AccountID, opts.Credentials.InstallationID, opts.UserID, "access_token", "refresh_token", "account_id", "user_id"} {
			if strings.Contains(string(data), secret) || strings.Contains(filepath.Base(path), secret) {
				t.Errorf("cache persisted private authentication field %q", secret)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestReadCacheRejectsExpiredFutureCorruptAndUnsafeEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	for _, test := range []struct {
		name        string
		mutate      func(*cachedCatalog)
		raw         string
		permissions os.FileMode
		oversized   bool
	}{
		{name: "expired", mutate: func(e *cachedCatalog) {
			e.ObservedAt = loadTestTime.Add(-catalogCacheMaxAge - time.Nanosecond)
			e.FetchedAt = e.ObservedAt
		}},
		{name: "future fetch", mutate: func(e *cachedCatalog) { e.FetchedAt = loadTestTime.Add(time.Second) }},
		{name: "future observation", mutate: func(e *cachedCatalog) { e.ObservedAt = loadTestTime.Add(time.Second) }},
		{name: "observation after fetch", mutate: func(e *cachedCatalog) { e.ObservedAt = e.FetchedAt.Add(time.Nanosecond) }},
		{name: "missing timestamp", mutate: func(e *cachedCatalog) { e.FetchedAt = time.Time{} }},
		{name: "wrong schema", mutate: func(e *cachedCatalog) { e.Schema++ }},
		{name: "wrong scope", mutate: func(e *cachedCatalog) { e.Scope = "different-scope" }},
		{name: "rejected", mutate: func(e *cachedCatalog) { e.Invalid = true }},
		{name: "invalid model metadata", mutate: func(e *cachedCatalog) { e.Models[0].Slug = "invalid/model" }},
		{name: "corrupt JSON", raw: "{"},
		{name: "public file", permissions: 0o644},
		{name: "oversized file", oversized: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
			path := loadCachePath(t, opts)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if test.mutate != nil {
				var entry cachedCatalog
				if err := json.Unmarshal(data, &entry); err != nil {
					t.Fatal(err)
				}
				test.mutate(&entry)
				data, err = json.Marshal(entry)
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.raw != "" {
				data = []byte(test.raw)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if test.permissions != 0 {
				if err := os.Chmod(path, test.permissions); err != nil {
					t.Fatal(err)
				}
			}
			if test.oversized {
				if err := os.Truncate(path, catalogCacheLimit+1); err != nil {
					t.Fatal(err)
				}
			}
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil || got.FromCache {
				t.Fatalf("unsafe cached catalog used: %#v, %v", got, err)
			}
		})
	}
}

func TestLoadCacheAcceptsExactlyTwentyFourHours(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	opts := loadOptions(t, server)
	seedLoadCache(t, opts, loadTestTime.Add(-catalogCacheMaxAge))
	if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err != nil || !got.FromCache {
		t.Fatalf("exactly 24-hour catalog = %#v, %v", got, err)
	}
}

func TestCacheRejectsSymlinksAndNonPrivateDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	for _, kind := range []string{"file symlink", "directory symlink", "public directory"} {
		t.Run(kind, func(t *testing.T) {
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
			path := loadCachePath(t, opts)
			if kind == "file symlink" {
				target := filepath.Join(t.TempDir(), "catalog.json")
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			} else if kind == "directory symlink" {
				dir := filepath.Dir(path)
				target := filepath.Join(t.TempDir(), "catalog")
				if err := os.Rename(dir, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, dir); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); err == nil || got.FromCache {
				t.Fatalf("unsafe path used: %#v, %v", got, err)
			}
			key := loadCacheKey(t, opts)
			if err := writeCache(opts.Home, key, cachedCatalog{Schema: catalogCacheSchema, Scope: key, ObservedAt: loadTestTime, FetchedAt: loadTestTime, Models: loadFixture(t)}); err == nil {
				t.Fatal("cache write accepted unsafe path")
			}
		})
	}
}

func TestOptionalCacheWriteFailureDoesNotBlockFreshDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, loadCatalogJSON) }))
	defer server.Close()
	opts := loadOptions(t, server)
	if err := os.MkdirAll(filepath.Join(opts.Home, ".claudodex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opts.Home, ".claudodex", "model-catalog"), []byte("fixture blocks directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	var warning bytes.Buffer
	opts.Warnings = &warning
	got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
	if err != nil || got.FromCache || got.Config.Opus != "gpt-8.2-sol" || !strings.Contains(warning.String(), "could not be cached") {
		t.Fatalf("optional cache write failure = %#v, %v, warning=%q", got, err, warning.String())
	}
}

func TestOlderInflightSuccessCannotOverwriteNewerRejection(t *testing.T) {
	var calls atomic.Int32
	firstStarted, releaseFirst := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			close(firstStarted)
			<-releaseFirst
			_, _ = io.WriteString(w, loadCatalogJSON)
		case 2:
			w.WriteHeader(401)
		default:
			w.WriteHeader(503)
		}
	}))
	defer server.Close()
	released := false
	defer func() {
		if !released {
			close(releaseFirst)
		}
	}()
	opts := loadOptions(t, server)
	firstDone := make(chan error, 1)
	go func() { _, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime)); firstDone <- err }()
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first discovery did not start")
	}
	if _, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(time.Minute))); err == nil {
		t.Fatal("newer rejected discovery succeeded")
	}
	close(releaseFirst)
	released = true
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("older successful request failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first discovery did not finish")
	}
	if got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(2*time.Minute))); err == nil || got.FromCache {
		t.Fatalf("older success replaced newer rejection: %#v, %v", got, err)
	}
}

func TestNormalizeCatalogBackend(t *testing.T) {
	for raw, want := range map[string]string{
		"": codex.DefaultBaseURL,
		" https://CHATGPT.COM:443/backend-api/// ": "https://chatgpt.com/backend-api",
		"http://LOCALHOST:80/fixture/":             "http://localhost/fixture",
		"https://[::1]:443/backend-api/":           "https://[::1]/backend-api",
	} {
		got, err := NormalizeBaseURL(raw)
		if err != nil || got != want {
			t.Errorf("NormalizeBaseURL(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, invalid := range []string{"https://user:secret@chatgpt.com/backend-api", "https://chatgpt.com/backend-api?account=secret", "https://chatgpt.com/backend-api#fragment", "file:///tmp/catalog", "https:///no-host", "opaque:catalog"} {
		if _, err := NormalizeBaseURL(invalid); err == nil {
			t.Errorf("unsafe backend accepted: %q", invalid)
		}
	}
}

func TestLoadDestinationConstraintPrecedesNetworkAndCache(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) }))
	defer server.Close()
	opts := loadOptions(t, server)
	seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
	opts.Credentials.AllowedOrigin = "https://chatgpt.com"
	got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
	if err == nil || got.FromCache || calls.Load() != 0 {
		t.Fatalf("borrowed credentials reached a custom backend or cache: %#v, %v, requests=%d", got, err, calls.Load())
	}
}
