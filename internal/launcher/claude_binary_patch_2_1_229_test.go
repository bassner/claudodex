package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude229SHA = "d732f0ba0a539c58c2ffcaef06ed03b4e523726f0cb6cc27b3a5b7e7ae0a7a21"

func TestClaude229PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.229", claude229SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.229 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.229", claude228SHA); got != nil {
		t.Fatalf("Claude 2.1.229 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.228", claude229SHA); got != nil {
		t.Fatalf("Claude 2.1.229 SHA matched wrong version: %#v", got)
	}
}

func TestClaude229ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function uL_(e=!1){` + strings.Repeat(" ", 16000) + `function lxe(e){}`)
	if !patchModelPickerOptions_2_1_229(data) {
		t.Fatal("patchModelPickerOptions_2_1_229 reported no changes")
	}
	assertClaude229Picker(t, string(data))
}

func TestApplyClaudeUIPatches229RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude229Transformations("0.3.8") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude229PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude229PatchFixture(t)
	if !applyClaudeUIPatches_2_1_229(data, "0.3.8", "2.1.229", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_229 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.8 using Claude Code v2.1.229"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:N().optional()`,
		`function T0(e){return $c()}`,
		`function dtt(e){return"Codex priority"}`,
		`function KAw(e,t,r){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function _8e(){return Q.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function OO(){return!!Q.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude229Picker(t, got)

	for _, target := range claude229RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude229PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_229(broken, "0.3.8", "2.1.229", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude229LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude229PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_229(data, strings.Repeat("x", 4000), "2.1.229") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func assertClaude229Picker(t *testing.T, got string) {
	t.Helper()
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

func claude229Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_229(data, version, "2.1.229") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_229},
		{"usage", patchUsageFetchFunction_2_1_229},
		{"model-options", patchModelPickerOptions_2_1_229},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_229},
		{"model-selection", patchModelPickerSelectionValue_2_1_229},
		{"agent-model-validator", patchAgentModelValidator_2_1_229},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_229},
		{"fast-mode-pricing", patchFastModePricing_2_1_229},
		{"context-warning", patchContextWarningHint_2_1_229},
		{"resume-hints", patchResumeCommandHints_2_1_229},
		{"compact-progress", patchCompactProgressCurve_2_1_229},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_229},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude229UIBrandingReplacements)
		}},
	}
}

func claude229RequiredTargets() []string {
	return []string{
		"function mLt(){",
		"function oEl(e){",
		"async function QFe(){",
		"function uL_(e=!1){",
		"function fL_(e,t){",
		"function n5s(e,t){",
		`model:Mr(["sonnet","opus","haiku","fable"]).optional()`,
		`function $c(){if(Jn()!=="firstParty")return!1;return!Q.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function NV(){return"Opus 5"}`,
		`function Jjt(){return"opus"+(s$()?"[1m]":"")}`,
		`function T0(e){if(!$c())return!1;let t=e??O3(),r=ls(t);if(y2(Bo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function dtt(e){return`${lAu(e.inputTokens)}/${lAu(e.outputTokens)} per Mtok`}",
		"function KAw(e,t,r){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function pXf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		"function _8e(){return}",
		`function OO(){if(qxo())return!0;if(Z5t())return!1;return!tK()&&Q5t()}`,
		`async function jIs(){if(qxo())return!0;if(Z5t())return!1;return ort()&&!tK()&&Vvr()&&await Yj("tengu_ccr_bridge")}`,
		"async function Wxo(){",
		claude229UIBrandingReplacements[0].old,
	}
}

func claude229PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function mLt(){` + strings.Repeat(" ", 1900) + `function Nwl(e,t,r){}`,
		`function oEl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function QFe(){` + strings.Repeat(" ", 1900) + `var ecb="fixture";`,
		`function uL_(e=!1){` + strings.Repeat(" ", 16000) + `function lxe(e){}`,
		`function fL_(e,t){` + strings.Repeat(" ", 16000) + `function n5s(e,t){` + strings.Repeat(" ", 1000) + `function qyd(){}`,
		`model:Mr(["sonnet","opus","haiku","fable"]).optional()`,
		`function $c(){if(Jn()!=="firstParty")return!1;return!Q.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function NV(){return"Opus 5"}`,
		`function Jjt(){return"opus"+(s$()?"[1m]":"")}`,
		`function T0(e){if(!$c())return!1;let t=e??O3(),r=ls(t);if(y2(Bo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function dtt(e){return`${lAu(e.inputTokens)}/${lAu(e.outputTokens)} per Mtok`}",
		`function KAw(e,t,r){` + strings.Repeat(" ", 2400) + `var wYh="fixture";`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function pXf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function _8e(){return}function b8e(){return}function S4(){let e=_8e();if(e!==void 0)return e;if(!ed()||!$i())return;return Js()?.accessToken}function L7t(){return b8e()??Sa().BASE_API_URL}function Fkt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||ihp.hostname();return shp(e)||"remote-control"}function shp(e){}`,
		`function OO(){if(qxo())return!0;if(Z5t())return!1;return!tK()&&Q5t()}`,
		`async function jIs(){if(qxo())return!0;if(Z5t())return!1;return ort()&&!tK()&&Vvr()&&await Yj("tengu_ccr_bridge")}`,
		`async function Wxo(){` + strings.Repeat(" ", 7000) + `function or_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude229UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude229UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.229 fixture failed branding-count validation")
	}
	return data
}
