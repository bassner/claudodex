package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthLogoutCommandSuppressesAmbientCodexReuse(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, "native-codex")
	if err := os.Mkdir(codexHome, 0700); err != nil {
		t.Fatal(err)
	}
	nativePath := filepath.Join(codexHome, "auth.json")
	if err := os.WriteFile(nativePath, []byte("native fixture must stay untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	var out, stderr bytes.Buffer
	cfg := Config{Home: home, Stdout: &out, Stderr: &stderr}
	if code := Run(context.Background(), cfg, []string{"clx:auth-logout"}); code != 0 {
		t.Fatalf("logout code=%d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), "logged out") {
		t.Fatalf("logout output=%q", out.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".claudodex", "codex-reuse-disabled")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := Run(context.Background(), cfg, []string{"clx:auth-status"}); code != 1 {
		t.Fatalf("status code=%d", code)
	}
	if strings.TrimSpace(out.String()) != "not logged in" {
		t.Fatalf("status=%q stderr=%q", out.String(), stderr.String())
	}
	data, err := os.ReadFile(nativePath)
	if err != nil || string(data) != "native fixture must stay untouched" {
		t.Fatal("logout command changed native Codex auth")
	}
	if _, err := os.Stat(filepath.Join(home, ".claudodex", "codex-login.json")); !os.IsNotExist(err) {
		t.Fatal("logout recreated native login reference")
	}
}

func TestAuthLoginCommandRejectsUnexpectedCodexArguments(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), Config{Home: t.TempDir(), Stderr: &stderr}, []string{"clx:auth-login", "--codex", "unexpected"})
	if code != 2 || !strings.Contains(stderr.String(), "[--codex]") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
