package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Refresher struct {
	store    Store
	client   *http.Client
	endpoint string
}

var refreshMu sync.Mutex

var ErrPermanentRefreshFailure = errors.New("permanent refresh failure")

var refreshLockRetry = 50 * time.Millisecond

const refreshLockStaleAfter = 5 * time.Minute

type tokenEndpointError struct {
	Error            json.RawMessage `json:"error"`
	ErrorDescription string          `json:"error_description"`
}

func NewRefresher(store Store, client *http.Client, endpoint string) *Refresher {
	if client == nil {
		client = http.DefaultClient
	}
	if endpoint == "" {
		endpoint = TokenEndpoint
	}
	return &Refresher{store: store, client: client, endpoint: endpoint}
}

func (r *Refresher) Refresh(ctx context.Context) (File, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	refreshMu.Lock()
	defer refreshMu.Unlock()

	release, err := acquireRefreshFileLock(ctx, r.store)
	if err != nil {
		return File{}, err
	}
	defer release()

	file, err := r.store.Load()
	if err != nil {
		return File{}, err
	}
	if file.Tokens.RefreshToken == "" {
		return File{}, fmt.Errorf("auth file missing refresh_token")
	}
	refreshToken := file.Tokens.RefreshToken

	reqBody := map[string]string{
		"client_id":     ClientID,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return File{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(data))
	if err != nil {
		return File{}, err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return File{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var body tokenEndpointError
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if code := body.permanentCode(); code != "" {
			if current, loadErr := r.store.Load(); loadErr == nil && current.Tokens.RefreshToken != "" && current.Tokens.RefreshToken != refreshToken {
				return current, nil
			}
			_ = r.store.Delete()
			return File{}, fmt.Errorf("%w: %s", ErrPermanentRefreshFailure, code)
		}
		return File{}, fmt.Errorf("refresh failed: HTTP %d", resp.StatusCode)
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return File{}, err
	}
	if tokens.AccessToken == "" {
		return File{}, fmt.Errorf("refresh response missing access_token")
	}

	file.Tokens.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		file.Tokens.RefreshToken = tokens.RefreshToken
	}
	if tokens.IDToken != "" {
		file.Tokens.IDToken = tokens.IDToken
	}
	file.LastRefresh = time.Now().UTC()
	ApplyClaims(&file)
	if err := r.store.Save(file); err != nil {
		return File{}, err
	}
	return file, nil
}

func acquireRefreshFileLock(ctx context.Context, store Store) (func(), error) {
	path, err := store.path()
	if err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, err
	}
	for {
		file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(file, "pid=%d created_at=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
			_ = file.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if refreshLockIsStale(lockPath, time.Now()) {
			_ = os.Remove(lockPath)
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

func refreshLockIsStale(path string, now time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return now.Sub(info.ModTime()) > refreshLockStaleAfter
}

func (r *Refresher) EnsureFresh(ctx context.Context, skew time.Duration) (File, error) {
	file, err := r.store.Load()
	if err != nil {
		return File{}, err
	}
	if file.Tokens.AccessTokenExpiresAt == 0 {
		return file, nil
	}
	expiresAt := time.Unix(file.Tokens.AccessTokenExpiresAt, 0)
	if time.Until(expiresAt) > skew {
		return file, nil
	}
	return r.Refresh(ctx)
}

func (e tokenEndpointError) permanentCode() string {
	if len(e.Error) > 0 {
		var nested struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(e.Error, &nested); err == nil && isPermanentRefreshCode(nested.Code) {
			return nested.Code
		}
		var flat string
		if err := json.Unmarshal(e.Error, &flat); err == nil && isPermanentRefreshCode(flat) {
			return flat
		}
	}
	if code := permanentRefreshCodeInText(e.ErrorDescription); code != "" {
		return code
	}
	return ""
}

func isPermanentRefreshCode(code string) bool {
	switch strings.ToLower(code) {
	case "invalid_grant", "refresh_token_expired", "refresh_token_reused", "refresh_token_invalidated":
		return true
	default:
		return false
	}
}

func permanentRefreshCodeInText(text string) string {
	text = strings.ToLower(text)
	for _, code := range []string{"invalid_grant", "refresh_token_expired", "refresh_token_reused", "refresh_token_invalidated"} {
		if strings.Contains(text, code) {
			return code
		}
	}
	return ""
}
