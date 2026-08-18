package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude234SHA = "08d8700313697cbe730a25420c908a299ce52d56f0eb2cf4fac94cab5109bc57"

func TestClaude234PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.234", claude234SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.234 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.234", claude233SHA); got != nil {
		t.Fatalf("Claude 2.1.234 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.233", claude234SHA); got != nil {
		t.Fatalf("Claude 2.1.234 SHA matched wrong version: %#v", got)
	}
}

func TestClaude234ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function knb(e=!1){` + strings.Repeat(" ", 16000) + `function GOe(e){}`)
	if !patchModelPickerOptions_2_1_234(data) {
		t.Fatal("patchModelPickerOptions_2_1_234 reported no changes")
	}
	assertClaude233Picker(t, string(data))
}

func TestApplyClaudeUIPatches234RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude234Transformations("0.3.10") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude234PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude234PatchFixture(t)
	if !applyClaudeUIPatches_2_1_234(data, "0.3.10", "2.1.234", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_234 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.10 using Claude Code v2.1.234"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:F().optional()`,
		`function yk(e){return ku()}`,
		`function Llt(e){return"Codex priority"}`,
		`function Ons(Bd1){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function k1e(){return V.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function kM(){return!!V.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude233Picker(t, got)

	for _, target := range claude234RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude234PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_234(broken, "0.3.10", "2.1.234", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude234LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude234PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_234(data, strings.Repeat("x", 4000), "2.1.234") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude234LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := claude234PatchFixture(t)
	if !patchLogoDisplayDataFunction_2_1_234(data, "0.3.10", "2.1.234") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`t.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude234Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_234(data, version, "2.1.234") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_234},
		{"usage", patchUsageFetchFunction_2_1_234},
		{"model-options", patchModelPickerOptions_2_1_234},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_234},
		{"model-selection", patchModelPickerSelectionValue_2_1_234},
		{"agent-model-validator", patchAgentModelValidator_2_1_234},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_234},
		{"fast-mode-pricing", patchFastModePricing_2_1_234},
		{"context-warning", patchContextWarningHint_2_1_234},
		{"resume-hints", patchResumeCommandHints_2_1_234},
		{"compact-progress", patchCompactProgressCurve_2_1_234},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_234},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude234UIBrandingReplacements)
		}},
	}
}

func claude234RequiredTargets() []string {
	return []string{
		"function Qjt(){",
		"function Gql(e){",
		"async function aje(){",
		"function knb(e=!1){",
		"function xnb(e,t){",
		"function $Js(e,t){",
		`model:Mr(["sonnet","opus","haiku","fable"]).optional()`,
		`function ku(){if(Jn()!=="firstParty")return!1;return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function iJ(){return"Opus 5"}`,
		`function SKt(){return"opus"+(_4()?"[1m]":"")}`,
		`function yk(e){if(!ku())return!1;`,
		"function Llt(e){return`${eJu(e.inputTokens)}/${eJu(e.outputTokens)} per Mtok`}",
		"function Ons(Bd1){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \\xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function oHm(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		"function k1e(){return}",
		`function kM(){if(Fjo())return!0;if(k7t())return!1;return!bJ()&&C7t()}`,
		`async function zGs(){if(Fjo())return!0;if(k7t())return!1;return rUe()&&!bJ()&&$Pr()&&await R6("tengu_ccr_bridge")}`,
		"async function $jo(){",
		claude234UIBrandingReplacements[0].old,
	}
}

func claude234PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function Qjt(){` + strings.Repeat(" ", 2200) + `function Cql(e,t,r){}`,
		`function Gql(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function aje(){` + strings.Repeat(" ", 2200) + `var PDb="fixture";`,
		`function knb(e=!1){` + strings.Repeat(" ", 18000) + `function GOe(e){}`,
		`function xnb(e,t){` + strings.Repeat(" ", 18000) + `function $Js(e,t){` + strings.Repeat(" ", 1200) + `function SAd(){}`,
		`model:Mr(["sonnet","opus","haiku","fable"]).optional()`,
		`function ku(){if(Jn()!=="firstParty")return!1;return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function iJ(){return"Opus 5"}`,
		`function SKt(){return"opus"+(_4()?"[1m]":"")}`,
		`function yk(e){if(!ku())return!1;let t=e??V3(),r=ys(t);if(sF(jo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function Llt(e){return`${eJu(e.inputTokens)}/${eJu(e.outputTokens)} per Mtok`}",
		`function Ons(Bd1){` + strings.Repeat(" ", 5000) + `var Y8w,Tmo,Ltg;`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \\xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function oHm(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function k1e(){return}function NXe(){return}function oj(){let e=k1e();if(e!==void 0)return e;if(!Sd()||!Yi())return;return ua()?.accessToken}function Jor(){return NXe()??qa().BASE_API_URL}function lLt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||sYp.hostname();return aYp(e)||"remote-control"}function aYp(e){}`,
		`function kM(){if(Fjo())return!0;if(k7t())return!1;return!bJ()&&C7t()}`,
		`async function zGs(){if(Fjo())return!0;if(k7t())return!1;return rUe()&&!bJ()&&$Pr()&&await R6("tengu_ccr_bridge")}`,
		`async function $jo(){` + strings.Repeat(" ", 10000) + `function N8_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude234UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude234UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.234 fixture failed branding-count validation")
	}
	return data
}
