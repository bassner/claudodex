package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	SourceDirect         = "direct"
	SourceCodex          = "codex"
	CodexAllowedOrigin   = "https://chatgpt.com"
	codexReferenceName   = "codex-login.json"
	codexSuppressionName = "codex-reuse-disabled"
	nativeCacheTTL       = 2 * time.Second
)

var ErrCodexIdentityChanged = errors.New("local Codex login changed; run claudodex clx:auth-login --codex to adopt the current login")

// Provider keeps direct OAuth and borrowed native Codex authority separate.
type Provider struct {
	store      Store
	client     *http.Client
	endpoint   string
	nativeRead func(context.Context, string, bool) (File, error)
	mu         sync.Mutex
	pinned     *codexReference
}

type codexReference struct {
	Version   int    `json:"version"`
	Home      string `json:"home"`
	UserID    string `json:"user_id"`
	AccountID string `json:"account_id"`
}

type nativeCacheEntry struct {
	file File
	at   time.Time
	err  error
}

var nativeAuthority = struct {
	guard chan struct{}
	cache map[string]nativeCacheEntry
}{guard: make(chan struct{}, 1), cache: make(map[string]nativeCacheEntry)}

func NewProvider(store Store, client *http.Client, tokenEndpoint string) *Provider {
	if tokenEndpoint == "" {
		tokenEndpoint = TokenEndpoint
	}
	return &Provider{store: store, client: client, endpoint: tokenEndpoint, nativeRead: readNativeCodex}
}

func (p *Provider) Load(ctx context.Context) (File, error) {
	file, err := p.loadDirect()
	if err == nil {
		ApplyClaims(&file)
		file.Source = SourceDirect
		return file, nil
	}
	if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	return p.borrow(ctx, false, "", false)
}

func (p *Provider) EnsureFresh(ctx context.Context, skew time.Duration) (File, error) {
	file, err := p.loadDirect()
	if err == nil {
		if file.Tokens.RefreshToken == "" {
			err = fmt.Errorf("%w: missing refresh_token", ErrInvalidAuth)
		} else {
			ApplyClaims(&file)
			if file.Tokens.AccessToken == "" || (file.Tokens.AccessTokenExpiresAt != 0 && time.Until(time.Unix(file.Tokens.AccessTokenExpiresAt, 0)) <= skew) {
				file, err = NewRefresher(p.store, p.client, p.endpoint).Refresh(ctx)
			}
			if err == nil {
				file.Source = SourceDirect
				return file, nil
			}
		}
	}
	if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	borrowed, borrowErr := p.borrow(ctx, true, "", false)
	if borrowErr == nil {
		if borrowed.Tokens.AccessTokenExpiresAt <= time.Now().Add(skew).Unix() {
			return p.borrow(ctx, true, borrowed.Tokens.AccessToken, true)
		}
		return borrowed, nil
	}
	if errors.Is(borrowErr, ErrNotLoggedIn) && !errors.Is(err, ErrNotLoggedIn) {
		return File{}, err
	}
	return File{}, borrowErr
}

// Refresh is forced only if the rejected access token is still authoritative.
func (p *Provider) Refresh(ctx context.Context, rejectedAccessToken string) (File, error) {
	file, err := p.loadDirect()
	if err == nil {
		if rejectedAccessToken != "" && file.Tokens.AccessToken != rejectedAccessToken {
			ApplyClaims(&file)
			if file.Tokens.AccessToken != "" && (file.Tokens.AccessTokenExpiresAt == 0 || file.Tokens.AccessTokenExpiresAt > time.Now().Unix()) {
				file.Source = SourceDirect
				return file, nil
			}
		}
		file, err = NewRefresher(p.store, p.client, p.endpoint).refresh(ctx, rejectedAccessToken)
		if err == nil {
			file.Source = SourceDirect
			return file, nil
		}
		if !shouldLoginForAuthError(err) {
			return File{}, err
		}
	}
	if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	return p.borrow(ctx, true, rejectedAccessToken, true)
}

func (p *Provider) loadDirect() (File, error) {
	file, err := p.store.Load()
	if err != nil {
		return File{}, err
	}
	if file.Tokens.RefreshToken == "" {
		return File{}, fmt.Errorf("%w: missing refresh_token", ErrInvalidAuth)
	}
	return file, nil
}

// EnsureLoggedIn tries the shared provider before the ordinary direct browser
// flow. An established borrowed source's transient failure never opens a browser.
func (p *Provider) EnsureLoggedIn(ctx context.Context, interactive bool) (File, error) {
	file, err := p.EnsureFresh(ctx, accessTokenRefreshSkew)
	if err == nil {
		return file, nil
	}
	if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	if !interactive {
		return File{}, fmt.Errorf("login required; run claudodex clx:auth-login (%w)", err)
	}
	file, err = Login(ctx, LoginOptions{Home: p.store.home, HTTPClient: p.client, TokenEndpoint: p.endpoint})
	if err != nil {
		return File{}, fmt.Errorf("login required and OAuth login failed: %w", err)
	}
	return file, nil
}

