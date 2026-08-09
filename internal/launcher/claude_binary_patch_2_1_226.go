package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_226 = claudeUIPatchSpec{
	Version: "2.1.226",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "013a1cf17df5ff1dcc189d5d6fd3fdd5f097ddc3cd41aa9992e99805574febbe",
	Apply:   applyClaudeUIPatches_2_1_226,
}

var claude226UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude223UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_226(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude226UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_226(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_226(data)
	usagePatched := patchUsageFetchFunction_2_1_226(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_226(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_226(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_226(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_226(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_226(data)
	fastModePricingPatched := patchFastModePricing_2_1_226(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_226(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_226(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_226(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_226(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude226UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_226(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function UIt(){let e=te.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=oji(),r=te.DEMO_VERSION?"/code/claude":Wp(Vt()),n=te.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=Ao().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function UIt(){", "function opl(", replacement)
}

func patchWhatsNewFeedFunction_2_1_226(data []byte) bool {
	const old = `function Cpl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Cpl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_226(data []byte) bool {
	const replacement = `async function YHe(){return $u("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function YHe(){", `var eey=`, replacement)
}

func patchModelPickerOptions_2_1_226(data []byte) bool {
	const replacement = `function CDX226(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(te.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(te.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(te.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function Hw_(e=!1){let t=te,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function Hw_(e=!1){", "function Kwe(", replacement)
}

func patchModelPickerExtraOptions_2_1_226(data []byte) bool {
	const replacement = "function Nw_(e){let t=Hw_(e),r=te.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX226(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:te.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:te.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function Nw_(e){", "function F_s(", replacement)
}

func patchModelPickerSelectionValue_2_1_226(data []byte) bool {
	return replaceFirstFixed(data, `ZhT=BGe===null?Tst:F_s(UGe,BGe)??BGe`, `ZhT=BGe===null?Tst:CDX226(BGe)`)
}

func patchAgentModelValidator_2_1_226(data []byte) bool {
	return replaceFirstFixed(data, `model:Nr(["sonnet","opus","haiku","fable"]).optional()`, `model:N().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_226(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function gc(){if(Kn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function gc(){return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function SG(){return"Opus 5"}`, `function SG(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function nFt(){return"opus"+(fF()?"[1m]":"")}`, `function nFt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function vw(e){if(!gc())return!1;let t=e??xB(),r=ns(t);if(lB(Eo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function vw(e){return gc()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_226(data []byte) bool {
	return replaceFirstFixed(data, "function UYe(e){return`${dpu(e.inputTokens)}/${dpu(e.outputTokens)} per Mtok`}", `function UYe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_226(data []byte) bool {
	return replaceClaude208Function(data, "function q9v(e,t,r){", "var qxh=", `function q9v(e,t,r){return null}`)
}

func patchResumeCommandHints_2_1_226(data []byte) bool {
	return patchResumeCommandHints_2_1_223(data)
}

func patchCompactProgressCurve_2_1_226(data []byte) bool {
	return replaceFirstFixed(data, `function qHf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function qHf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_226(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function eje(){", "function oWd(", `function eje(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}function tje(){return}function F8(){return eje()||Ls()?.accessToken}function PSn(){return tje()??sa().BASE_API_URL}function uvt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||nWd.hostname();return oWd(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function oP(){if(Wxo())return!0;if(BUt())return!1;return!i8()&&FUt()}`, `function oP(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function aOs(){if(Wxo())return!0;if(BUt())return!1;return KXe()&&!i8()&&byr()&&await p3("tengu_ccr_bridge")}`, `async function aOs(){return!BUt()&&!i8()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function Gxo(){", "function Mty(){", "async function Gxo(){if(BUt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(i8())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
