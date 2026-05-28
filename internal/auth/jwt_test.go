package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseClaims(t *testing.T) {
	token := fakeJWT(t, map[string]any{
		"exp": 1770000000,
		"https://api.openai.com/profile": map[string]any{
			"email": "pat@example.com",
		},
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_plan_type":          "pro",
			"chatgpt_user_id":            "user-123",
			"chatgpt_account_id":         "account-123",
			"chatgpt_account_is_fedramp": true,
		},
	})
	claims, err := ParseClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "pat@example.com" || claims.PlanType != "pro" || claims.UserID != "user-123" || claims.AccountID != "account-123" || !claims.FedRAMP || claims.Exp != 1770000000 {
		t.Fatalf("claims = %#v", claims)
	}
}

func fakeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}
