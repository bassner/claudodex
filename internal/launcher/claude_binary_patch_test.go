package launcher

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestApplyClaudeUIPatchesBrandsHeaderAndModelPicker(t *testing.T) {
	data := []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Welcome back!`,
		`Set the AI model for Claude Code`,
		`Claude Code'll be able to read, edit, and execute files here.`,
		`WARNING: Claude Code running in Bypass Permissions mode`,
		`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`,
		`No, exit Claude Code`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`Most capable for complex work`,
		`Best for everyday tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`j4.createElement(V,{bold:!0},"Claude Code")`,
		`j4.createElement(V,{dimColor:!0},"v",E)`,
		`Lq("claude",d)("Claude Code")`,
		`Lq("inactive",d)(` + "`v${h}`" + `)`,
		`Lq("claude",d)(" Claude Code ")`,
		`w_=h4()?Y?P4.createElement(B):null:null`,
		`function P6_(){let H="2.1.153",_=Px6(),q="/code/claude",K=q,O=vq(),T=O!=="firstParty"?eYH[O]:Zq()?zH6():"API Usage Billing",z=o8().agent;return{version:H,cwd:K,billingType:T,agentName:z}}                                                                                                                                                                                                                                                                                                                                                                               function RuK(H,_,q){}`,
		`function muK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`function jl3(H=!1){if(Zq()){if(Re()||wAH()||IUH()){let z=[ML6(H)];if(!LP()&&X6H()&&!Zr8())z.push(lkK());if(z.push(Al3),Q5H())z.push(ckK());return z.push(nkK),z}function Jl3(H){}`,
	}, "\n"))

	if !applyClaudeUIPatches_2_1_153(data, "0.1.0", "2.1.153", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_153 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`Bugs:\ngithub.com/bassner/claudodex/issues`,
		"Info",
		"Welcome back",
		"Set the AI model for Claudodex",
		"Claudodex can read, edit, and execute files here.",
		"WARNING: Claudodex running in Bypass Permissions mode",
		"In Bypass Permissions mode, Claudodex will not ask",
		"No, exit Claudodex",
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"Experimental - treat it as such.",
		"If you run into issues, please file a report at",
		"https://github.com/bassner/claudodex/issues",
		"Codex Plan",
		"Switch between Codex-backed models.",
		"Codex model",
		"Default (Claudodex)",
		"default Codex work",
		"gpt-5.4 everyday coding",
		"gpt-5.4-mini quick code",
		` via Codex model \xB7 `,
		`"Claudodex  "`,
		`"0.1.0 using Claude Code v2.1.153"`,
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

func TestFindClaudeUIPatchRequiresVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.153", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.153 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.154", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254"); got != nil {
		t.Fatalf("patch matched unsupported version: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.153", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
}

func TestWarnClaudePatchSkippedMentionsPatchTargetAndIssue(t *testing.T) {
	var stderr bytes.Buffer
	warnClaudePatchSkipped(&stderr, "2.1.154", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", errClaudePatchUnsupported)
	got := stderr.String()
	for _, want := range []string{
		"warning: Claudodex has no verified UI patch",
		"Claude Code 2.1.154",
		runtime.GOOS + "/" + runtime.GOARCH,
		"sha256:ffffffffffff",
		"launching with the unpatched Claude Code UI",
		"https://github.com/bassner/claudodex/issues",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("warning missing %q:\n%s", want, got)
		}
	}
}

func TestWarnClaudePatchSkippedHandlesPatchFailures(t *testing.T) {
	var stderr bytes.Buffer
	warnClaudePatchSkipped(&stderr, "2.1.153", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254", errors.New("boom"))
	got := stderr.String()
	if !strings.Contains(got, "could not prepare the UI patch") || !strings.Contains(got, "boom") {
		t.Fatalf("warning = %q", got)
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
