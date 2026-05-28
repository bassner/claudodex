package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const accessTokenRefreshSkew = 5 * time.Minute

var ErrInvalidAuth = errors.New("invalid auth")
var defaultTokenEndpoint = TokenEndpoint

func EnsureLoggedIn(ctx context.Context, home string, interactive bool) (File, error) {
	store := NewStore(home)
	file, err := ensureStoredAuth(ctx, store)
	if err == nil {
		return file, nil
	}
	if !shouldLoginForAuthError(err) {
		return File{}, err
	}
	if !interactive {
		return File{}, fmt.Errorf("login required; run claudodex clx:auth-login (%w)", err)
	}
	file, err = Login(ctx, LoginOptions{Home: home})
	if err != nil {
		return File{}, fmt.Errorf("login required and OAuth login failed: %w", err)
	}
	return file, nil
}

func ensureStoredAuth(ctx context.Context, store Store) (File, error) {
	file, err := store.Load()
	if err != nil {
		return File{}, err
	}
	if file.Tokens.RefreshToken == "" {
		return File{}, fmt.Errorf("%w: missing refresh_token", ErrInvalidAuth)
	}
	if file.Tokens.AccessToken == "" {
		return NewRefresher(store, nil, defaultTokenEndpoint).Refresh(ctx)
	}
	ApplyClaims(&file)
	if file.Tokens.AccessTokenExpiresAt == 0 {
		return file, nil
	}
	expiresAt := time.Unix(file.Tokens.AccessTokenExpiresAt, 0)
	if time.Until(expiresAt) > accessTokenRefreshSkew {
		return file, nil
	}
	return NewRefresher(store, nil, defaultTokenEndpoint).Refresh(ctx)
}

func shouldLoginForAuthError(err error) bool {
	return errors.Is(err, ErrNotLoggedIn) ||
		errors.Is(err, ErrInvalidAuth) ||
		errors.Is(err, ErrPermanentRefreshFailure)
}
