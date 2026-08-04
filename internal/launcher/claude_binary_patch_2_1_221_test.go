package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestClaude221PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.221", "7a181f36ed0fc4fbac6cee4ecf2b615eff93d8b434221fff5d7c878dc5ebf380")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.221 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.221", "8addc857f3fe64d5a0368af9ee50321b50afb4a6918ba3ef018ab84f5dbbe081"); got != nil {
		t.Fatalf("Claude 2.1.221 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.220", "7a181f36ed0fc4fbac6cee4ecf2b615eff93d8b434221fff5d7c878dc5ebf380"); got != nil {
		t.Fatalf("Claude 2.1.221 SHA matched wrong version: %#v", got)
	}
}

func TestClaude221ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function xGg(e=!1){` + strings.Repeat(" ", 14000) + `function wTe(e){}`)
	if !patchModelPickerOptions_2_1_221(data) {
		t.Fatal("patchModelPickerOptions_2_1_221 reported no changes")
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

func TestApplyClaudeUIPatches221RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_221(data, "0.2.3", "2.1.221") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_221},
		{"usage", patchUsageFetchFunction_2_1_221},
		{"model-options", patchModelPickerOptions_2_1_221},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_221},
		{"model-selection", patchModelPickerSelectionValue_2_1_221},
		{"agent-model-validator", patchAgentModelValidator_2_1_221},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_221},
		{"fast-mode-pricing", patchFastModePricing_2_1_221},
		{"context-warning", patchContextWarningHint_2_1_221},
		{"resume-hints", patchResumeCommandHints_2_1_221},
		{"compact-progress", patchCompactProgressCurve_2_1_221},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_221},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude221UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude221PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude221PatchFixture(t)
	if !applyClaudeUIPatches_2_1_221(data, "0.2.3", "2.1.221", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_221 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.2.3 using Claude Code v2.1.221"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:E.string().optional()`,
		`function gE(e){return Wl()}`,
		`function Mze(e){return"Codex priority"}`,
		`function JEi(LpP){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function V4e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function jx(){return!!re.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
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
		"function AAt(){",
		"function sVa(e){",
		"async function qPe(){",
		"function xGg(e=!1){",
		"function OGg(e){",
		`NxS=Mje===null?Wtt:$rs(Lje,Mje)??Mje`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function Wl(){if(Ln()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function LW(){return"Opus 5"}`,
		`function DHt(){return"opus"+(GN()?"[1m]":"")}`,
		`function gE(e){if(!Wl())return!1;let t=e??z2(),r=Oi(t);if(R2(co(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function Mze(e){return`${pzc(e.inputTokens)}/${pzc(e.outputTokens)} per Mtok`}",
		"function JEi(LpP){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function Nif(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function V4e(){return}`,
		`function jx(){if(gJo())return!0;if(Fjt())return!1;return!pG()&&T$t()}`,
		`async function Qma(){if(gJo())return!0;if(Fjt())return!1;return kKe()&&!pG()&&kRr()&&await VW("tengu_ccr_bridge")}`,
		"async function _Jo(){",
		claude221UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude221PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_221(broken, "0.2.3", "2.1.221", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude221LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude221PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_221(data, strings.Repeat("x", 4000), "2.1.221") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func claude221PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function AAt(){` + strings.Repeat(" ", 1000) + `function Fza(e,t,r){}`,
		`function sVa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function qPe(){` + strings.Repeat(" ", 1600) + `var rB_="fixture";`,
		`function xGg(e=!1){` + strings.Repeat(" ", 14000) + `function wTe(e){}`,
		`function OGg(e){` + strings.Repeat(" ", 7000) + `function $rs(e,t){}`,
		`NxS=Mje===null?Wtt:$rs(Lje,Mje)??Mje`,
		`model:E.enum(["sonnet","opus","haiku","fable"]).optional()`,
		`function Wl(){if(Ln()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function LW(){return"Opus 5"}`,
		`function DHt(){return"opus"+(GN()?"[1m]":"")}`,
		`function gE(e){if(!Wl())return!1;let t=e??z2(),r=Oi(t);if(R2(co(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function Mze(e){return`${pzc(e.inputTokens)}/${pzc(e.outputTokens)} per Mtok`}",
		`function JEi(LpP){` + strings.Repeat(" ", 2000) + `var BCT,uUn,fAm;`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function Nif(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function V4e(){return}function K4e(){return}function WG(){let e=V4e();if(e!==void 0)return e;if(!su()||!hi())return;return As()?.accessToken}function yhn(){return K4e()??Us().BASE_API_URL}function bhn(){}`,
		`function jx(){if(gJo())return!0;if(Fjt())return!1;return!pG()&&T$t()}`,
		`async function Qma(){if(gJo())return!0;if(Fjt())return!1;return kKe()&&!pG()&&kRr()&&await VW("tengu_ccr_bridge")}`,
		`async function _Jo(){` + strings.Repeat(" ", 4000) + `function SNb(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude221UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude221UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.221 fixture failed branding-count validation")
	}
	return data
}
