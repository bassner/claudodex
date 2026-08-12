package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude228SHA = "43484b1352cef03a08346f36ef0437755b1aad646ab9313ce187857b794b7247"

func TestClaude228PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.228", claude228SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.228 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.228", claude227SHA); got != nil {
		t.Fatalf("Claude 2.1.228 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.227", claude228SHA); got != nil {
		t.Fatalf("Claude 2.1.228 SHA matched wrong version: %#v", got)
	}
}

func TestClaude228ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function BB_(e=!1){` + strings.Repeat(" ", 14000) + `function ixe(e){}`)
	if !patchModelPickerOptions_2_1_228(data) {
		t.Fatal("patchModelPickerOptions_2_1_228 reported no changes")
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

func TestApplyClaudeUIPatches228RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude228Transformations("0.3.5") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude228PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude228PatchFixture(t)
	if !applyClaudeUIPatches_2_1_228(data, "0.3.5", "2.1.228", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_228 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.5 using Claude Code v2.1.228"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:N().optional()`,
		`function dC(e){return xc()}`,
		`function UZe(e){return"Codex priority"}`,
		`function JgE(e,t,r){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function cGe(){return X.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function FO(){return!!X.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
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

	for _, target := range claude228RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude228PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_228(broken, "0.3.5", "2.1.228", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude228LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude228PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_228(data, strings.Repeat("x", 4000), "2.1.228") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func claude228Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_228(data, version, "2.1.228") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_228},
		{"usage", patchUsageFetchFunction_2_1_228},
		{"model-options", patchModelPickerOptions_2_1_228},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_228},
		{"model-selection", patchModelPickerSelectionValue_2_1_228},
		{"agent-model-validator", patchAgentModelValidator_2_1_228},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_228},
		{"fast-mode-pricing", patchFastModePricing_2_1_228},
		{"context-warning", patchContextWarningHint_2_1_228},
		{"resume-hints", patchResumeCommandHints_2_1_228},
		{"compact-progress", patchCompactProgressCurve_2_1_228},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_228},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude228UIBrandingReplacements)
		}},
	}
}

func claude228RequiredTargets() []string {
	return []string{
		"function kMt(){",
		"function Kbl(e){",
		"async function ZFe(){",
		"function BB_(e=!1){",
		"function qB_(e,t){",
		`yjT=J7e===null?Eut:Rzs(X7e,J7e)??J7e`,
		`model:Dr(["sonnet","opus","haiku","fable"]).optional()`,
		`function xc(){if(Vn()!=="firstParty")return!1;return!X.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function uV(){return"Opus 5"}`,
		`function m4t(){return"opus"+(N2()?"[1m]":"")}`,
		`function dC(e){if(!xc())return!1;let t=e??p3(),r=as(t);if(zU(Do(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function UZe(e){return`${kvu(e.inputTokens)}/${kvu(e.outputTokens)} per Mtok`}",
		"function JgE(e,t,r){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function iGf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function cGe(){return}`,
		`function FO(){if(OLo())return!0;if(Lzt())return!1;return!rK()&&Mzt()}`,
		`async function zBs(){if(OLo())return!0;if(Lzt())return!1;return Hrt()&&!rK()&&Fwr()&&await Ij("tengu_ccr_bridge")}`,
		"async function DLo(){",
		claude228UIBrandingReplacements[0].old,
	}
}

func claude228PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function kMt(){` + strings.Repeat(" ", 1800) + `function Rbl(e,t,r){}`,
		`function Kbl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function ZFe(){` + strings.Repeat(" ", 1800) + `var Rnb="fixture";`,
		`function BB_(e=!1){` + strings.Repeat(" ", 14000) + `function ixe(e){}`,
		`function qB_(e,t){` + strings.Repeat(" ", 14000) + `function Rzs(e,t){}`,
		`yjT=J7e===null?Eut:Rzs(X7e,J7e)??J7e`,
		`model:Dr(["sonnet","opus","haiku","fable"]).optional()`,
		`function xc(){if(Vn()!=="firstParty")return!1;return!X.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function uV(){return"Opus 5"}`,
		`function m4t(){return"opus"+(N2()?"[1m]":"")}`,
		`function dC(e){if(!xc())return!1;let t=e??p3(),r=as(t);if(zU(Do(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function UZe(e){return`${kvu(e.inputTokens)}/${kvu(e.outputTokens)} per Mtok`}",
		`function JgE(e,t,r){` + strings.Repeat(" ", 2200) + `var bzh="fixture";`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function iGf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function cGe(){return}function uGe(){return}function X6(){let e=cGe();if(e!==void 0)return e;if(!Vu()||!Ui())return;return Js()?.accessToken}function pRn(){return uGe()??Sa().BASE_API_URL}function xAt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||Lap.hostname();return Nap(e)||"remote-control"}function Nap(e){}`,
		`function FO(){if(OLo())return!0;if(Lzt())return!1;return!rK()&&Mzt()}`,
		`async function zBs(){if(OLo())return!0;if(Lzt())return!1;return Hrt()&&!rK()&&Fwr()&&await Ij("tengu_ccr_bridge")}`,
		`async function DLo(){` + strings.Repeat(" ", 6000) + `function Rk_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude228UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude228UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.228 fixture failed branding-count validation")
	}
	return data
}
