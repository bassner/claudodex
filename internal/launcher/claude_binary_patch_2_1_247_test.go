package launcher

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude247SHA = "5086b9b64d8bb842e1f599cdd3767ab08c6b2266e462fcc5686ae4b019cca8f7"

func TestClaude247PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.247", claude247SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.247 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.247", claude245SHA); got != nil {
		t.Fatalf("Claude 2.1.247 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.245", claude247SHA); got != nil {
		t.Fatalf("Claude 2.1.247 SHA matched wrong version: %#v", got)
	}
}

func TestClaude247WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.247"
	if err := os.WriteFile(claudePath, []byte("not the verified binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	got := prepareClaudeExecutable(context.Background(), t.TempDir(), claudePath, "test", modelconfig.Default(), &stderr)
	if got != claudePath {
		t.Fatalf("unsupported executable path = %q, want original %q", got, claudePath)
	}
	if !strings.Contains(stderr.String(), "no verified UI patch") || !strings.Contains(stderr.String(), "sha256:") {
		t.Fatalf("unsupported fallback warning = %q", stderr.String())
	}
}

func TestClaude247ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`)
	if !patchModelPickerOptions_2_1_247(data) {
		t.Fatal("patchModelPickerOptions_2_1_247 reported no changes")
	}
	assertClaude233Picker(t, string(data))
}

func TestClaude247ModelPickerSuppressesCustomFourthTier(t *testing.T) {
	data := []byte(`function vt(e,n){let o=mt(e),` + strings.Repeat(" ", 24000) + `function _t(e){}`)
	if !patchModelPickerExtraOptions_2_1_247(data) {
		t.Fatal("patchModelPickerExtraOptions_2_1_247 reported no changes")
	}
	got := string(data)
	if !strings.Contains(got, `function vt(e,n){return mt(e)}`) {
		t.Fatalf("model picker extra-options patch did not reduce the picker to the maintained three tiers: %s", got)
	}
	if strings.Contains(got, "ANTHROPIC_CUSTOM_MODEL_OPTION") {
		t.Fatalf("model picker retained a custom fourth-tier path: %s", got)
	}
}

func TestApplyClaudeUIPatches247RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude247Transformations("0.3.13") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude247PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude247PatchFixture(t)
	if !applyClaudeUIPatches_2_1_247(data, "0.3.13", "2.1.247", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_247 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.13 using Claude Code v2.1.247"`,
		"Claudodex  \x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar",
		"Sonnet",
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:I().optional()`,
		`function Gr(e){return Kt()}`,
		"Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok",
		`function gI(e){return e.fastMode===!0}`,
		`function r2(e,t){return Kt()&&!!e}`,
		`function pI(e){return Kt()&&(H("flagSettings")?.fastMode===!0||gI(Se()))}`,
		`function o2(e,t){return Kt()&&(t!==void 0?!!t:gI(Se()))}`,
		`fastMode:Ct.settings.fastMode===!0`,
		`fastMode:Pr(Ne??null)`,
		`fastMode:W.options.fastMode`,
		`o={model:n.model,fastMode:n.fastMode}`,
		`fastMode:t.fastMode`,
		`fastMode:nt`,
		`let Xe=[...L,...Ae],nt=!!s.fastMode;`,
		`if(et.fastMode)Ov="fast";`,
		`function Hu(e){return"Codex priority"}`,
		`function fN(_9t){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,t)/2000`,
		`function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function SB(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude233Picker(t, got)

	for _, target := range claude247RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude247PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_247(broken, "0.3.13", "2.1.247", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude247LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude247PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_247(data, strings.Repeat("x", 4000), "2.1.247") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude247LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := claude247PatchFixture(t)
	if !patchLogoDisplayDataFunction_2_1_247(data, "0.3.13", "2.1.247") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`r.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude247Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_247(data, version, "2.1.247") }},
		{"active-header-brand", patchActiveHeaderBrand_2_1_247},
		{"default-tier-label", patchDefaultTierLabel_2_1_247},
		{"whats-new", patchWhatsNewFeedFunction_2_1_247},
		{"usage", patchUsageFetchFunction_2_1_247},
		{"model-options", patchModelPickerOptions_2_1_247},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_247},
		{"model-selection", patchModelPickerSelectionValue_2_1_247},
		{"agent-model-validator", patchAgentModelValidator_2_1_247},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_247},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_247},
		{"fast-mode-pricing", patchFastModePricing_2_1_247},
		{"context-warning", patchContextWarningHint_2_1_247},
		{"resume-hints", patchResumeCommandHints_2_1_247},
		{"compact-progress", patchCompactProgressCurve_2_1_247},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_247},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude247UIBrandingReplacements)
		}},
	}
}

func claude247RequiredTargets() []string {
	return []string{
		"function Z(){let o=f.DEMO_VERSION??",
		claude247ActiveHeaderBrandTarget,
		"Default (recommended)",
		"var ue=async(o,e)=>{try{",
		"async function J(t){",
		"function mt(e=!1){",
		"function vt(e,n){let o=mt(e),",
		"function ks(e,n){if(e.some((i)=>i.value===n))return n;",
		`model:un(["sonnet","opus","haiku","fable"]).optional()`,
		`function Kt(){if(k()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		claude247ActiveFastModeBrandTarget,
		`function dI(){return"Opus 5"}`,
		`function fI(){return"opus"+(zr()?"[1m]":"")}`,
		`function gI(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(H("policySettings")?.fastModePerSessionOptIn===!0)return!1;return H("flagSettings")?.fastMode===!0}`,
		`function r2(e,t){if(!Kt())return!1;return!!e&&(yf()||Ef()||t)}`,
		`function pI(e){if(!Kt())return!1;if(!Ef(e))return!1;if(!Gr(e))return!1;return gI(Se())}`,
		`function o2(e,t){if(yf()){if(e===null)return!!t;return!!t&&Gr(e)}if(!Gr(e))return!1;return!!t||pI(e)}`,
		`if(GP(W,r,l.storageV5),Tw())r((Ct)=>{let sn=Rw(Ct.settings);return Ct.fastMode===sn?Ct:{...Ct,fastMode:sn}});`,
		`function Gr(e){if(!Kt())return!1;`,
		`...dl()&&{fastMode:Pr(Ne??null)}`,
		`...T.gates.fastModeEnabled&&{fastMode:W.options.fastMode}`,
		`o={model:n.model,...Am()&&{fastMode:n.fastMode}}`,
		`...Am()&&{fastMode:t.fastMode}`,
		`...Am()&&{fastMode:nt}`,
		`let Xe=[...L,...Ae],nt=Am()&&nGt()&&!dhe()&&rGt(y)&&!!s.fastMode;`,
		`if(Am()&&nGt()&&!dhe()&&rGt(y)&&!!et.fastMode)Ov="fast";`,
		"function Hu(e){return`${Eb(e.inputTokens)}/${Eb(e.outputTokens)} per Mtok`}",
		"function fN(_9t){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \\xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		"function f(){return}function E(){return}",
		`function SB(){if(fc())return!0;if(ra())return!1;return!go()&&wg()}`,
		`async function vB(){if(fc())return!0;if(ra())return!1;return na()&&!go()&&hi()&&await ui("tengu_ccr_bridge")}`,
		"async function CT(){",
		claude247UIBrandingReplacements[0].old,
	}
}

func claude247PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function Z(){let o=f.DEMO_VERSION??` + strings.Repeat(" ", 2200) + `function L(o,r,t){}`,
		claude247ActiveHeaderBrandTarget,
		strings.Repeat("Default (recommended)\x00", 4),
		`var ue=async(o,e)=>{try{` + strings.Repeat(" ", 1800) + `function z(X){}`,
		`async function J(t){` + strings.Repeat(" ", 2400) + `var P="fixture";`,
		`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`,
		`function vt(e,n){let o=mt(e),` + strings.Repeat(" ", 24000) + `function _t(e){}`,
		`function ks(e,n){if(e.some((i)=>i.value===n))return n;` + strings.Repeat(" ", 1200) + `function mo(){}`,
		`model:un(["sonnet","opus","haiku","fable"]).optional()`,
		`function Kt(){if(k()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		claude247ActiveFastModeBrandTarget,
		`function dI(){return"Opus 5"}`,
		`function fI(){return"opus"+(zr()?"[1m]":"")}`,
		`function gI(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(H("policySettings")?.fastModePerSessionOptIn===!0)return!1;return H("flagSettings")?.fastMode===!0}`,
		`function r2(e,t){if(!Kt())return!1;return!!e&&(yf()||Ef()||t)}`,
		`function pI(e){if(!Kt())return!1;if(!Ef(e))return!1;if(!Gr(e))return!1;return gI(Se())}`,
		`function o2(e,t){if(yf()){if(e===null)return!!t;return!!t&&Gr(e)}if(!Gr(e))return!1;return!!t||pI(e)}`,
		`if(GP(W,r,l.storageV5),Tw())r((Ct)=>{let sn=Rw(Ct.settings);return Ct.fastMode===sn?Ct:{...Ct,fastMode:sn}});`,
		`function Gr(e){if(!Kt())return!1;let t=e??Lu(),n=J(t);if(fn(O(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`,
		`...dl()&&{fastMode:Pr(Ne??null)}`,
		`...T.gates.fastModeEnabled&&{fastMode:W.options.fastMode}`,
		`o={model:n.model,...Am()&&{fastMode:n.fastMode}}`,
		`...Am()&&{fastMode:t.fastMode}`,
		strings.Repeat(`...Am()&&{fastMode:nt}`+"\x00", 2),
		`let Xe=[...L,...Ae],nt=Am()&&nGt()&&!dhe()&&rGt(y)&&!!s.fastMode;`,
		`if(Am()&&nGt()&&!dhe()&&rGt(y)&&!!et.fastMode)Ov="fast";`,
		"function Hu(e){return`${Eb(e.inputTokens)}/${Eb(e.outputTokens)} per Mtok`}",
		`function fN(_9t){` + strings.Repeat(" ", 7000) + `JF();Se();ta();r3();He();wt();Vr();hN();Cot();ne();`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \\xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		`function f(){return}function E(){return}function l(){let e=f();if(e!==void 0)return e;if(!r()||!d())return;return o()?.accessToken}async function R(e){if(!(t()&&e!==void 0))return l();let n=f();if(n!==void 0)return n;if(!r()||!await u(e))return;return(await s(e))?.accessToken}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}function _(e){}`,
		`function SB(){if(fc())return!0;if(ra())return!1;return!go()&&wg()}`,
		`async function vB(){if(fc())return!0;if(ra())return!1;return na()&&!go()&&hi()&&await ui("tengu_ccr_bridge")}`,
		`async function CT(){` + strings.Repeat(" ", 12000) + `function CB(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude247UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude247UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.247 fixture failed branding-count validation")
	}
	return data
}
