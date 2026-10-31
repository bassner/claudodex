package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func providerFixture(t *testing.T) (*Provider, string) {
	t.Helper()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".codex"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", "")
	p := NewProvider(NewStore(home), nil, "")
	return p, home
}

func borrowedFixture(token, user, account string) File {
	return File{AuthMode: "chatgpt", Source: SourceCodex, AllowedOrigin: CodexAllowedOrigin, Tokens: Tokens{
		AccessToken: token, UserID: user, AccountID: account, AccessTokenExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}
}

func TestProviderPrefersDirectAndNeverReadsNative(t *testing.T) {
	p, _ := providerFixture(t)
	if err := p.store.Save(File{Tokens: Tokens{AccessToken: "direct", RefreshToken: "direct-refresh"}}); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		t.Fatal("native read with active direct login")
		return File{}, nil
	}
	file, err := p.EnsureFresh(context.Background(), 5*time.Minute)
	if err != nil || file.Source != SourceDirect || file.Tokens.AccessToken != "direct" {
		t.Fatalf("source=%q, err=%v", file.Source, err)
	}
}

func TestProviderBorrowsWithoutCopyingTokens(t *testing.T) {
	p, home := providerFixture(t)
	var reads atomic.Int32
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads.Add(1)
		return borrowedFixture("borrowed-access", "user", "account"), nil
	}
	file, err := p.Load(context.Background())
	if err != nil || file.Source != SourceCodex || file.Tokens.RefreshToken != "" || file.AllowedOrigin != CodexAllowedOrigin {
		t.Fatalf("source=%q, err=%v", file.Source, err)
	}
	dir := filepath.Join(home, ".claudodex")
	if _, err := os.Stat(filepath.Join(dir, codexReferenceName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("status created persistent reference")
	}
	file, err = p.EnsureFresh(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reads.Load() != 1 {
		t.Fatalf("coalesced cached native reads=%d", reads.Load())
	}
	if _, err := p.store.Load(); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatal("borrowed credentials entered direct store")
	}
	data, err := os.ReadFile(filepath.Join(dir, codexReferenceName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "borrowed-access") || strings.Contains(string(data), "token") {
		t.Fatal("source reference contains credential data")
	}
	info, _ := os.Stat(filepath.Join(dir, codexReferenceName))
	if info.Mode().Perm() != 0600 {
		t.Fatalf("reference mode=%o", info.Mode().Perm())
	}
}

func TestProviderCoalescesConcurrentNativeReads(t *testing.T) {
	p, _ := providerFixture(t)
	var reads atomic.Int32
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads.Add(1)
		time.Sleep(25 * time.Millisecond)
		return borrowedFixture("access", "user", "account"), nil
	}
	var wg sync.WaitGroup
	errors := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := p.EnsureFresh(context.Background(), 0); errors <- err }()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if reads.Load() != 1 {
		t.Fatalf("native reads=%d", reads.Load())
	}
}

func TestProviderCoalescesConcurrentNativeFailures(t *testing.T) {
	p, _ := providerFixture(t)
	var reads atomic.Int32
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads.Add(1)
		time.Sleep(20 * time.Millisecond)
		return File{}, errors.New("native unavailable")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Load(context.Background()); err == nil {
				t.Error("native failure accepted")
			}
		}()
	}
	wg.Wait()
	if reads.Load() != 1 {
		t.Fatalf("concurrent failed native reads=%d", reads.Load())
	}
}

func TestProviderRefreshReReadsAndUsesAlreadyRotatedToken(t *testing.T) {
	p, _ := providerFixture(t)
	var forced atomic.Int32
	p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
		if refresh {
			forced.Add(1)
		}
		return borrowedFixture("already-rotated", "user", "account"), nil
	}
	file, err := p.Refresh(context.Background(), "rejected")
	if err != nil || file.Tokens.AccessToken != "already-rotated" || forced.Load() != 0 {
		t.Fatalf("forced=%d, err=%v", forced.Load(), err)
	}
}

