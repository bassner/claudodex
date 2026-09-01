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

const claude252SHA = "b661c6a094fcc32656bf7c0071c5b45bf900b34d4f0a1ab3d78fd59aeba2c2c7"

func TestClaude252PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.252", claude252SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.252 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.252", claude251SHA); got != nil {
		t.Fatalf("Claude 2.1.252 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.251", claude252SHA); got != nil {
		t.Fatalf("Claude 2.1.252 SHA matched wrong version: %#v", got)
	}
}

func TestClaude252WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.252"
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

func TestClaude252ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function tn(e=!1){` + strings.Repeat(" ", 32000) + `function S(e){}`)
	if !patchModelPickerOptions_2_1_252(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_252 reported no changes")
	}
	assertClaude233Picker(t, string(data))
}

func TestClaude252ModelPickerNormalizerIncludesConfiguredTargets(t *testing.T) {
	data := []byte(`function tn(e=!1){` + strings.Repeat(" ", 32000) + `function S(e){}`)
	configured := modelconfig.Config{Opus: "custom-sol", Sonnet: "custom-terra", Haiku: "custom-luna"}
	if !patchModelPickerOptions_2_1_252(data, configured) {
		t.Fatal("patchModelPickerOptions_2_1_252 reported no changes")
	}
	got := string(data)
	for _, want := range []string{`o="custom-sol"`, `s="custom-terra"`, `h="custom-luna"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker normalizer missing configured target %q", want)
		}
	}
}

func TestClaude252ModelPickerSuppressesFourthTierAndCustomOptions(t *testing.T) {
	data := []byte(`function iwe(e=!1,o=null){` + strings.Repeat(" ", 32000) + `function ln(e,o){let t=tn(e),` + strings.Repeat(" ", 32000) + `function Smt(e,o){}` + `function npn(e){}` + `Be=ye??n,Ve=Be===null?T:Smt(q,Be)??Be,` + `Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:Smt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:wC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])` + `defaultValue:Ve,selectedValue:Ve,`)
	if !patchModelPickerExtraOptions_2_1_252(data) {
		t.Fatal("patchModelPickerExtraOptions_2_1_252 reported no changes")
	}
	got := string(data)
	if !strings.Contains(got, `function ln(e,o){return tn(e)}`) {
		t.Fatalf("model picker extra-options patch did not reduce the picker to three tiers: %s", got)
	}
	if !strings.Contains(got, "Ho=q.slice(0,3)") {
		t.Fatalf("model picker retained a session-only fourth option: %s", got)
	}
	if !strings.Contains(got, `Be=ye??n,Ve=Be===null?T:CDX252(Be),`) {
		t.Fatalf("model picker retained a direct configured target as a custom selected option: %s", got)
	}
	if strings.Contains(got, `defaultValue:Ve`) || strings.Contains(got, `selectedValue:Ve`) || strings.Contains(got, `selectedValue:CDX252(Ve)`) {
		t.Fatalf("model picker retained selected-value injection that creates a fourth row: %s", got)
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_CUSTOM_MODEL_OPTION"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestApplyClaudeUIPatches252RequiresAndAppliesEveryTransformation(t *testing.T) {
	for _, transformation := range claude252Transformations("0.3.15") {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			if !transformation.apply(claude252PatchFixture(t)) {
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude252PatchFixture(t)
	if !applyClaudeUIPatches_2_1_252(data, "0.3.15", "2.1.252", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_252 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.3.15 using Claude Code v2.1.252"`,
		"Claudodex  \x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle",
		"Sonnet",
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`model:i().optional()`,
		`function Smt(e,o){let n=CDX252(o),t=e.find((r)=>r.value===n||CDX252(r.value)===n);return t?.value??n}`,
		`function lf(e){return Yr()}`,
		"Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok",
		`function DSn(e){return e.fastMode===!0}`,
		`function Her(e,t){return Yr()&&!!e}`,
		`function NSt(e){return Yr()&&(ye("flagSettings")?.fastMode===!0||DSn(Je()))}`,
		`function dw(e,t){return Yr()&&(t!==void 0?!!t:DSn(Je()))}`,
		`fastMode:NSt(et??null)`,
		`fastMode:Ke.options.fastMode`,
		`u={model:r.model,fastMode:r.fastMode}`,
		`fastMode:t.fastMode`,
		`fastMode:Ar`,
		`let jn=[...Pt,...Jn],Ar=!!d.fastMode;`,
		`if(Yn.fastMode)cv="fast";`,
		`function _re(e){return"Codex priority"}`,
		`function Dy(S,P,T){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,n)/2000`,
		`function _H(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function ch(){return!!_H()}`,
		`function e(){return!0}`,
		`get isHidden(){return!1}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	assertClaude233Picker(t, got)

	for _, target := range claude252RequiredTargets() {
		t.Run("missing/"+target, func(t *testing.T) {
			fixture := string(claude252PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_252(broken, "0.3.15", "2.1.252", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func TestClaude252LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := claude252PatchFixture(t)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_252(data, strings.Repeat("x", 4000), "2.1.252") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude252LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := claude252PatchFixture(t)
	if !patchLogoDisplayDataFunction_2_1_252(data, "0.3.15", "2.1.252") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`r.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude252Transformations(version string) []claude227Transformation {
	return []claude227Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_252(data, version, "2.1.252") }},
		{"active-header-brand", patchActiveHeaderBrand_2_1_251},
		{"default-tier-label", patchDefaultTierLabel_2_1_251},
		{"whats-new", patchWhatsNewFeedFunction_2_1_251},
		{"usage", patchUsageFetchFunction_2_1_252},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_252(data, modelconfig.Default()) }},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_252},
		{"model-selection", patchModelPickerSelectionValue_2_1_252},
		{"agent-model-validator", patchAgentModelValidator_2_1_252},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_252},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_251},
		{"fast-mode-pricing", patchFastModePricing_2_1_251},
		{"context-warning", patchContextWarningHint_2_1_251},
		{"resume-hints", patchResumeCommandHints_2_1_251},
		{"compact-progress", patchCompactProgressCurve_2_1_251},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_252},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude252UIBrandingReplacements)
		}},
	}
}

