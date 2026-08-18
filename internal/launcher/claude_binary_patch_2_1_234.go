package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_234 = claudeUIPatchSpec{
	Version: "2.1.234",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "08d8700313697cbe730a25420c908a299ce52d56f0eb2cf4fac94cab5109bc57",
	Apply:   applyClaudeUIPatches_2_1_234,
}

var claude234UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude233UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_234(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude234UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_234(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_234(data)
	usagePatched := patchUsageFetchFunction_2_1_234(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_234(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_234(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_234(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_234(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_234(data)
	fastModePricingPatched := patchFastModePricing_2_1_234(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_234(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_234(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_234(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_234(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude234UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_234(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Qjt(){let e=V.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=t_s(),r=V.DEMO_VERSION?"/code/claude":Tm(Yt()),n=V.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=Vo().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function Qjt(){", "function Cql(", replacement)
}

func patchWhatsNewFeedFunction_2_1_234(data []byte) bool {
	const old = `function Gql(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Gql(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_234(data []byte) bool {
	const replacement = `async function aje(){return _d("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function aje(){", `var PDb=`, replacement)
}

func patchModelPickerOptions_2_1_234(data []byte) bool {
	const replacement = `function CDX234(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(V.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(V.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(V.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function knb(e=!1){let t=V,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function knb(e=!1){", "function GOe(", replacement)
}

func patchModelPickerExtraOptions_2_1_234(data []byte) bool {
	const replacement = "function xnb(e,t){let r=knb(e),n=V.ANTHROPIC_CUSTOM_MODEL_OPTION,o=CDX234(n);if(n&&o===n&&!r.some((c)=>c.value===n))r.push({value:n,label:V.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:V.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return r}"
	return replaceClaude208Function(data, "function xnb(e,t){", "function $Js(", replacement)
}

func patchModelPickerSelectionValue_2_1_234(data []byte) bool {
	return replaceClaude208Function(data, "function $Js(e,t){", "function SAd(", `function $Js(e,t){return CDX234(t)}`)
}

func patchAgentModelValidator_2_1_234(data []byte) bool {
	return replaceFirstFixed(data, `model:Mr(["sonnet","opus","haiku","fable"]).optional()`, `model:F().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_234(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function ku(){if(Jn()!=="firstParty")return!1;return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ku(){return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function iJ(){return"Opus 5"}`, `function iJ(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function SKt(){return"opus"+(_4()?"[1m]":"")}`, `function SKt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function yk(e){if(!ku())return!1;let t=e??V3(),r=ys(t);if(sF(jo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function yk(e){return ku()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_234(data []byte) bool {
	return replaceFirstFixed(data, "function Llt(e){return`${eJu(e.inputTokens)}/${eJu(e.outputTokens)} per Mtok`}", `function Llt(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_234(data []byte) bool {
	return replaceClaude208Function(data, "function Ons(Bd1){", "var Y8w,", `function Ons(Bd1){return null}`)
}

func patchResumeCommandHints_2_1_234(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ", 2},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
		{"Open `claude agents` to attach to it, or stop it there first to resume here.", "Open `claudodex agents` to attach, or stop it there first to resume here.", 2},
		{"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", "). Use `claudodex agents` to attach, or add --fork-session to branch off a copy.", 2},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount {
			return false
		}
	}
	for _, target := range required {
		if !replaceAllFixed(data, target.old, target.replacement) {
			return false
		}
	}
	for _, target := range required {
		if bytes.Contains(data, []byte(target.old)) {
			return false
		}
	}
	return true
}

func patchCompactProgressCurve_2_1_234(data []byte) bool {
	return replaceFirstFixed(data, `function oHm(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function oHm(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_234(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function k1e(){", "function aYp(", `function k1e(){return V.CLAUDE_BRIDGE_OAUTH_TOKEN}function NXe(){return}function oj(){return k1e()||ua()?.accessToken}function Jor(){return NXe()??qa().BASE_API_URL}function lLt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||sYp.hostname();return aYp(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function kM(){if(Fjo())return!0;if(k7t())return!1;return!bJ()&&C7t()}`, `function kM(){return!!V.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function zGs(){if(Fjo())return!0;if(k7t())return!1;return rUe()&&!bJ()&&$Pr()&&await R6("tengu_ccr_bridge")}`, `async function zGs(){return!k7t()&&!bJ()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function $jo(){", "function N8_(){", "async function $jo(){if(k7t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(bJ())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
