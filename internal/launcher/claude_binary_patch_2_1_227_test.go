package launcher

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude227SHA = "7432511ba3be818e01f23f6eef8630d214a8b618451e188c3c7d61a987eef6c7"

func TestClaude227PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.227", claude227SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.227 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.227", claude226SHA); got != nil {
		t.Fatalf("Claude 2.1.227 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.226", claude227SHA); got != nil {
		t.Fatalf("Claude 2.1.227 SHA matched wrong version: %#v", got)
	}
}

func TestClaude227ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Ixy(e=!1){` + strings.Repeat(" ", 14000) + `function bAe(e){}`)
	if !patchModelPickerOptions_2_1_227(data) {
		t.Fatal("patchModelPickerOptions_2_1_227 reported no changes")
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

func TestApplyClaudeUIPatches227RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := claude227Transformations("0.3.5")
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude227PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude227PatchFixture(t)
	if !applyClaudeUIPatches_2_1_227(data, "0.3.5", "2.1.227", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_227 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.5 using Claude Code v2.1.227"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
		`model:N().optional()`,
		`function nC(e){return Cc()}`,
		`function nQe(e){return"Codex priority"}`,
		`function U7v(e,t,r){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function f5e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function aO(){return!!re.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
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
		"function aDt(){",
		"function lul(e){",
		"async function HLe(){",
		"function Ixy(e=!1){",
		"function Dxy(e,t){",
		`kAT=cKe===null?Rlt:Xhs(uKe,cKe)??cKe`,
		`model:Pr(["sonnet","opus","haiku","fable"]).optional()`,
		`function Cc(){if(Wn()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function M8(){return"Opus 5"}`,
		`function MBt(){return"opus"+(g2()?"[1m]":"")}`,
		`function nC(e){if(!Cc())return!1;let t=e??$U(),r=ns(t);if(gU(Ro(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function nQe(e){return`${Euu(e.inputTokens)}/${Euu(e.outputTokens)} per Mtok`}",
		"function U7v(e,t,r){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function wNf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function f5e(){return}`,
		`function aO(){if(X0o())return!0;if(F4t())return!1;return!yV()&&N4t()}`,
		`async function tks(){if(X0o())return!0;if(F4t())return!1;return $Ze()&&!yV()&&Pbr()&&await nj("tengu_ccr_bridge")}`,
		"async function Q0o(){",
		claude227UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude227PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_227(broken, "0.3.5", "2.1.227", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude227LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude227PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_227(data, strings.Repeat("x", 4000), "2.1.227") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

type claude227Transformation struct {
	name  string
	apply func([]byte) bool
}

func claude227Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_227(data, version, "2.1.227") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_227},
		{"usage", patchUsageFetchFunction_2_1_227},
		{"model-options", patchModelPickerOptions_2_1_227},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_227},
		{"model-selection", patchModelPickerSelectionValue_2_1_227},
		{"agent-model-validator", patchAgentModelValidator_2_1_227},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_227},
		{"fast-mode-pricing", patchFastModePricing_2_1_227},
		{"context-warning", patchContextWarningHint_2_1_227},
		{"resume-hints", patchResumeCommandHints_2_1_227},
		{"compact-progress", patchCompactProgressCurve_2_1_227},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_227},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude227UIBrandingReplacements)
		}},
	}
}

func claude227PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function aDt(){` + strings.Repeat(" ", 1800) + `function Ucl(e,t,r){}`,
		`function lul(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function HLe(){` + strings.Repeat(" ", 1800) + `var bs_="fixture";`,
		`function Ixy(e=!1){` + strings.Repeat(" ", 14000) + `function bAe(e){}`,
		`function Dxy(e,t){` + strings.Repeat(" ", 14000) + `function Xhs(e,t){}`,
		`kAT=cKe===null?Rlt:Xhs(uKe,cKe)??cKe`,
		`model:Pr(["sonnet","opus","haiku","fable"]).optional()`,
		`function Cc(){if(Wn()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function M8(){return"Opus 5"}`,
		`function MBt(){return"opus"+(g2()?"[1m]":"")}`,
		`function nC(e){if(!Cc())return!1;let t=e??$U(),r=ns(t);if(gU(Ro(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`,
		"function nQe(e){return`${Euu(e.inputTokens)}/${Euu(e.outputTokens)} per Mtok`}",
		`function U7v(e,t,r){` + strings.Repeat(" ", 2200) + `var BHh="fixture";`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function wNf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function f5e(){return}function m5e(){return}function sK(){let e=f5e();if(e!==void 0)return e;if(!Nu()||!Di())return;return Bs()?.accessToken}function Tvn(){return m5e()??ua().BASE_API_URL}function Iwt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||u8d.hostname();return d8d(e)||"remote-control"}function d8d(e){}`,
		`function aO(){if(X0o())return!0;if(F4t())return!1;return!yV()&&N4t()}`,
		`async function tks(){if(X0o())return!0;if(F4t())return!1;return $Ze()&&!yV()&&Pbr()&&await nj("tengu_ccr_bridge")}`,
		`async function Q0o(){` + strings.Repeat(" ", 6000) + `function wt_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude227UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude227UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.227 fixture failed branding-count validation")
	}
	return data
}
