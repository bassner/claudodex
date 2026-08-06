package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude223SHA = "fcbe0b8d47570c501302dd1ad31cc26ac2810f022c45fa253936a6961dee32bf"

func TestClaude223PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.223", claude223SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.223 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.223", claude222SHA); got != nil {
		t.Fatalf("Claude 2.1.223 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.222", claude223SHA); got != nil {
		t.Fatalf("Claude 2.1.223 SHA matched wrong version: %#v", got)
	}
}

func TestClaude223ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function aZg(e=!1){` + strings.Repeat(" ", 14000) + `function Cve(e){}`)
	if !patchModelPickerOptions_2_1_223(data) {
		t.Fatal("patchModelPickerOptions_2_1_223 reported no changes")
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

func TestApplyClaudeUIPatches223RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := claude223Transformations("0.2.3")
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude223PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude223PatchFixture(t)
	if !applyClaudeUIPatches_2_1_223(data, "0.2.3", "2.1.223", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_223 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.2.3 using Claude Code v2.1.223"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:E.string().optional()`,
		`function xE(e){return rc()}`,
		`function $Ve(e){return"Codex priority"}`,
		`function fRi(FIP){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function u9e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function Xx(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
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
		"function NAt(){",
		"function JXa(e){",
		"async function yOe(){",
		"function aZg(e=!1){",
		"function uZg(e){",
		`pBS=M5e===null?Jrt:Wss(L5e,M5e)??M5e`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function rc(){if(Mn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function cG(){return"Opus 5"}`,
		`function JMt(){return"opus"+(o$()?"[1m]":"")}`,
		`function xE(e){if(!rc())return!1;let t=e??mF(),r=Bi(t);if(K2(to(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function $Ve(e){return`${sXc(e.inputTokens)}/${sXc(e.outputTokens)} per Mtok`}",
		"function fRi(FIP){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function lmf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function u9e(){return}`,
		`function Xx(){if(mvo())return!0;if(S$t())return!1;return!zG()&&b$t()}`,
		`async function XSs(){if(mvo())return!0;if(S$t())return!1;return jKe()&&!zG()&&efr()&&await GG("tengu_ccr_bridge")}`,
		"async function hvo(){",
		claude223UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude223PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_223(broken, "0.2.3", "2.1.223", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude223LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude223PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_223(data, strings.Repeat("x", 4000), "2.1.223") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

type claude223Transformation struct {
	name  string
	apply func([]byte) bool
}

func claude223Transformations(version string) []claude223Transformation {
	return []claude223Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_223(data, version, "2.1.223") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_223},
		{"usage", patchUsageFetchFunction_2_1_223},
		{"model-options", patchModelPickerOptions_2_1_223},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_223},
		{"model-selection", patchModelPickerSelectionValue_2_1_223},
		{"agent-model-validator", patchAgentModelValidator_2_1_223},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_223},
		{"fast-mode-pricing", patchFastModePricing_2_1_223},
		{"context-warning", patchContextWarningHint_2_1_223},
		{"resume-hints", patchResumeCommandHints_2_1_223},
		{"compact-progress", patchCompactProgressCurve_2_1_223},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_223},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude223UIBrandingReplacements)
		}},
	}
}

func claude223PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function NAt(){` + strings.Repeat(" ", 1200) + `function xXa(e,t,r){}`,
		`function JXa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function yOe(){` + strings.Repeat(" ", 1800) + `var NP_="fixture";`,
		`function aZg(e=!1){` + strings.Repeat(" ", 14000) + `function Cve(e){}`,
		`function uZg(e){` + strings.Repeat(" ", 14000) + `function Wss(e,t){}`,
		`pBS=M5e===null?Jrt:Wss(L5e,M5e)??M5e`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function rc(){if(Mn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function cG(){return"Opus 5"}`,
		`function JMt(){return"opus"+(o$()?"[1m]":"")}`,
		`function xE(e){if(!rc())return!1;let t=e??mF(),r=Bi(t);if(K2(to(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function $Ve(e){return`${sXc(e.inputTokens)}/${sXc(e.outputTokens)} per Mtok`}",
		`function fRi(FIP){` + strings.Repeat(" ", 2200) + `var g1T,y9n,HHm;`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function lmf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function u9e(){return}function d9e(){return}function E8(){let e=u9e();if(e!==void 0)return e;if(!iu()||!Ri())return;return Os()?.accessToken}function jfn(){return d9e()??Zs().BASE_API_URL}function Wfn(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||QDd.hostname();return ZDd(e)||"remote-control"}function ZDd(e){}`,
		`function Xx(){if(mvo())return!0;if(S$t())return!1;return!zG()&&b$t()}`,
		`async function XSs(){if(mvo())return!0;if(S$t())return!1;return jKe()&&!zG()&&efr()&&await GG("tengu_ccr_bridge")}`,
		`async function hvo(){` + strings.Repeat(" ", 5000) + `function yD_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude223UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude223UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.223 fixture failed branding-count validation")
	}
	return data
}
