package launcher

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestCatalogRecognitionOverlayRequiresExactRegisteredIdentity(t *testing.T) {
	for _, test := range []struct{ version, sha, function, matches string }{
		{"2.1.314", claude314SHA, "kat", "uv"},
		{"2.1.316", claude316SHA, "tlt", "wv"},
		{"2.1.318", claude318SHA, "Lmt", "bv"},
		{"2.1.317", claude317SHA, "Ect", "Lv"},
	} {
		t.Run(test.version, func(t *testing.T) {
			original := `function ` + test.function + `(e){return[i.ANTHROPIC_DEFAULT_FABLE_MODEL,i.ANTHROPIC_DEFAULT_OPUS_MODEL,i.ANTHROPIC_DEFAULT_SONNET_MODEL,i.ANTHROPIC_DEFAULT_HAIKU_MODEL].some((n)=>n!==void 0&&` + test.matches + `(n,e))}`
			data := []byte(original)
			if !patchClaudeCatalogRecognition(data, claudeSource{Version: test.version, SHA256: "wrong"}) || string(data) != original {
				t.Fatal("overlay changed a different binary")
			}
			if !patchClaudeCatalogRecognition(data, claudeSource{Version: test.version, SHA256: test.sha}) {
				t.Fatal("overlay failed for the exact registered anchor")
			}
			if len(data) != len(original) || !bytes.Contains(data, []byte(catalogModelIDsEnv)) || !bytes.Contains(data, []byte(test.matches+"(n,e)")) {
				t.Fatal("overlay changed size or dropped native matching")
			}
		})
	}
}

func TestCatalogRecognitionEnvReplacesInheritedAllowlist(t *testing.T) {
	env := envMap(BuildClaudeEnv([]string{catalogModelIDsEnv + "=untrusted-inherited-model"}, 1, t.TempDir(), "", "", "", []codex.ModelInfo{
		{Slug: "gpt-8.2-sol", SupportedInAPI: true, Visibility: "list"},
		{Slug: "gpt-6.1-sol", SupportedInAPI: true, Visibility: "list"},
		{Slug: "gpt-hidden-sol", SupportedInAPI: true, Visibility: "hidden"},
		{Slug: "gpt-noapi-sol", Visibility: "list"},
	}, modelconfig.Default()))
	if got := env[catalogModelIDsEnv]; got != "gpt-8.2-sol,gpt-6.1-sol" || strings.Contains(got, "inherited") {
		t.Fatalf("wrong catalog recognition allowlist: %q", got)
	}
}