func TestProviderRefreshDelegatesRotationAndRejectsUnchanged(t *testing.T) {
	for _, unchanged := range []bool{false, true} {
		t.Run(map[bool]string{false: "rotated", true: "unchanged"}[unchanged], func(t *testing.T) {
			p, _ := providerFixture(t)
			var forced bool
			p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
				token := "rejected"
				if refresh {
					forced = true
					if !unchanged {
						token = "rotated"
					}
				}
				return borrowedFixture(token, "user", "account"), nil
			}
			file, err := p.Refresh(context.Background(), "rejected")
			if !forced {
				t.Fatal("native refresh not requested")
			}
			if unchanged {
				if err == nil {
					t.Fatal("unchanged rejected token accepted")
				}
				return
			}
			if err != nil || file.Tokens.AccessToken != "rotated" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestProviderRefreshPinsIdentityBeforeAndAfter(t *testing.T) {
	p, _ := providerFixture(t)
	p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
		if refresh {
			return borrowedFixture("new", "other-user", "other-account"), nil
		}
		return borrowedFixture("old", "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Refresh(context.Background(), "old"); !errors.Is(err, ErrCodexIdentityChanged) {
		t.Fatalf("err=%v", err)
	}
}

func TestProviderReferenceRejectsSourceAndAccountChangesAndExplicitlyAdopts(t *testing.T) {
	p, home := providerFixture(t)
	account := "account"
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		return borrowedFixture("access", "user", account), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	account = "new-account"
	if _, err := p.Refresh(context.Background(), "old"); !errors.Is(err, ErrCodexIdentityChanged) {
		t.Fatalf("account switch err=%v", err)
	}
	if _, err := p.UseCodex(context.Background()); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(home, "other-codex")
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", other)
	if _, err := p.Load(context.Background()); !errors.Is(err, ErrCodexIdentityChanged) {
		t.Fatalf("source switch err=%v", err)
	}
}

func TestProviderLogoutSuppressesReuseWithoutTouchingNativeStore(t *testing.T) {
	p, home := providerFixture(t)
	nativePath := filepath.Join(home, ".codex", "auth.json")
	if err := os.WriteFile(nativePath, []byte("native authority fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		return borrowedFixture("access", "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if err := p.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.EnsureFresh(context.Background(), 0); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("ambient reuse after logout err=%v", err)
	}
	data, err := os.ReadFile(nativePath)
	if err != nil || string(data) != "native authority fixture" {
		t.Fatal("native credentials changed")
	}
	if _, err := p.UseCodex(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestProviderDirectTransientFailureDoesNotSwitchSources(t *testing.T) {
	for _, permanent := range []bool{false, true} {
		t.Run(map[bool]string{false: "transient", true: "permanent"}[permanent], func(t *testing.T) {
			p, _ := providerFixture(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if permanent {
					w.WriteHeader(400)
					_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				} else {
					w.WriteHeader(503)
				}
			}))
			defer server.Close()
			// Keep the provider official for borrowing; inject only its direct refresher transport.
			p.client = &http.Client{Transport: rewriteAuthTransport{target: server.URL, base: server.Client().Transport}}
			if err := p.store.Save(File{Tokens: Tokens{AccessToken: "expired", RefreshToken: "refresh", AccessTokenExpiresAt: 1}}); err != nil {
				t.Fatal(err)
			}
			var native bool
			p.nativeRead = func(context.Context, string, bool) (File, error) {
				native = true
				return borrowedFixture("access", "user", "account"), nil
			}
			_, err := p.EnsureFresh(context.Background(), 0)
			if permanent {
				if err != nil || !native {
					t.Fatalf("permanent fallback native=%v err=%v", native, err)
				}
			} else if err == nil || native {
				t.Fatalf("transient fallback native=%v err=%v", native, err)
			}
		})
	}
}

type rewriteAuthTransport struct {
	target string
	base   http.RoundTripper
}

func (r rewriteAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	parsed, _ := http.NewRequest(req.Method, r.target, nil)
	copy.URL = parsed.URL
	return r.base.RoundTrip(copy)
}

func TestProviderBorrowedFailureDoesNotSwitchToBrowserAndRejectsCustomEndpoint(t *testing.T) {
	p, _ := providerFixture(t)
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		return borrowedFixture("access", "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		return File{}, errors.New("temporary native outage")
	}
	if _, err := p.Refresh(context.Background(), "access"); err == nil || shouldLoginForAuthError(err) {
		t.Fatalf("existing native failure should not trigger browser: %v", err)
	}
	p.endpoint = "https://example.invalid/token"
	if _, err := p.Refresh(context.Background(), "access"); err == nil {
		t.Fatal("custom borrowed OAuth endpoint accepted")
	}
}

func TestProviderNativeLockCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock")
	release, err := acquireNamedAuthLock(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := acquireNamedAuthLock(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock err=%v", err)
	}
}

func TestProviderReferenceContainsOnlyAllowedFields(t *testing.T) {
	ref := codexReference{Version: 1, Home: "/fixture/codex", UserID: "user", AccountID: "account"}
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("reference fields=%d", len(fields))
	}
}

func TestProviderInvalidDirectLoginDoesNotShadowBorrowedSource(t *testing.T) {
	p, _ := providerFixture(t)
	if err := p.store.Save(File{Tokens: Tokens{AccessToken: "incomplete-direct"}}); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		return borrowedFixture("borrowed", "user", "account"), nil
	}
	for _, read := range []func(context.Context) (File, error){p.Load, func(ctx context.Context) (File, error) { return p.EnsureFresh(ctx, 0) }, func(ctx context.Context) (File, error) { return p.Refresh(ctx, "old") }} {
		file, err := read(context.Background())
		if err != nil || file.Source != SourceCodex {
			t.Fatalf("source=%s err=%v", file.Source, err)
		}
	}
}

func TestProviderExpiredBorrowedRefreshOutageBlocksWithoutFallback(t *testing.T) {
	p, _ := providerFixture(t)
	forced := false
	p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
		if refresh {
			forced = true
			return File{}, errors.New("temporary native refresh outage")
		}
		file := borrowedFixture("expired", "user", "account")
		file.Tokens.AccessTokenExpiresAt = 1
		return file, nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err == nil || shouldLoginForAuthError(err) {
		t.Fatalf("expired refresh outage err=%v", err)
	}
	if !forced {
		t.Fatal("expired native access was not refreshed")
	}
}

func TestProviderFreshBorrowedTokenDoesNotForceNativeRefresh(t *testing.T) {
	p, _ := providerFixture(t)
	p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
		if refresh {
			t.Fatal("fresh startup token forced refresh")
		}
		return borrowedFixture("fresh", "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 5*time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestProviderForcedRefreshIgnoresTwoSecondMemoryResult(t *testing.T) {
	p, _ := providerFixture(t)
	reads := 0
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads++
		token := "first"
		if reads > 1 {
			token = "rotated"
		}
		return borrowedFixture(token, "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	file, err := p.Refresh(context.Background(), "first")
	if err != nil || reads != 2 || file.Tokens.AccessToken != "rotated" {
		t.Fatalf("reads=%d err=%v", reads, err)
	}
}

func TestProviderHotReadsNeverExtendNativeObservationTime(t *testing.T) {
	p, _ := providerFixture(t)
	reads := 0
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads++
		account := "account"
		if reads > 1 {
			account = "changed-account"
		}
		return borrowedFixture("access", "user", account), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	home, err := p.codexHome()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := dataDir(p.store.home)
	if err != nil {
		t.Fatal(err)
	}
	key := home + "\x00" + dir
	// Make elapsed age deterministic without waiting two wall-clock seconds.
	nativeAuthority.guard <- struct{}{}
	entry := nativeAuthority.cache[key]
	entry.at = time.Now().Add(-time.Second)
	observedAt := entry.at
	nativeAuthority.cache[key] = entry
	<-nativeAuthority.guard
	for i := 0; i < 8; i++ {
		if _, err := p.Load(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	nativeAuthority.guard <- struct{}{}
	entry = nativeAuthority.cache[key]
	if !entry.at.Equal(observedAt) {
		<-nativeAuthority.guard
		t.Fatal("cached hits extended the native observation timestamp")
	}
	entry.at = time.Now().Add(-nativeCacheTTL - time.Millisecond)
	nativeAuthority.cache[key] = entry
	<-nativeAuthority.guard
	if _, err := p.Load(context.Background()); !errors.Is(err, ErrCodexIdentityChanged) {
		t.Fatalf("expired hot cache hid changed identity: %v", err)
	}
	if reads != 2 {
		t.Fatalf("native reads=%d", reads)
	}
}

func TestProviderRejectedTokenFailuresNeverRestoreCachedSuccess(t *testing.T) {
	for _, mode := range []string{"account-change", "native-read-failure", "forced-refresh-failure", "unchanged-rejected-token", "expired-refresh-result"} {
		t.Run(mode, func(t *testing.T) {
			p, _ := providerFixture(t)
			reads := 0
			failure := false
			p.nativeRead = func(_ context.Context, _ string, refresh bool) (File, error) {
				reads++
				if failure {
					switch mode {
					case "account-change":
						return borrowedFixture("new-account-access", "user", "changed-account"), nil
					case "native-read-failure":
						return File{}, errors.New("native authority unavailable")
					case "forced-refresh-failure":
						if refresh {
							return File{}, errors.New("native refresh unavailable")
						}
					case "expired-refresh-result":
						if refresh {
							file := borrowedFixture("expired-result", "user", "account")
							file.Tokens.AccessTokenExpiresAt = 1
							return file, nil
						}
					}
				}
				return borrowedFixture("rejected-access", "user", "account"), nil
			}
			if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
				t.Fatal(err)
			}
			failure = true
			if _, err := p.Refresh(context.Background(), "rejected-access"); err == nil {
				t.Fatal("rejected-token failure accepted")
			}
			readsAfterFailure := reads
			file, err := p.EnsureFresh(context.Background(), 0)
			if err == nil || file.Tokens.AccessToken != "" {
				t.Fatal("next request resurrected pre-401 cached credentials")
			}
			if reads != readsAfterFailure {
				t.Fatal("bounded failed-auth result was not coalesced")
			}
		})
	}
}

func TestProviderCanceledBypassReadEvictsOldSuccess(t *testing.T) {
	p, _ := providerFixture(t)
	reads := 0
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads++
		return borrowedFixture("old-access", "user", "account"), nil
	}
	if _, err := p.EnsureFresh(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) { reads++; return File{}, context.Canceled }
	if _, err := p.Refresh(context.Background(), "old-access"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read error=%v", err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		reads++
		return borrowedFixture("rotated-access", "user", "account"), nil
	}
	file, err := p.EnsureFresh(context.Background(), 0)
	if err != nil || file.Tokens.AccessToken != "rotated-access" || reads != 3 {
		t.Fatalf("canceled read retained stale success: reads=%d err=%v", reads, err)
	}
}

func TestProviderExplicitCodexLoginRequiresDirectLogout(t *testing.T) {
	p, _ := providerFixture(t)
	if err := p.store.Save(File{Tokens: Tokens{AccessToken: "own", RefreshToken: "own-refresh"}}); err != nil {
		t.Fatal(err)
	}
	p.nativeRead = func(context.Context, string, bool) (File, error) {
		t.Fatal("read native while direct login active")
		return File{}, nil
	}
	if _, err := p.UseCodex(context.Background()); err == nil || !strings.Contains(err.Error(), "logout") {
		t.Fatalf("explicit adoption err=%v", err)
	}
}

func TestProviderCodexHomeUsesSuppliedHomeAndCanonicalOverride(t *testing.T) {
	p, home := providerFixture(t)
	actual, err := p.codexHome()
	expected, _ := filepath.EvalSymlinks(filepath.Join(home, ".codex"))
	if err != nil || actual != expected {
		t.Fatalf("home=%s err=%v", actual, err)
	}
	other := t.TempDir()
	alias := filepath.Join(home, "codex-alias")
	if err := os.Symlink(other, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", alias)
	actual, err = p.codexHome()
	expected, _ = filepath.EvalSymlinks(other)
	if err != nil || actual != expected {
		t.Fatalf("canonical home=%s err=%v", actual, err)
	}
}

func TestProviderDirectLogoutAlsoSuppressesAmbientReuse(t *testing.T) {
	p, home := providerFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer server.Close()
	p.client = &http.Client{Transport: rewriteAuthTransport{target: server.URL, base: server.Client().Transport}}
	if err := p.store.Save(File{Tokens: Tokens{AccessToken: "own", RefreshToken: "own-refresh"}}); err != nil {
		t.Fatal(err)
	}
	if err := p.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claudodex", codexSuppressionName)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.EnsureFresh(context.Background(), 0); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("direct logout ambient reuse err=%v", err)
	}
	if err := clearCodexReuseState(p.store); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claudodex", codexSuppressionName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("successful direct-login cleanup did not clear suppression")
	}
}

func TestProviderReferenceRejectsUnsafePathsAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reference")
	for _, content := range []string{`{"version":1,"home":"/fixture","user_id":"user","account_id":"account","token":"forbidden"}`, `{"version":1,"home":"relative","user_id":"user","account_id":"account"}`, `{"version":1,"home":"/fixture","user_id":"user","account_id":"account"} {}`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readCodexReference(path); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := readCodexReference(path); err == nil {
		t.Fatal("symlink reference accepted")
	}
}
