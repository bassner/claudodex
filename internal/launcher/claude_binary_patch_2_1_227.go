package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_227 = claudeUIPatchSpec{
	Version: "2.1.227",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "7432511ba3be818e01f23f6eef8630d214a8b618451e188c3c7d61a987eef6c7",
	Apply:   applyClaudeUIPatches_2_1_227,
}

var claude227UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude226UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_227(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude227UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_227(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_227(data)
	usagePatched := patchUsageFetchFunction_2_1_227(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_227(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_227(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_227(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_227(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_227(data)
	fastModePricingPatched := patchFastModePricing_2_1_227(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_227(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_227(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_227(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_227(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude227UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_227(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function aDt(){let e=re.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=i9i(),r=re.DEMO_VERSION?"/code/claude":sf(Vt()),n=re.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=Io().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function aDt(){", "function Ucl(", replacement)
}

func patchWhatsNewFeedFunction_2_1_227(data []byte) bool {
	const old = `function lul(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function lul(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_227(data []byte) bool {
	const replacement = `async function HLe(){return Gu("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function HLe(){", `var bs_=`, replacement)
}

func patchModelPickerOptions_2_1_227(data []byte) bool {
	const replacement = `function CDX227(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(re.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(re.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(re.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function Ixy(e=!1){let t=re,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function Ixy(e=!1){", "function bAe(", replacement)
}

func patchModelPickerExtraOptions_2_1_227(data []byte) bool {
	const replacement = "function Dxy(e,t){let r=Ixy(e),n=re.ANTHROPIC_CUSTOM_MODEL_OPTION,o=CDX227(n);if(n&&o===n&&!r.some((c)=>c.value===n))r.push({value:n,label:re.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:re.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return r}"
	return replaceClaude208Function(data, "function Dxy(e,t){", "function Xhs(", replacement)
}

func patchModelPickerSelectionValue_2_1_227(data []byte) bool {
	return replaceFirstFixed(data, `kAT=cKe===null?Rlt:Xhs(uKe,cKe)??cKe`, `kAT=cKe===null?Rlt:CDX227(cKe)`)
}

func patchAgentModelValidator_2_1_227(data []byte) bool {
	return replaceFirstFixed(data, `model:Pr(["sonnet","opus","haiku","fable"]).optional()`, `model:N().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_227(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Cc(){if(Wn()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Cc(){return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function M8(){return"Opus 5"}`, `function M8(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function MBt(){return"opus"+(g2()?"[1m]":"")}`, `function MBt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function nC(e){if(!Cc())return!1;let t=e??$U(),r=ns(t);if(gU(Ro(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function nC(e){return Cc()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_227(data []byte) bool {
	return replaceFirstFixed(data, "function nQe(e){return`${Euu(e.inputTokens)}/${Euu(e.outputTokens)} per Mtok`}", `function nQe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_227(data []byte) bool {
	return replaceClaude208Function(data, "function U7v(e,t,r){", "var BHh=", `function U7v(e,t,r){return null}`)
}

func patchResumeCommandHints_2_1_227(data []byte) bool {
	return patchResumeCommandHints_2_1_223(data)
}

func patchCompactProgressCurve_2_1_227(data []byte) bool {
	return replaceFirstFixed(data, `function wNf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function wNf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_227(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function f5e(){", "function d8d(", `function f5e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}function m5e(){return}function sK(){return f5e()||Bs()?.accessToken}function Tvn(){return m5e()??ua().BASE_API_URL}function Iwt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||u8d.hostname();return d8d(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function aO(){if(X0o())return!0;if(F4t())return!1;return!yV()&&N4t()}`, `function aO(){return!!re.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function tks(){if(X0o())return!0;if(F4t())return!1;return $Ze()&&!yV()&&Pbr()&&await nj("tengu_ccr_bridge")}`, `async function tks(){return!F4t()&&!yV()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function Q0o(){", "function wt_(){", "async function Q0o(){if(F4t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(yV())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
