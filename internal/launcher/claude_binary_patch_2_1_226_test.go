package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude226SHA = "013a1cf17df5ff1dcc189d5d6fd3fdd5f097ddc3cd41aa9992e99805574febbe"

func TestClaude226PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.226", claude226SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.226 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.226", claude223SHA); got != nil {
		t.Fatalf("Claude 2.1.226 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.223", claude226SHA); got != nil {
		t.Fatalf("Claude 2.1.226 SHA matched wrong version: %#v", got)
	}
}

func TestClaude226ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Hw_(e=!1){` + strings.Repeat(" ", 14000) + `function Kwe(e){}`)
	if !patchModelPickerOptions_2_1_226(data) {
		t.Fatal("patchModelPickerOptions_2_1_226 reported no changes")
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

func TestApplyClaudeUIPatches226RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := claude226Transformations("0.3.5")
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude226PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude226PatchFixture(t)
	if !applyClaudeUIPatches_2_1_226(data, "0.3.5", "2.1.226", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_226 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.5 using Claude Code v2.1.226"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:N().optional()`,
		`function vw(e){return gc()}`,
		`function UYe(e){return"Codex priority"}`,
		`function q9v(e,t,r){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function eje(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function oP(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
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
		"function UIt(){",
		"function Cpl(e){",
		"async function YHe(){",
		"function Hw_(e=!1){",
		"function Nw_(e){",
		`ZhT=BGe===null?Tst:F_s(UGe,BGe)??BGe`,
		`model:Nr(["sonnet","opus","haiku","fable"]).optional()`,
		`function gc(){if(Kn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function SG(){return"Opus 5"}`,
		`function nFt(){return"opus"+(fF()?"[1m]":"")}`,
		`function vw(e){if(!gc())return!1;let t=e??xB(),r=ns(t);if(lB(Eo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function UYe(e){return`${dpu(e.inputTokens)}/${dpu(e.outputTokens)} per Mtok`}",
		"function q9v(e,t,r){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function qHf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function eje(){return}`,
		`function oP(){if(Wxo())return!0;if(BUt())return!1;return!i8()&&FUt()}`,
		`async function aOs(){if(Wxo())return!0;if(BUt())return!1;return KXe()&&!i8()&&byr()&&await p3("tengu_ccr_bridge")}`,
		"async function Gxo(){",
		claude226UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude226PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_226(broken, "0.3.5", "2.1.226", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude226LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude226PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_226(data, strings.Repeat("x", 4000), "2.1.226") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

type claude226Transformation struct {
	name  string
	apply func([]byte) bool
}

func claude226Transformations(version string) []claude226Transformation {
	return []claude226Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_226(data, version, "2.1.226") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_226},
		{"usage", patchUsageFetchFunction_2_1_226},
		{"model-options", patchModelPickerOptions_2_1_226},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_226},
		{"model-selection", patchModelPickerSelectionValue_2_1_226},
		{"agent-model-validator", patchAgentModelValidator_2_1_226},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_226},
		{"fast-mode-pricing", patchFastModePricing_2_1_226},
		{"context-warning", patchContextWarningHint_2_1_226},
		{"resume-hints", patchResumeCommandHints_2_1_226},
		{"compact-progress", patchCompactProgressCurve_2_1_226},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_226},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude226UIBrandingReplacements)
		}},
	}
}

func claude226PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function UIt(){` + strings.Repeat(" ", 1800) + `function opl(e,t,r){}`,
		`function Cpl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function YHe(){` + strings.Repeat(" ", 1800) + `var eey="fixture";`,
		`function Hw_(e=!1){` + strings.Repeat(" ", 14000) + `function Kwe(e){}`,
		`function Nw_(e){` + strings.Repeat(" ", 14000) + `function F_s(e,t){}`,
		`ZhT=BGe===null?Tst:F_s(UGe,BGe)??BGe`,
		`model:Nr(["sonnet","opus","haiku","fable"]).optional()`,
		`function gc(){if(Kn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function SG(){return"Opus 5"}`,
		`function nFt(){return"opus"+(fF()?"[1m]":"")}`,
		`function vw(e){if(!gc())return!1;let t=e??xB(),r=ns(t);if(lB(Eo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function UYe(e){return`${dpu(e.inputTokens)}/${dpu(e.outputTokens)} per Mtok`}",
		`function q9v(e,t,r){` + strings.Repeat(" ", 2200) + `var qxh="fixture";`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function qHf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function eje(){return}function tje(){return}function F8(){let e=eje();if(e!==void 0)return e;if(!Ru()||!Ni())return;return Ls()?.accessToken}function PSn(){return tje()??sa().BASE_API_URL}function uvt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||nWd.hostname();return oWd(e)||"remote-control"}function oWd(e){}`,
		`function oP(){if(Wxo())return!0;if(BUt())return!1;return!i8()&&FUt()}`,
		`async function aOs(){if(Wxo())return!0;if(BUt())return!1;return KXe()&&!i8()&&byr()&&await p3("tengu_ccr_bridge")}`,
		`async function Gxo(){` + strings.Repeat(" ", 6000) + `function Mty(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude226UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude226UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.226 fixture failed branding-count validation")
	}
	return data
}
