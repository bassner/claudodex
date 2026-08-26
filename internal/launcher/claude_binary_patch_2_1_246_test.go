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

const claude246SHA = "7b09f01cb76a38e0e3a7c47c5d698d382162a5ff26538fc778683770caf9218b"

func TestClaude246PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.246", claude246SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.246 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.246", claude245SHA); got != nil {
		t.Fatalf("Claude 2.1.246 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.245", claude246SHA); got != nil {
		t.Fatalf("Claude 2.1.246 SHA matched wrong version: %#v", got)
	}
}

func TestClaude246WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.246"
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

func TestClaude246ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`)
	if !patchModelPickerOptions_2_1_246(data) {
		t.Fatal("patchModelPickerOptions_2_1_246 reported no changes")
	}
	assertClaude233Picker(t, string(data))
}

func TestApplyClaudeUIPatches246RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude246Transformations("0.3.12") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude246PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude246PatchFixture(t)
	if !applyClaudeUIPatches_2_1_246(data, "0.3.12", "2.1.246", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_246 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.12 using Claude Code v2.1.246"`,
		"Claudodex  \x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar",
		"Sonnet",
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:I().optional()`,
		`function Pr(e){return Bt()}`,
		"Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok",
		`function HD(e){return e.fastMode===!0}`,
		`function bY(e,t){return Bt()&&!!e}`,
		`function BD(e){return Bt()&&(G("flagSettings")?.fastMode===!0||HD(ve()))}`,
		`function EY(e,t){return Bt()&&(t!==void 0?!!t:HD(ve()))}`,
		`fastMode:gt.settings.fastMode===!0`,
		`fastMode:Pr(Ne??null)`,
		`fastMode:z.options.fastMode`,
		`o={model:n.model,fastMode:n.fastMode}`,
		`fastMode:t.fastMode`,
		`fastMode:et`,
		`let qe=[...O,...Xe],et=!!s.fastMode;`,
		`if(He.fastMode)wv="fast";`,
		`function Lu(e){return"Codex priority"}`,
		`function gL($Xt){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,t)/2000`,
		`function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function W0(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude233Picker(t, got)

	for _, target := range claude246RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude246PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_246(broken, "0.3.12", "2.1.246", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude246LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude246PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_246(data, strings.Repeat("x", 4000), "2.1.246") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude246LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := claude246PatchFixture(t)
	if !patchLogoDisplayDataFunction_2_1_246(data, "0.3.12", "2.1.246") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`r.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude246Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_246(data, version, "2.1.246") }},
		{"active-header-brand", patchActiveHeaderBrand_2_1_246},
		{"default-tier-label", patchDefaultTierLabel_2_1_246},
		{"whats-new", patchWhatsNewFeedFunction_2_1_246},
		{"usage", patchUsageFetchFunction_2_1_246},
		{"model-options", patchModelPickerOptions_2_1_246},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_246},
		{"model-selection", patchModelPickerSelectionValue_2_1_246},
		{"agent-model-validator", patchAgentModelValidator_2_1_246},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_246},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_246},
		{"fast-mode-pricing", patchFastModePricing_2_1_246},
		{"context-warning", patchContextWarningHint_2_1_246},
		{"resume-hints", patchResumeCommandHints_2_1_246},
		{"compact-progress", patchCompactProgressCurve_2_1_246},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_246},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude246UIBrandingReplacements)
		}},
	}
}

func claude246RequiredTargets() []string {
	return []string{
		"function Z(){let o=f.DEMO_VERSION??",
		claude246ActiveHeaderBrandTarget,
		"Default (recommended)",
		"var ue=async(o,e)=>{try{",
		"async function J(t){",
		"function mt(e=!1){",
		"function vt(e,n){let o=mt(e),",
		"function ks(e,n){if(e.some((i)=>i.value===n))return n;",
		`model:ln(["sonnet","opus","haiku","fable"]).optional()`,
		`function Bt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		claude246ActiveFastModeBrandTarget,
		`function LD(){return"Opus 5"}`,
		`function UD(){return"opus"+(Mr()?"[1m]":"")}`,
		`function HD(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(G("policySettings")?.fastModePerSessionOptIn===!0)return!1;return G("flagSettings")?.fastMode===!0}`,
		`function bY(e,t){if(!Bt())return!1;return!!e&&(uf()||cf()||t)}`,
		`function BD(e){if(!Bt())return!1;if(!cf(e))return!1;if(!Pr(e))return!1;return HD(ve())}`,
		`function EY(e,t){if(uf()){if(e===null)return!!t;return!!t&&Pr(e)}if(!Pr(e))return!1;return!!t||BD(e)}`,
		`if(sE(B,r,a.storageV5),xC())r((gt)=>{let At=FC(gt.settings);return gt.fastMode===At?gt:{...gt,fastMode:At}});`,
		`function Pr(e){if(!Bt())return!1;`,
		`...dl()&&{fastMode:Pr(Ne??null)}`,
		`...k.gates.fastModeEnabled&&{fastMode:z.options.fastMode}`,
		`o={model:n.model,...hm()&&{fastMode:n.fastMode}}`,
		`...hm()&&{fastMode:t.fastMode}`,
		`...hm()&&{fastMode:et}`,
		`let qe=[...O,...Xe],et=hm()&&xWt()&&!wge()&&IWt(y)&&!!s.fastMode;`,
		`if(hm()&&xWt()&&!wge()&&IWt(y)&&!!He.fastMode)wv="fast";`,
		"function Lu(e){return`${sb(e.inputTokens)}/${sb(e.outputTokens)} per Mtok`}",
		"function gL($Xt){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \\xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		"function f(){return}function E(){return}",
		`function W0(){if(sc())return!0;if(qs())return!1;return!ro()&&pg()}`,
		`async function j0(){if(sc())return!0;if(qs())return!1;return Ys()&&!ro()&&fi()&&await oi("tengu_ccr_bridge")}`,
		"async function eT(){",
		claude246UIBrandingReplacements[0].old,
	}
}

