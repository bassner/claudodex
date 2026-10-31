package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func nativeJWT(t *testing.T, user, account string, fedramp *bool) string {
	t.Helper()
	auth := map[string]any{"chatgpt_user_id": user, "chatgpt_account_id": account, "chatgpt_plan_type": "free"}
	if fedramp != nil {
		auth["chatgpt_account_is_fedramp"] = *fedramp
	}
	return fakeJWT(t, map[string]any{"iss": Issuer, "exp": time.Now().Add(time.Hour).Unix(), "email": "fixture@example.invalid", "https://api.openai.com/auth": auth})
}

func writeNativeMetadata(t *testing.T, path, id string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"auth_mode": "chatgpt", "tokens": map[string]any{"id_token": id, "account_id": "account", "refresh_token": "must-never-be-copied"}})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNativeResidencyUsesOnlyBoundFileAuthority(t *testing.T) {
	for _, fedramp := range []bool{false, true} {
		t.Run(fmt.Sprint(fedramp), func(t *testing.T) {
			home := t.TempDir()
			access, _ := validatedNativeClaims(nativeJWT(t, "user", "account", nil))
			writeNativeMetadata(t, filepath.Join(home, "auth.json"), nativeJWT(t, "user", "account", &fedramp))
			actual, err := readNativeResidency(context.Background(), home, access)
			if err != nil || actual != fedramp {
				t.Fatalf("fedramp=%v err=%v", actual, err)
			}
		})
	}
	home := t.TempDir()
	access, _ := validatedNativeClaims(nativeJWT(t, "user", "account", nil))
	writeNativeMetadata(t, filepath.Join(home, "auth.json"), nativeJWT(t, "wrong-user", "account", nil))
	if _, err := readNativeResidency(context.Background(), home, access); err == nil {
		t.Fatal("unbound ID metadata accepted")
	}
	writeNativeMetadata(t, filepath.Join(home, "auth.json"), nativeJWT(t, "user", "account", nil))
	if actual, err := readNativeResidency(context.Background(), home, access); err != nil || actual {
		t.Fatalf("native optional-false interpretation=%v, err=%v", actual, err)
	}
}

func TestNativeClaimsRejectUnknownIssuerAndMissingIdentity(t *testing.T) {
	for _, claims := range []map[string]any{
		{"iss": "https://example.invalid", "exp": time.Now().Add(time.Hour).Unix(), "https://api.openai.com/auth": map[string]any{"chatgpt_user_id": "user", "chatgpt_account_id": "account"}},
		{"iss": Issuer, "exp": time.Now().Add(time.Hour).Unix()},
		{"iss": Issuer, "https://api.openai.com/auth": map[string]any{"chatgpt_user_id": "user", "chatgpt_account_id": "account"}},
	} {
		if _, err := validatedNativeClaims(fakeJWT(t, claims)); err == nil {
			t.Fatal("unsupported access claims accepted")
		}
	}
}