func (p *Provider) Logout(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	dir, err := dataDir(p.store.home)
	if err != nil {
		return err
	}
	select {
	case nativeAuthority.guard <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-nativeAuthority.guard }()
	release, err := acquireNamedAuthLock(ctx, filepath.Join(dir, "codex-authority.lock"))
	if err != nil {
		return err
	}
	defer release()
	if err := writePrivateJSON(filepath.Join(dir, codexSuppressionName), map[string]bool{"disabled": true}); err != nil {
		return err
	}
	if err := removeIfExists(filepath.Join(dir, codexReferenceName)); err != nil {
		return err
	}
	p.mu.Lock()
	p.pinned = nil
	p.mu.Unlock()
	// Only the direct store is ever revoked. No Codex logout method is called.
	refreshMu.Lock()
	defer refreshMu.Unlock()
	releaseDirect, err := acquireRefreshFileLock(ctx, p.store)
	if err != nil {
		return err
	}
	defer releaseDirect()
	return RevokeAndDelete(ctx, p.store, p.client, "")
}

func (p *Provider) UseCodex(ctx context.Context) (File, error) {
	if _, err := p.loadDirect(); err == nil {
		return File{}, fmt.Errorf("a direct Claudodex login is active; run claudodex clx:auth-logout first")
	} else if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	return p.borrow(ctx, true, "", true, true)
}

func (p *Provider) borrow(ctx context.Context, persist bool, rejected string, bypass bool, adopt ...bool) (File, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	explicit := len(adopt) > 0 && adopt[0]
	dir, err := dataDir(p.store.home)
	if err != nil {
		return File{}, err
	}
	ref, refErr := readCodexReference(filepath.Join(dir, codexReferenceName))
	if refErr != nil && !errors.Is(refErr, os.ErrNotExist) {
		return File{}, refErr
	}
	existing := refErr == nil
	if !explicit {
		if _, err := os.Lstat(filepath.Join(dir, codexSuppressionName)); err == nil {
			return File{}, ErrNotLoggedIn
		} else if !errors.Is(err, os.ErrNotExist) {
			return File{}, err
		}
	}
	if p.endpoint != TokenEndpoint {
		return File{}, codexCandidateError(existing && !explicit, fmt.Errorf("local Codex login requires the official OAuth endpoint"))
	}
	home, err := p.codexHome()
	if err != nil {
		return File{}, codexCandidateError(existing && !explicit, err)
	}
	if existing && !explicit && ref.Home != home {
		return File{}, ErrCodexIdentityChanged
	}
	select {
	case nativeAuthority.guard <- struct{}{}:
	case <-ctx.Done():
		return File{}, ctx.Err()
	}
	defer func() { <-nativeAuthority.guard }()
	// Even native reads can refresh. Both reads and explicit refresh therefore
	// use the same Claudodex-owned lock, never a file under CODEX_HOME.
	release, err := acquireNamedAuthLock(ctx, filepath.Join(dir, "codex-authority.lock"))
	if err != nil {
		return File{}, err
	}
	defer release()
	// Logout or another process can change state while this request waits.
	if !explicit {
		if _, err := os.Lstat(filepath.Join(dir, codexSuppressionName)); err == nil {
			return File{}, ErrNotLoggedIn
		} else if !errors.Is(err, os.ErrNotExist) {
			return File{}, err
		}
		ref, refErr = readCodexReference(filepath.Join(dir, codexReferenceName))
		if refErr != nil && !errors.Is(refErr, os.ErrNotExist) {
			return File{}, refErr
		}
		existing = refErr == nil
		if existing && ref.Home != home {
			return File{}, ErrCodexIdentityChanged
		}
	}
	if explicit {
		p.mu.Lock()
		p.pinned = nil
		p.mu.Unlock()
	}
	cacheKey := home + "\x00" + dir
	entry, cached := nativeAuthority.cache[cacheKey]
	cacheFailure := func(err error) {
		// A rejected/expired token must not become a cached success again when
		// its forced refresh fails. Do not propagate one caller's cancellation
		// to other callers, but keep a bounded failed-auth result for them.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("local Codex authentication could not complete; retry the request")
		}
		nativeAuthority.cache[cacheKey] = nativeCacheEntry{at: time.Now(), err: err}
	}
	file := entry.file
	observedAt := entry.at
	if !cached || bypass || time.Since(entry.at) < 0 || time.Since(entry.at) >= nativeCacheTTL {
		// An uncached observation supersedes the previous result even if it
		// fails, detects an account change, or is canceled partway through.
		delete(nativeAuthority.cache, cacheKey)
		file, err = p.nativeRead(ctx, home, false)
		if err != nil {
			// Concurrent normal reads share bounded failures as well as success.
			// A caller's cancellation must never poison another request's result.
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				nativeAuthority.cache[cacheKey] = nativeCacheEntry{at: time.Now(), err: err}
			}
			return File{}, codexCandidateError(existing && !explicit, err)
		}
		observedAt = time.Now()
	} else if entry.err != nil {
		return File{}, codexCandidateError(existing && !explicit, entry.err)
	}
	if err := p.checkIdentity(file, home, ref, existing && !explicit); err != nil {
		cacheFailure(err)
		return File{}, err
	}
	if (rejected != "" && file.Tokens.AccessToken == rejected) || (persist && file.Tokens.AccessTokenExpiresAt <= time.Now().Unix()) {
		beforeUser, beforeAccount := file.Tokens.UserID, file.Tokens.AccountID
		delete(nativeAuthority.cache, cacheKey)
		file, err = p.nativeRead(ctx, home, true)
		if err != nil {
			err = fmt.Errorf("local Codex token refresh failed: %w", err)
			cacheFailure(err)
			return File{}, err
		}
		observedAt = time.Now()
		if err := p.checkIdentity(file, home, ref, existing && !explicit); err != nil {
			cacheFailure(err)
			return File{}, err
		}
		if file.Tokens.UserID != beforeUser || file.Tokens.AccountID != beforeAccount {
			cacheFailure(ErrCodexIdentityChanged)
			return File{}, ErrCodexIdentityChanged
		}
		if file.Tokens.AccessToken == rejected {
			err = fmt.Errorf("local Codex did not replace the rejected access token; sign in with Codex again")
			cacheFailure(err)
			return File{}, err
		}
	}
	if file.Tokens.AccessTokenExpiresAt <= time.Now().Unix() {
		err = fmt.Errorf("local Codex returned an expired access token; sign in with Codex again")
		cacheFailure(err)
		return File{}, err
	}
	file.Source, file.AllowedOrigin = SourceCodex, CodexAllowedOrigin
	nativeAuthority.cache[cacheKey] = nativeCacheEntry{file: file, at: observedAt}
	if persist {
		newRef := codexReference{Version: 1, Home: home, UserID: file.Tokens.UserID, AccountID: file.Tokens.AccountID}
		if err := writePrivateJSON(filepath.Join(dir, codexReferenceName), newRef); err != nil {
			return File{}, err
		}
		if explicit {
			if err := removeIfExists(filepath.Join(dir, codexSuppressionName)); err != nil {
				return File{}, err
			}
		}
		p.mu.Lock()
		p.pinned = &newRef
		p.mu.Unlock()
	}
	return file, nil
}