func claude246PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function Z(){let o=f.DEMO_VERSION??` + strings.Repeat(" ", 2200) + `function L(o,r,t){}`,
		claude246ActiveHeaderBrandTarget,
		strings.Repeat("Default (recommended)\x00", 4),
		`var ue=async(o,e)=>{try{` + strings.Repeat(" ", 1800) + `function z(X){}`,
		`async function J(t){` + strings.Repeat(" ", 2400) + `var P="fixture";`,
		`function mt(e=!1){` + strings.Repeat(" ", 24000) + `function h(e){}`,
		`function vt(e,n){let o=mt(e),` + strings.Repeat(" ", 24000) + `function _t(e){}`,
		`function ks(e,n){if(e.some((i)=>i.value===n))return n;` + strings.Repeat(" ", 1200) + `function mo(){}`,
		`model:ln(["sonnet","opus","haiku","fable"]).optional()`,
		`function Bt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		claude246ActiveFastModeBrandTarget,
		`function LD(){return"Opus 5"}`,
		`function UD(){return"opus"+(Mr()?"[1m]":"")}`,
		`function HD(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(G("policySettings")?.fastModePerSessionOptIn===!0)return!1;return G("flagSettings")?.fastMode===!0}`,
		`function bY(e,t){if(!Bt())return!1;return!!e&&(uf()||cf()||t)}`,
		`function BD(e){if(!Bt())return!1;if(!cf(e))return!1;if(!Pr(e))return!1;return HD(ve())}`,
		`function EY(e,t){if(uf()){if(e===null)return!!t;return!!t&&Pr(e)}if(!Pr(e))return!1;return!!t||BD(e)}`,
		`if(sE(B,r,a.storageV5),xC())r((gt)=>{let At=FC(gt.settings);return gt.fastMode===At?gt:{...gt,fastMode:At}});`,
		`function Pr(e){if(!Bt())return!1;let t=e??Iu(),n=X(t);if(un(P(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`,
		`...dl()&&{fastMode:Pr(Ne??null)}`,
		`...k.gates.fastModeEnabled&&{fastMode:z.options.fastMode}`,
		`o={model:n.model,...hm()&&{fastMode:n.fastMode}}`,
		`...hm()&&{fastMode:t.fastMode}`,
		strings.Repeat(`...hm()&&{fastMode:et}`+"\x00", 2),
		`let qe=[...O,...Xe],et=hm()&&xWt()&&!wge()&&IWt(y)&&!!s.fastMode;`,
		`if(hm()&&xWt()&&!wge()&&IWt(y)&&!!He.fastMode)wv="fast";`,
		"function Lu(e){return`${sb(e.inputTokens)}/${sb(e.outputTokens)} per Mtok`}",
		`function gL($Xt){` + strings.Repeat(" ", 7000) + `nN();_e();Ts();`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \\xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`,
		`function f(){return}function E(){return}function l(){let e=f();if(e!==void 0)return e;if(!r()||!d())return;return o()?.accessToken}async function R(e){if(!(t()&&e!==void 0))return l();let n=f();if(n!==void 0)return n;if(!r()||!await u(e))return;return(await s(e))?.accessToken}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}function _(e){}`,
		`function W0(){if(sc())return!0;if(qs())return!1;return!ro()&&pg()}`,
		`async function j0(){if(sc())return!0;if(qs())return!1;return Ys()&&!ro()&&fi()&&await oi("tengu_ccr_bridge")}`,
		`async function eT(){` + strings.Repeat(" ", 12000) + `function Y0(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude246UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude246UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.246 fixture failed branding-count validation")
	}
	return data
}
