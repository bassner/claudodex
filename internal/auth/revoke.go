package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const RevokeEndpoint = "https://auth.openai.com/oauth/revoke"

var ErrRevokeFailed = errors.New("revoke failed")

func RevokeAndDelete(ctx context.Context, store Store, client *http.Client, endpoint string) error {
	file, loadErr := store.Load()
	if client == nil {
		client = http.DefaultClient
	}
	if endpoint == "" {
		endpoint = RevokeEndpoint
	}

	var revokeErr error
	if loadErr == nil {
		token := file.Tokens.RefreshToken
		hint := "refresh_token"
		if token == "" {
			token = file.Tokens.AccessToken
			hint = "access_token"
		}
		if token != "" {
			revokeErr = revokeToken(ctx, client, endpoint, token, hint)
		}
	}

	if err := store.Delete(); err != nil {
		return err
	}
	if loadErr != nil && loadErr != ErrNotLoggedIn {
		return loadErr
	}
	if revokeErr != nil {
		return fmt.Errorf("%w: %v", ErrRevokeFailed, revokeErr)
	}
	return nil
}

func revokeToken(ctx context.Context, client *http.Client, endpoint, token, hint string) error {
	body := map[string]string{
		"token":           token,
		"token_type_hint": hint,
		"client_id":       ClientID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("revoke failed: HTTP %d", resp.StatusCode)
	}
	return nil
}
