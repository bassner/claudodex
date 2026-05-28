package launcher

import (
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestApplyClaudeUIPatchesBrandsHeaderAndModelPicker(t *testing.T) {
	data := []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`Most capable for complex work`,
		`Best for everyday tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`j4.createElement(V,{bold:!0},"Claude Code")`,
		`Lq("claude",d)("Claude Code")`,
		`Lq("claude",d)(" Claude Code ")`,
		`w_=h4()?Y?P4.createElement(B):null:null`,
		`function jl3(H=!1){if(Zq()){if(Re()||wAH()||IUH()){let z=[ML6(H)];if(!LP()&&X6H()&&!Zr8())z.push(lkK());if(z.push(Al3),Q5H())z.push(ckK());return z.push(nkK),z}function Jl3(H){}`,
	}, "\n"))

	if !applyClaudeUIPatches(data, "0.1.2", "2.1.153", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		"Claudodex v0.1.2 using Claude Code v2.1.153",
		"Codex news",
		"Codex Plan",
		"Switch between Codex-backed models.",
		"Codex model",
		"Default (Claudodex)",
		"default Codex work",
		"gpt-5.4 everyday coding",
		"gpt-5.4-mini quick code",
		` via Codex model \xB7 `,
		`"Claudodex  "`,
		"w_=0?",
		"let z=[]",
		"return z",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"z.push(lkK())",
		"z.push(Al3)",
		"z.push(ckK())",
		"return z.push(nkK),z",
		"let z=[ML6(H)]",
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("patched data still contains %q:\n%s", notWant, got)
		}
	}
}

func TestDetectClaudeVersionFromResolvedPath(t *testing.T) {
	if !looksLikeVersion("2.1.153") {
		t.Fatal("version-like path was rejected")
	}
	if looksLikeVersion("claude") {
		t.Fatal("non-version path was accepted")
	}
}