func (p *Provider) checkIdentity(file File, home string, ref codexReference, existing bool) error {
	if file.Tokens.UserID == "" || file.Tokens.AccountID == "" {
		return fmt.Errorf("local Codex did not provide a usable subscription identity")
	}
	if existing && (ref.Home != home || ref.UserID != file.Tokens.UserID || ref.AccountID != file.Tokens.AccountID) {
		return ErrCodexIdentityChanged
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pinned != nil && (p.pinned.Home != home || p.pinned.UserID != file.Tokens.UserID || p.pinned.AccountID != file.Tokens.AccountID) {
		return ErrCodexIdentityChanged
	}
	return nil
}

func codexCandidateError(existing bool, err error) error {
	if existing || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: no eligible local Codex login (%v)", ErrNotLoggedIn, err)
}

func (p *Provider) codexHome() (string, error) {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		userHome := p.store.home
		if userHome == "" {
			var err error
			userHome, err = os.UserHomeDir()
			if err != nil {
				return "", err
			}
		}
		home = filepath.Join(userHome, ".codex")
	}
	absolute, err := filepath.Abs(home)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("Codex home is unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("Codex home is not a directory")
	}
	return resolved, nil
}

func clearCodexReuseState(store Store) error {
	dir, err := dataDir(store.home)
	if err != nil {
		return err
	}
	for _, name := range []string{codexReferenceName, codexSuppressionName} {
		if err := removeIfExists(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func readCodexReference(path string) (codexReference, error) {
	var ref codexReference
	file, err := openPrivateRegular(path, 16<<10)
	if err != nil {
		return ref, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return ref, fmt.Errorf("invalid local Codex login reference")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return ref, fmt.Errorf("invalid local Codex login reference")
	}
	if ref.Version != 1 || !filepath.IsAbs(ref.Home) || ref.AccountID == "" || ref.UserID == "" {
		return ref, fmt.Errorf("invalid local Codex login reference")
	}
	return ref, nil
}

func openPrivateRegular(path string, max int64) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("unsafe or oversized authentication metadata file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Size() > max || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		file.Close()
		return nil, fmt.Errorf("authentication metadata changed while opening")
	}
	return file, nil
}

func writePrivateJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe authentication state path")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".codex-login-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func acquireNamedAuthLock(ctx context.Context, lockPath string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, err
	}
	for {
		file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			file.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if refreshLockIsStale(lockPath, time.Now()) {
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			continue
		}
		timer := time.NewTimer(refreshLockRetry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
