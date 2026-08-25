package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude245SHA = "9f7c2260251765a18d0b35198669dacc1912f6e8129a3b01f6b58d93365ff1f1"

func TestClaude245PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.245", claude245SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.245 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.245", claude234SHA); got != nil {
		t.Fatalf("Claude 2.1.245 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.234", claude245SHA); got != nil {
		t.Fatalf("Claude 2.1.245 SHA matched wrong version: %#v", got)
	}
}

func TestClaude245ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`)
	if !patchModelPickerOptions_2_1_245(data) {
		t.Fatal("patchModelPickerOptions_2_1_245 reported no changes")
	}
	assertClaude233Picker(t, string(data))
}

func TestApplyClaudeUIPatches245RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude245Transformations("0.3.11") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude245PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude245PatchFixture(t)
	if !applyClaudeUIPatches_2_1_245(data, "0.3.11", "2.1.245", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_245 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.11 using Claude Code v2.1.245"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:D().optional()`,
		`function Ar(e){return Dt()}`,
		`function nu(e){return"Codex priority"}`,
		`function bL(r1t){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,t)/2000`,
		`function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function ZF(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude233Picker(t, got)

	for _, target := range claude245RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude245PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_245(broken, "0.3.11", "2.1.245", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude245LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude245PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_245(data, strings.Repeat("x", 4000), "2.1.245") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude245LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := claude245PatchFixture(t)
	if !patchLogoDisplayDataFunction_2_1_245(data, "0.3.11", "2.1.245") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`n.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude245Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_245(data, version, "2.1.245") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_245},
		{"usage", patchUsageFetchFunction_2_1_245},
		{"model-options", patchModelPickerOptions_2_1_245},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_245},
		{"model-selection", patchModelPickerSelectionValue_2_1_245},
		{"agent-model-validator", patchAgentModelValidator_2_1_245},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_245},
		{"fast-mode-pricing", patchFastModePricing_2_1_245},
		{"context-warning", patchContextWarningHint_2_1_245},
		{"resume-hints", patchResumeCommandHints_2_1_245},
		{"compact-progress", patchCompactProgressCurve_2_1_245},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_245},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude245UIBrandingReplacements)
		}},
	}
}

func claude245RequiredTargets() []string {
	return []string{
		"function gt(){let r=p.DEMO_VERSION??",
		"function Qa(e){",
		"async function J(t){",
		"function mt(e=!1){",
		"function vt(e,n){",
		"function ks(e,n){if(e.some((i)=>i.value===n))return n;",
		`model:Cn(["sonnet","opus","haiku","fable"]).optional()`,
		`function Dt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function QO(){return"Opus 5"}`,
		`function ZO(){return"opus"+(wr()?"[1m]":"")}`,
		`function Ar(e){if(!Dt())return!1;`,
		"function nu(e){return`${$_(e.inputTokens)}/${$_(e.outputTokens)} per Mtok`}",
		"function bL(r1t){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \\xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		"function f(){return}function E(){return}",
		`function ZF(){if(Rl())return!0;if(As())return!1;return!Jr()&&lp()}`,
		`async function eL(){if(Rl())return!0;if(As())return!1;return Cs()&&!Jr()&&qo()&&await Ko("tengu_ccr_bridge")}`,
		"async function VC(){",
		claude245UIBrandingReplacements[0].old,
	}
}

func claude245PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function gt(){let r=p.DEMO_VERSION??` + strings.Repeat(" ", 2200) + `function mt(r,n,t){}`,
		`function Qa(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function J(t){` + strings.Repeat(" ", 2400) + `var P="fixture";`,
		`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`,
		`function vt(e,n){` + strings.Repeat(" ", 24000) + `function _t(e){}`,
		`function ks(e,n){if(e.some((i)=>i.value===n))return n;` + strings.Repeat(" ", 1200) + `function mo(){}`,
		`model:Cn(["sonnet","opus","haiku","fable"]).optional()`,
		`function Dt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function QO(){return"Opus 5"}`,
		`function ZO(){return"opus"+(wr()?"[1m]":"")}`,
		`function Ar(e){if(!Dt())return!1;let t=e??Qa(),n=q(t);if(en(R(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`,
		"function nu(e){return`${$_(e.inputTokens)}/${$_(e.outputTokens)} per Mtok`}",
		`function bL(r1t){` + strings.Repeat(" ", 7000) + `vD();`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \\xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		`function f(){return}function E(){return}function l(){let e=f();if(e!==void 0)return e;if(!r()||!d())return;return o()?.accessToken}async function R(e){if(!(t()&&e!==void 0))return l();let n=f();if(n!==void 0)return n;if(!r()||!await u(e))return;return(await s(e))?.accessToken}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}function _(e){}`,
		`function ZF(){if(Rl())return!0;if(As())return!1;return!Jr()&&lp()}`,
		`async function eL(){if(Rl())return!0;if(As())return!1;return Cs()&&!Jr()&&qo()&&await Ko("tengu_ccr_bridge")}`,
		`async function VC(){` + strings.Repeat(" ", 12000) + `function tL(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude245UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude245UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.245 fixture failed branding-count validation")
	}
	return data
}
