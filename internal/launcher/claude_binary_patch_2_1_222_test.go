package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude222SHA = "c66a6cc6fa2e8145bb1a6e77831f2caf4b83690ff04650500dfa6e2c05ca997c"

func TestClaude222PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.222", claude222SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.222 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.222", "7a181f36ed0fc4fbac6cee4ecf2b615eff93d8b434221fff5d7c878dc5ebf380"); got != nil {
		t.Fatalf("Claude 2.1.222 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.221", claude222SHA); got != nil {
		t.Fatalf("Claude 2.1.222 SHA matched wrong version: %#v", got)
	}
}

func TestClaude222ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function E7g(e=!1){` + strings.Repeat(" ", 14000) + `function QTe(e){}`)
	if !patchModelPickerOptions_2_1_222(data) {
		t.Fatal("patchModelPickerOptions_2_1_222 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched picker missing %q", want)
		}
	}
	if tiers := strings.Count(got, `n("`); tiers != 3 {
		t.Fatalf("patched picker tier count = %d, want 3", tiers)
	}
	for _, forbidden := range []string{"claude-opus-5", "Opus 5", "fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("patched picker retained forbidden native-tier marker %q", forbidden)
		}
	}
}

func TestApplyClaudeUIPatches222RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_222(data, "0.2.3", "2.1.222") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_222},
		{"usage", patchUsageFetchFunction_2_1_222},
		{"model-options", patchModelPickerOptions_2_1_222},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_222},
		{"model-selection", patchModelPickerSelectionValue_2_1_222},
		{"agent-model-validator", patchAgentModelValidator_2_1_222},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_222},
		{"fast-mode-pricing", patchFastModePricing_2_1_222},
		{"context-warning", patchContextWarningHint_2_1_222},
		{"resume-hints", patchResumeCommandHints_2_1_222},
		{"compact-progress", patchCompactProgressCurve_2_1_222},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_222},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude222UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude222PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude222PatchFixture(t)
	if !applyClaudeUIPatches_2_1_222(data, "0.2.3", "2.1.222", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_222 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.2.3 using Claude Code v2.1.222"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:E.string().optional()`,
		`function EE(e){return Ql()}`,
		`function lVe(e){return"Codex priority"}`,
		`function iAi(BSP){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function F4e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function $x(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	if tiers := strings.Count(got, `n("`); tiers != 3 {
		t.Fatalf("patched picker tier count = %d, want 3", tiers)
	}
	for _, forbidden := range []string{`n("claude-opus-5",`, `n("fable",`, `n("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL", "Fable 5", "Mythos 5"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("patched fixture retained forbidden native-tier marker %q", forbidden)
		}
	}

	requiredTargets := []string{
		"function d0t(){",
		"function nYa(e){",
		"async function UPe(){",
		"function E7g(e=!1){",
		"function A7g(e){",
		`tMS=a5e===null?wrt:Nos(l5e,a5e)??a5e`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function Ql(){if(Ln()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function QW(){return"Opus 5"}`,
		`function SMt(){return"opus"+(r$()?"[1m]":"")}`,
		`function EE(e){if(!Ql())return!1;let t=e??sF(),r=$i(t);if(B2(ao(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function lVe(e){return`${W7c(e.inputTokens)}/${W7c(e.outputTokens)} per Mtok`}",
		"function iAi(BSP){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function ucf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function F4e(){return}`,
		`function $x(){if(OSo())return!0;if(BNt())return!1;return!NG()&&FNt()}`,
		`async function Bys(){if(OSo())return!0;if(BNt())return!1;return _Ke()&&!NG()&&dpr()&&await LG("tengu_ccr_bridge")}`,
		"async function DSo(){",
		claude222UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude222PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_222(broken, "0.2.3", "2.1.222", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude222LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude222PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_222(data, strings.Repeat("x", 4000), "2.1.222") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func claude222PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function d0t(){` + strings.Repeat(" ", 1200) + `function LKa(e,t,r){}`,
		`function nYa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function UPe(){` + strings.Repeat(" ", 1800) + `var KA_="fixture";`,
		`function E7g(e=!1){` + strings.Repeat(" ", 14000) + `function QTe(e){}`,
		`function A7g(e){` + strings.Repeat(" ", 9000) + `function Nos(e,t){}`,
		`tMS=a5e===null?wrt:Nos(l5e,a5e)??a5e`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function Ql(){if(Ln()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function QW(){return"Opus 5"}`,
		`function SMt(){return"opus"+(r$()?"[1m]":"")}`,
		`function EE(e){if(!Ql())return!1;let t=e??sF(),r=$i(t);if(B2(ao(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function lVe(e){return`${W7c(e.inputTokens)}/${W7c(e.outputTokens)} per Mtok`}",
		`function iAi(BSP){` + strings.Repeat(" ", 2200) + `var nIT,W3n,Ixm;`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function ucf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function F4e(){return}function B4e(){return}function d8(){let e=F4e();if(e!==void 0)return e;if(!mu()||!vi())return;return Fs()?.accessToken}function Hpn(){return B4e()??Qs().BASE_API_URL}function Mpn(){}`,
		`function $x(){if(OSo())return!0;if(BNt())return!1;return!NG()&&FNt()}`,
		`async function Bys(){if(OSo())return!0;if(BNt())return!1;return _Ke()&&!NG()&&dpr()&&await LG("tengu_ccr_bridge")}`,
		`async function DSo(){` + strings.Repeat(" ", 5000) + `function xR_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude222UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude222UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.222 fixture failed branding-count validation")
	}
	return data
}
