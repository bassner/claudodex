package modelcatalog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bassner/claudodex/internal/filelock"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestCacheLockTimeoutBarrierBlocksOlderCatalogAndFreshSuccessRecovers(t *testing.T) {
	replacement := strings.Replace(loadCatalogJSON, `"gpt-8.2-sol"`, `"gpt-9-sol"`, 1)
	if replacement == loadCatalogJSON {
		t.Fatal("replacement fixture did not change the Opus catalog model")
	}

	tests := []struct {
		name          string
		status        int
		body          string
		removedPin    bool
		wantFreshPass bool
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"error":"unauthorized"}`},
		{name: "completed malformed JSON", status: http.StatusOK, body: `{"models":`},
		{name: "successful replacement", status: http.StatusOK, body: replacement, wantFreshPass: true},
		{name: "removed explicit pin", status: http.StatusOK, body: replacement, removedPin: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var state atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				switch state.Load() {
				case 0:
					w.WriteHeader(test.status)
					_, _ = io.WriteString(w, test.body)
				case 1:
					w.WriteHeader(http.StatusServiceUnavailable)
				case 2:
					_, _ = io.WriteString(w, replacement)
				}
			}))
			defer server.Close()

			opts := loadOptions(t, server)
			if test.removedPin {
				opts.Requested.Opus = "gpt-8.2-sol"
				opts.Pins = modelconfig.PinOpus
			}
			seedLoadCache(t, opts, loadTestTime.Add(-time.Hour))
			key := loadCacheKey(t, opts)
			unlock := holdCatalogCacheLock(t, opts, key)
			got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
			if test.wantFreshPass {
				if err != nil || got.FromCache || got.Config.Opus != "gpt-9-sol" {
					t.Fatalf("fresh replacement while cache lock is held = %#v, %v", got, err)
				}
			} else if err == nil || got.FromCache {
				t.Fatalf("authoritative observation while cache lock is held = %#v, %v", got, err)
			}
			assertPrivateRejectionBarrier(t, opts, key, loadTestTime)
			unlock()

			// Model an older in-flight launch finishing after the rejection marker.
			olderObservedAt := loadTestTime.Add(-30 * time.Minute)
			if err := writeCache(opts.Home, key, cachedCatalog{
				Schema: catalogCacheSchema, Scope: key, ObservedAt: olderObservedAt,
				FetchedAt: olderObservedAt, Models: loadFixture(t),
			}); err != nil {
				t.Fatalf("write older in-flight catalog: %v", err)
			}
			data, err := os.ReadFile(loadCachePath(t, opts))
			if err != nil {
				t.Fatal(err)
			}
			var written cachedCatalog
			if err := json.Unmarshal(data, &written); err != nil || !written.ObservedAt.Equal(olderObservedAt) {
				t.Fatalf("older in-flight catalog was not written: %#v, %v", written, err)
			}
			if _, err := readCache(opts.Home, key, loadTestTime.Add(time.Minute)); err == nil {
				t.Fatal("older catalog was readable despite a later rejection barrier")
			}

			state.Store(1)
			got, err = loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(time.Minute)))
			if err == nil || got.FromCache {
				t.Fatalf("catalog was resurrected during outage: %#v, %v", got, err)
			}

			// A later successful, unpinned discovery replaces the old cache and
			// removes the earlier barrier so a subsequent outage can use it.
			opts.Requested = modelconfig.Default()
			opts.Pins = 0
			state.Store(2)
			fresh, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(2*time.Minute)))
			if err != nil || fresh.FromCache || fresh.Config.Opus != "gpt-9-sol" {
				t.Fatalf("fresh recovery discovery = %#v, %v", fresh, err)
			}
			barriers, err := filepath.Glob(filepath.Join(opts.Home, ".claudodex", "model-catalog-rejections", key+"-*.json"))
			if err != nil || len(barriers) != 0 {
				t.Fatalf("fresh success left rejection barriers: %v, %v", barriers, err)
			}

			state.Store(1)
			recovered, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime.Add(3*time.Minute)))
			if err != nil || !recovered.FromCache || recovered.Config.Opus != "gpt-9-sol" {
				t.Fatalf("later outage did not use fresh catalog: %#v, %v", recovered, err)
			}
		})
	}
}

func TestLoadTruncatedHTTP200UsesRecentCache(t *testing.T) {
	truncatedBody := `{"models":[`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(truncatedBody)+128))
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, truncatedBody)
	}))
	defer server.Close()

	opts := loadOptions(t, server)
	seedLoadCache(t, opts, loadTestTime.Add(-time.Minute))
	var warning strings.Builder
	opts.Warnings = &warning
	got, err := loadAt(context.Background(), opts, loadFixedClock(loadTestTime))
	if err != nil || !got.FromCache || got.Config.Opus != "gpt-8.2-sol" {
		t.Fatalf("truncated HTTP response = %#v, %v", got, err)
	}
	if !strings.Contains(warning.String(), "using model metadata cached") {
		t.Fatalf("cache fallback warning = %q", warning.String())
	}
}

func TestRejectionBarrierIsPrivateAndContainsNoIdentity(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	opts := loadOptions(t, server)
	key := loadCacheKey(t, opts)
	if err := writeCacheBarrier(opts.Home, key, loadTestTime); err != nil {
		t.Fatalf("write rejection barrier: %v", err)
	}
	assertPrivateRejectionBarrier(t, opts, key, loadTestTime)
}

func TestReadCacheFailsClosedOnMalformedOrUnsafeRejectionBarrier(t *testing.T) {
	for _, test := range []struct {
		name     string
		mode     os.FileMode
		contents string
		symlink  bool
	}{
		{name: "malformed timestamp", mode: 0o600, contents: `"not-a-timestamp"`},
		{name: "trailing data after old timestamp", mode: 0o600, contents: `"2026-10-31T11:58:00Z" {}`},
		{name: "permissive file", mode: 0o644, contents: `"2026-10-31T12:00:00Z"`},
		{name: "symlink", mode: 0o600, contents: `"2026-10-31T12:00:00Z"`, symlink: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.NotFoundHandler())
			defer server.Close()
			opts := loadOptions(t, server)
			seedLoadCache(t, opts, loadTestTime.Add(-time.Minute))
			key := loadCacheKey(t, opts)
			dir := filepath.Join(opts.Home, ".claudodex", "model-catalog-rejections")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, key+"-fixture.json")
			if test.symlink {
				target := filepath.Join(t.TempDir(), "barrier.json")
				if err := os.WriteFile(target, []byte(test.contents), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, []byte(test.contents), test.mode); err != nil {
				t.Fatal(err)
			}
			if _, err := readCache(opts.Home, key, loadTestTime); err == nil {
				t.Fatal("cache was readable with malformed or unsafe rejection state")
			}
		})
	}
}

func holdCatalogCacheLock(t *testing.T, opts Options, key string) func() {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("file-backed catalog locks are unsupported on Windows")
	}
	path := filepath.Join(filepath.Dir(loadCachePath(t, opts)), key+".lock")
	unlock, err := filelock.Acquire(context.Background(), path)
	if err != nil {
		t.Fatalf("acquire catalog cache lock: %v", err)
	}
	return unlock
}

func assertPrivateRejectionBarrier(t *testing.T, opts Options, key string, want time.Time) {
	t.Helper()
	dir := filepath.Join(opts.Home, ".claudodex", "model-catalog-rejections")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat rejection directory: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("rejection directory permissions = %04o, want 0700", info.Mode().Perm())
	}
	paths, err := filepath.Glob(filepath.Join(dir, key+"-*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("rejection markers = %v, %v", paths, err)
	}
	identities := []string{opts.Credentials.AccessToken, opts.Credentials.AccountID, opts.Credentials.InstallationID, opts.UserID, opts.Issuer}
	for _, path := range paths {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !fileInfo.Mode().IsRegular() || fileInfo.Mode().Perm() != 0o600 {
			t.Fatalf("rejection marker permissions/type = %v, want regular 0600", fileInfo.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, identity := range identities {
			if identity != "" && (strings.Contains(string(data), identity) || strings.Contains(filepath.Base(path), identity)) {
				t.Fatalf("rejection marker contains private identity data")
			}
		}
		var observed time.Time
		if err := json.Unmarshal(data, &observed); err != nil || !observed.Equal(want) {
			t.Fatalf("rejection marker timestamp = %s, %v; want %s", observed, err, want)
		}
	}
}
