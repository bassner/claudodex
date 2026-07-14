package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_209 = claudeUIPatchSpec{
	Version: "2.1.209",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "59d2de7f49db2f75d5c33bbb46a6b8f288ad24d40b61e30602a502bb7ddc380c",
	Apply:   applyClaudeUIPatches_2_1_209,
}

func applyClaudeUIPatches_2_1_209(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	versionPatched := patchLogoDisplayDataFunction_2_1_209(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_209(data)
	usagePatched := patchUsageFetchFunction_2_1_209(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_209(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_209(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_209(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_209(data)
	fastModePricingPatched := patchFastModePricing_2_1_209(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_209(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_209(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_209(data)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_209(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function npt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=wVo(),r=process.env.DEMO_VERSION?"/code/claude":Zd(At()),n=Se.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=Jn().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function npt(){", "function jta(", replacement)
}

func patchWhatsNewFeedFunction_2_1_209(data []byte) bool {
	const old = `function Qta(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Qta(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_209(data []byte) bool {
	const replacement = `function bwe(){return Dc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "function bwe(){", "var _Ig=", replacement)
}

func patchModelPickerOptions_2_1_209(data []byte) bool {
	const replacement = `function CDX209(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function mch(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function mch(e=!1){", "function jEe(", replacement)
}

func patchModelPickerExtraOptions_2_1_209(data []byte) bool {
	const replacement = "function ych(e){let t=mch(e),r=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX209(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function ych(e){", "function ESi(", replacement)
}

func patchModelPickerSelectionValue_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, `WWy=TNe===null?$8e:ESi(SNe,TNe)??TNe`, `WWy=TNe===null?$8e:CDX209(TNe)`)
}

func patchFastModeRuntimeFunctions_2_1_209(data []byte) bool {
	rlPatched := replaceFirstFixed(data, `function rl(){if(En()!=="firstParty")return!1;return!dt(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function rl(){return!dt(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	e9Patched := replaceFirstFixed(data, `function e9(){return"Opus 4.8"}`, `function e9(){return"Codex AI"}`)
	ybtPatched := replaceFirstFixed(data, `function Ybt(){return"opus"+(z1()?"[1m]":"")}`, `function Ybt(){return"opus"}`)
	wtPatched := replaceFirstFixed(data, `function WT(e){if(!rl())return!1;let t=e??QO(),r=Qo(t);if(MW(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function WT(e){return rl()}`)
	return rlPatched && e9Patched && ybtPatched && wtPatched
}

func patchFastModePricing_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, "function GUe(e){return`${IVl(e.inputTokens)}/${IVl(e.outputTokens)} per Mtok`}", `function GUe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_209(data []byte) bool {
	return replaceClaude208Function(data, "function TJi(e,t,r){", "function cZn(", `function TJi(e,t,r){return null}`)
}

func patchCompactProgressCurve_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, `function Iod(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function Iod(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_209(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function Ox(){if(PHo())return!0;if(UOt())return!1;return!n2()&&DCt()}`, `function Ox(){return!!Se.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function P7s(){if(PHo())return!0;if(UOt())return!1;return BOt()&&!n2()&&jpr()&&await Lq("tengu_ccr_bridge")}`, `async function P7s(){return!UOt()&&!n2()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function OHo(){", "function BS_(", "async function OHo(){if(UOt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(n2())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
