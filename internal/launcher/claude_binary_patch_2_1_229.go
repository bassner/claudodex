package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_229 = claudeUIPatchSpec{
	Version: "2.1.229",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "d732f0ba0a539c58c2ffcaef06ed03b4e523726f0cb6cc27b3a5b7e7ae0a7a21",
	Apply:   applyClaudeUIPatches_2_1_229,
}

var claude229UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude228UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_229(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude229UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_229(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_229(data)
	usagePatched := patchUsageFetchFunction_2_1_229(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_229(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_229(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_229(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_229(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_229(data)
	fastModePricingPatched := patchFastModePricing_2_1_229(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_229(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_229(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_229(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_229(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude229UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_229(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function mLt(){let e=Q.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=nJi(),r=Q.DEMO_VERSION?"/code/claude":yf(Gt()),n=Q.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=$o().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function mLt(){", "function Nwl(", replacement)
}

func patchWhatsNewFeedFunction_2_1_229(data []byte) bool {
	const old = `function oEl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function oEl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_229(data []byte) bool {
	const replacement = `async function QFe(){return wu("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function QFe(){", `var ecb=`, replacement)
}

func patchModelPickerOptions_2_1_229(data []byte) bool {
	const replacement = `function CDX229(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(Q.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(Q.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(Q.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function uL_(e=!1){let t=Q,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function uL_(e=!1){", "function lxe(", replacement)
}

func patchModelPickerExtraOptions_2_1_229(data []byte) bool {
	const replacement = "function fL_(e,t){let r=uL_(e),n=Q.ANTHROPIC_CUSTOM_MODEL_OPTION,o=CDX229(n);if(n&&o===n&&!r.some((c)=>c.value===n))r.push({value:n,label:Q.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:Q.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return r}"
	return replaceClaude208Function(data, "function fL_(e,t){", "function n5s(", replacement)
}

func patchModelPickerSelectionValue_2_1_229(data []byte) bool {
	return replaceClaude208Function(data, "function n5s(e,t){", "function qyd(", `function n5s(e,t){return CDX229(t)}`)
}

func patchAgentModelValidator_2_1_229(data []byte) bool {
	return replaceFirstFixed(data, `model:Mr(["sonnet","opus","haiku","fable"]).optional()`, `model:N().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_229(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function $c(){if(Jn()!=="firstParty")return!1;return!Q.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function $c(){return!Q.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function NV(){return"Opus 5"}`, `function NV(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function Jjt(){return"opus"+(s$()?"[1m]":"")}`, `function Jjt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function T0(e){if(!$c())return!1;let t=e??O3(),r=ls(t);if(y2(Bo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function T0(e){return $c()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_229(data []byte) bool {
	return replaceFirstFixed(data, "function dtt(e){return`${lAu(e.inputTokens)}/${lAu(e.outputTokens)} per Mtok`}", `function dtt(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_229(data []byte) bool {
	return replaceClaude208Function(data, "function KAw(e,t,r){", "var wYh=", `function KAw(e,t,r){return null}`)
}

func patchResumeCommandHints_2_1_229(data []byte) bool {
	return patchResumeCommandHints_2_1_223(data)
}

func patchCompactProgressCurve_2_1_229(data []byte) bool {
	return replaceFirstFixed(data, `function pXf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function pXf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_229(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function _8e(){", "function shp(", `function _8e(){return Q.CLAUDE_BRIDGE_OAUTH_TOKEN}function b8e(){return}function S4(){return _8e()||Js()?.accessToken}function L7t(){return b8e()??Sa().BASE_API_URL}function Fkt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||ihp.hostname();return shp(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function OO(){if(qxo())return!0;if(Z5t())return!1;return!tK()&&Q5t()}`, `function OO(){return!!Q.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function jIs(){if(qxo())return!0;if(Z5t())return!1;return ort()&&!tK()&&Vvr()&&await Yj("tengu_ccr_bridge")}`, `async function jIs(){return!Z5t()&&!tK()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function Wxo(){", "function or_(){", "async function Wxo(){if(Z5t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(tK())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