func TestNativeConfigRejectsEndpointAndProviderOverrides(t *testing.T) {
	config := nativeFixtureConfig()
	if storage, err := validateNativeConfig(config); err != nil || storage != "file" {
		t.Fatalf("storage=%s err=%v", storage, err)
	}
	for _, key := range []string{"model_provider", "openai_base_url", "chatgpt_base_url", "forced_login_method"} {
		t.Run(key, func(t *testing.T) {
			config := nativeFixtureConfig()
			config[key] = json.RawMessage(`"unsafe"`)
			if _, err := validateNativeConfig(config); err == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
	config["model_providers"] = json.RawMessage(`{"openai":{"base_url":"https://example.invalid"}}`)
	if _, err := validateNativeConfig(config); err == nil {
		t.Fatal("custom built-in provider accepted")
	}
}

func nativeFixtureConfig() map[string]json.RawMessage {
	values := map[string]string{"forced_login_method": "chatgpt", "model_provider": "openai", "openai_base_url": nativeOpenAIBase, "chatgpt_base_url": nativeChatGPTBase, "cli_auth_credentials_store": "file"}
	config := map[string]json.RawMessage{}
	for key, value := range values {
		config[key], _ = json.Marshal(value)
	}
	return config
}

func TestNativeEnvironmentStripsCredentialAndEndpointOverrides(t *testing.T) {
	for _, key := range []string{"OPENAI_API_KEY", "CODEX_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_BASE_URL", "CODEX_APP_SERVER_CHATGPT_BASE_URL", "CODEX_REFRESH_TOKEN_URL_OVERRIDE", "CODEX_REVOKE_TOKEN_URL_OVERRIDE", "CODEX_APP_SERVER_LOGIN_CLIENT_ID"} {
		t.Setenv(key, "sensitive-fixture")
	}
	t.Setenv("CODEX_HOME", "wrong-home")
	for _, entry := range nativeEnvironment("/fixture/codex") {
		if strings.Contains(entry, "sensitive-fixture") || entry == "CODEX_HOME=wrong-home" {
			t.Fatal("native environment inherited credentials or endpoint override")
		}
	}
}

func installNativeFixture(t *testing.T, mode string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses Unix executable wrapper")
	}
	dir := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	wrapper := "#!/bin/sh\nexec \"$CLAUDODEX_AUTH_TEST_BINARY\" -test.run=TestNativeRPCSubprocess -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(wrapper), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CLAUDODEX_AUTH_TEST_BINARY", executable)
	t.Setenv("CLAUDODEX_AUTH_NATIVE_FIXTURE", mode)
	t.Setenv("CLAUDODEX_AUTH_NATIVE_ACCESS", nativeJWT(t, "user", "account", nil))
}

func TestNativeRPCReadsAccessWithoutPersistingAndValidatesProtocol(t *testing.T) {
	for _, mode := range []string{"ok", "api-key", "unsupported", "keyring", "bad-config", "oversized", "stderr-flood", "unexpected-request", "mismatched-email"} {
		t.Run(mode, func(t *testing.T) {
			installNativeFixture(t, mode)
			home := t.TempDir()
			writeNativeMetadata(t, filepath.Join(home, "auth.json"), nativeJWT(t, "user", "account", nil))
			before, _ := os.ReadFile(filepath.Join(home, "auth.json"))
			file, err := readNativeCodex(context.Background(), home, false)
			if mode == "ok" {
				if err != nil {
					t.Fatal(err)
				}
				if file.Tokens.RefreshToken != "" || file.Tokens.IDToken != "" || file.Tokens.UserID != "user" || file.Tokens.PlanType != "free" {
					t.Fatal("native credential result contains unsupported data")
				}
			} else if err == nil {
				t.Fatalf("mode %s accepted", mode)
			}
			if err != nil && (strings.Contains(err.Error(), os.Getenv("CLAUDODEX_AUTH_NATIVE_ACCESS")) || strings.Contains(err.Error(), "secret-error-payload")) {
				t.Fatal("native error exposed credentials or raw error payload")
			}
			after, _ := os.ReadFile(filepath.Join(home, "auth.json"))
			if string(before) != string(after) {
				t.Fatal("Claudodex changed the native credential file")
			}
			entries, _ := os.ReadDir(home)
			if len(entries) != 1 {
				t.Fatalf("Claudodex created %d extra files in Codex home", len(entries)-1)
			}
		})
	}
}

func TestNativeRPCCancelsOwnedProcess(t *testing.T) {
	installNativeFixture(t, "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := readNativeCodex(ctx, t.TempDir(), false); err == nil {
		t.Fatal("hung native process accepted")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("native cancellation failed to clean up promptly")
	}
}

// Test subprocess speaks only the few authentication RPC methods allowed by the
// provider. Any thread/generation call fails the fixture instead of making a call.
func TestNativeRPCSubprocess(t *testing.T) {
	mode := os.Getenv("CLAUDODEX_AUTH_NATIVE_FIXTURE")
	if mode == "" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params map[string]any  `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(3)
		}
		if request.Method == "initialized" {
			continue
		}
		if mode == "hang" {
			time.Sleep(time.Minute)
			return
		}
		if mode == "oversized" {
			fmt.Println(strings.Repeat("x", nativeMaxMessage+1))
			continue
		}
		if mode == "stderr-flood" {
			fmt.Fprint(os.Stderr, strings.Repeat("x", nativeMaxOutput+1))
			continue
		}
		if mode == "unexpected-request" {
			fmt.Println(`{"id":99,"method":"account/chatgptAuthTokens/refresh","params":{}}`)
			continue
		}
		var result any
		switch request.Method {
		case "initialize":
			result = map[string]any{}
		case "config/read":
			config := nativeFixtureConfig()
			if mode == "keyring" {
				config["cli_auth_credentials_store"] = json.RawMessage(`"keyring"`)
			}
			if mode == "bad-config" {
				config["model_provider"] = json.RawMessage(`"custom"`)
			}
			result = map[string]any{"config": config}
		case "account/read":
			kind, email := "chatgpt", "fixture@example.invalid"
			if mode == "api-key" {
				kind = "apiKey"
			}
			if mode == "mismatched-email" {
				email = "other@example.invalid"
			}
			result = map[string]any{"account": map[string]any{"type": kind, "email": email, "planType": "free"}}
		case "getAuthStatus":
			if mode == "unsupported" {
				encoded, _ := json.Marshal(map[string]any{"id": request.ID, "error": map[string]any{"message": "secret-error-payload"}})
				fmt.Println(string(encoded))
				continue
			}
			if request.Params["includeToken"] != true || request.Params["refreshToken"] != false {
				os.Exit(4)
			}
			result = map[string]any{"authMethod": "chatgpt", "authToken": os.Getenv("CLAUDODEX_AUTH_NATIVE_ACCESS")}
		default:
			os.Exit(5)
		}
		encoded, _ := json.Marshal(map[string]any{"id": request.ID, "result": result})
		fmt.Println(string(encoded))
	}
	os.Exit(0)
}