func claude252RequiredTargets() []string {
	return []string{
		"function ARe(){let o=a.DEMO_VERSION??",
		claude251ActiveHeaderBrandTarget,
		"Default (recommended)",
		"var W=async(i,n)=>{try{",
		"async function dI(e,{atWall:r=!1}={}){",
		"function tn(e=!1){",
		"function iwe(e=!1,o=null){",
		"function ln(e,o){let t=tn(e),",
		`Be=ye??n,Ve=Be===null?T:Smt(q,Be)??Be,`,
		`defaultValue:Ve,selectedValue:Ve,`,
		`Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:Smt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:wC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])`,
		"function Smt(e,o){if(e.some((r)=>r.value===o))return o;",
		`model:oe(["sonnet","opus","haiku","fable"]).optional()`,
		`function Yr(){if(Ne()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		claude251ActiveFastModeBrandTarget,
		`function yC(){return"Opus 5"}`,
		`function CMe(){return"opus"+(YS()?"[1m]":"")}`,
		`function Her(e,t){if(!Yr())return!1;return!!e&&(sn()||Zy()||t)}`,
		`function NSt(e){if(!Yr())return!1;if(!Zy(e))return!1;if(!lf(e))return!1;return DSn(Je())}`,
		`function DSn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`,
		`function lf(e){if(!Yr())return!1;`,
		`function dw(e,t){if(sn()){if(e===null)return!!t;return!!t&&lf(e)}if(!lf(e))return!1;return!!t||NSt(e)}`,
		`...Yr()&&{fastMode:NSt(et??null)}`,
		`...Oe.gates.fastModeEnabled&&{fastMode:Ke.options.fastMode}`,
		`u={model:r.model,...Yr()&&{fastMode:r.fastMode}}`,
		`...Yr()&&{fastMode:t.fastMode}`,
		`...Yr()&&{fastMode:Ar}`,
		`...Yr()?{fastMode:Ar}:!1`,
		`let jn=[...Pt,...Jn],Ar=Yr()&&Zy()&&!R3()&&lf(fe)&&!!d.fastMode;`,
		`if(Yr()&&Zy()&&!R3()&&lf(fe)&&!!Yn.fastMode)cv="fast";`,
		"function _re(e){return`${Ck(e.inputTokens)}/${Ck(e.outputTokens)} per Mtok`}",
		"function Dy(S,P,T){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \\xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		`function Cn(n){let i=Math.max(0,n)/1000,l=1-Math.exp(-i/90);return Math.min(95,Math.round(l*100))}`,
		"function _H(){return}function s3(){return}",
		`function ch(){if(u())return!0;if(TEe())return!1;return!eA()&&aMe()}`,
		`function wyn(){if(u())return!0;return!TEe()&&!eA()&&h6()}`,
		`async function Tyn(){if(u())return!0;if(TEe())return!1;return h6()&&!eA()&&i()&&await Mp("tengu_ccr_bridge")}`,
		"async function h9t(){",
		`function e(){if(ch())return!0;try{return h6()&&!eA()&&!TEe()&&Fl().source==="none"&&py({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!upn.isC4EUpsellCommandEnabled()}catch{return!1}}`,
		`get isHidden(){return!ch()}`,
		claude252UIBrandingReplacements[0].old,
	}
}

func claude252PatchFixture(t *testing.T) []byte {
	t.Helper()
	data := claude251PatchFixture(t)
	data = append(data, []byte(`Be=ye??n,Ve=Be===null?T:Smt(q,Be)??Be,defaultValue:Ve,selectedValue:Ve,`)...)
	replacements := []struct {
		old string
		new string
	}{
		{"function a7t(o,r,t){", "function s7t(o,r,t){"},
		{"async function uI(e,{atWall:r=!1}={}){", "async function dI(e,{atWall:r=!1}={}){"},
		{"var bDe=", "var TDe="},
		{`Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:fmt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:bC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])`, `Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:Smt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:wC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])`},
		{"function fmt(e,o){if(e.some((r)=>r.value===o))return o;", "function Smt(e,o){if(e.some((r)=>r.value===o))return o;"},
		{`model:ie(["sonnet","opus","haiku","fable"]).optional()`, `model:oe(["sonnet","opus","haiku","fable"]).optional()`},
		{`function _C(){return"Opus 5"}`, `function yC(){return"Opus 5"}`},
		{`function ker(e,t){if(!Yr())return!1;return!!e&&(sn()||Zy()||t)}`, `function Her(e,t){if(!Yr())return!1;return!!e&&(sn()||Zy()||t)}`},
		{`function NSt(e){if(!Yr())return!1;if(!Zy(e))return!1;if(!af(e))return!1;return OSn(Je())}`, `function NSt(e){if(!Yr())return!1;if(!Zy(e))return!1;if(!lf(e))return!1;return DSn(Je())}`},
		{`function OSn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function DSn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`},
		{`function af(e){if(!Yr())return!1;let t=e??eS(),r=Ot(t);if(hh(Ye(r),"fast_mode"))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function lf(e){if(!Yr())return!1;let t=e??eS(),r=Ot(t);if(hh(Ye(r),"fast_mode"))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`},
		{`function dw(e,t){if(sn()){if(e===null)return!!t;return!!t&&af(e)}if(!af(e))return!1;return!!t||NSt(e)}`, `function dw(e,t){if(sn()){if(e===null)return!!t;return!!t&&lf(e)}if(!lf(e))return!1;return!!t||NSt(e)}`},
		{`...Fe.gates.fastModeEnabled&&{fastMode:ct.options.fastMode}`, `...Oe.gates.fastModeEnabled&&{fastMode:Ke.options.fastMode}`},
		{strings.Repeat(`...Yr()&&{fastMode:xr}`+"\x00", 2), strings.Repeat(`...Yr()&&{fastMode:Ar}`+"\x00", 2)},
		{`...Yr()?{fastMode:xr}:!1`, `...Yr()?{fastMode:Ar}:!1`},
		{`let Yn=[...At,...mr],xr=Yr()&&Zy()&&!v3()&&af(fe)&&!!d.fastMode;`, `let jn=[...Pt,...Jn],Ar=Yr()&&Zy()&&!R3()&&lf(fe)&&!!d.fastMode;`},
		{`if(Yr()&&Zy()&&!v3()&&af(fe)&&!!sr.fastMode)lv="fast";`, `if(Yr()&&Zy()&&!R3()&&lf(fe)&&!!Yn.fastMode)cv="fast";`},
		{`function gH(){return}function s3(){return}function __(){let e=gH();if(e!==void 0)return e;if(!pr()||!Tt())return;return Yt()?.accessToken}async function cC(e){if(!(O()&&e!==void 0))return __();let r=gH();if(r!==void 0)return r;if(!pr()||!await Ore(e))return;return(await qa(e))?.accessToken}function Kue(){return s3()??zt().BASE_API_URL}function Dne(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return ayr(e)||"remote-control"}function ayr(e){}`, `function _H(){return}function s3(){return}function __(){let e=_H();if(e!==void 0)return e;if(!pr()||!Tt())return;return Yt()?.accessToken}async function uC(e){if(!(O()&&e!==void 0))return __();let r=_H();if(r!==void 0)return r;if(!pr()||!await Ore(e))return;return(await qa(e))?.accessToken}function Yue(){return s3()??zt().BASE_API_URL}function Lne(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return N_r(e)||"remote-control"}function N_r(e){}`},
		{`function Tyn(){if(u())return!0;return!TEe()&&!eA()&&g6()}`, `function wyn(){if(u())return!0;return!TEe()&&!eA()&&h6()}`},
		{`async function Eyn(){if(u())return!0;if(TEe())return!1;return g6()&&!eA()&&i()&&await Lp("tengu_ccr_bridge")}`, `async function Tyn(){if(u())return!0;if(TEe())return!1;return h6()&&!eA()&&i()&&await Mp("tengu_ccr_bridge")}`},
		{"async function _9t(){", "async function h9t(){"},
		{"function Xvr(){", "function Yvr(){"},
		{`function e(){if(ch())return!0;try{return g6()&&!eA()&&!TEe()&&Nl().source==="none"&&py({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!dpn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){if(ch())return!0;try{return h6()&&!eA()&&!TEe()&&Fl().source==="none"&&py({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!upn.isC4EUpsellCommandEnabled()}catch{return!1}}`},
	}
	for _, replacement := range replacements {
		if !bytes.Contains(data, []byte(replacement.old)) {
			t.Fatalf("Claude 2.1.251 base fixture missing migration target %q", replacement.old)
		}
		data = []byte(strings.Replace(string(data), replacement.old, replacement.new, 1))
	}
	selectionStart := bytes.Index(data, []byte("function Smt(e,o){if(e.some((r)=>r.value===o))return o;"))
	selectionEndMarker := []byte("function Ce(){}")
	if selectionStart < 0 {
		t.Fatal("Claude 2.1.252 fixture missing model-selection start")
	}
	selectionEnd := bytes.Index(data[selectionStart:], selectionEndMarker)
	if selectionEnd < 0 {
		t.Fatal("Claude 2.1.252 fixture missing model-selection span")
	}
	selectionEnd += selectionStart + len(selectionEndMarker)
	selectionBlock := append([]byte(nil), data[selectionStart:selectionEnd]...)
	data = append(data[:selectionStart], data[selectionEnd:]...)
	extraOptionsEnd := bytes.Index(data, []byte("function npn(e){}"))
	if extraOptionsEnd < 0 {
		t.Fatal("Claude 2.1.252 fixture missing model-extra-options end marker")
	}
	reordered := make([]byte, 0, len(data)+len(selectionBlock))
	reordered = append(reordered, data[:extraOptionsEnd]...)
	reordered = append(reordered, selectionBlock...)
	reordered = append(reordered, data[extraOptionsEnd:]...)
	data = reordered
	return data
}
