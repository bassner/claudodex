package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_228 = claudeUIPatchSpec{
	Version: "2.1.228",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "43484b1352cef03a08346f36ef0437755b1aad646ab9313ce187857b794b7247",
	Apply:   applyClaudeUIPatches_2_1_228,
}

var claude228UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude227UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_228(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude228UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_228(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_228(data)
	usagePatched := patchUsageFetchFunction_2_1_228(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_228(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_228(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_228(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_228(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_228(data)
	fastModePricingPatched := patchFastModePricing_2_1_228(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_228(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_228(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_228(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_228(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude228UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_228(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function kMt(){let e=X.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=PVi(),r=X.DEMO_VERSION?"/code/claude":pf(Yt()),n=X.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=Ho().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function kMt(){", "function Rbl(", replacement)
}

func patchWhatsNewFeedFunction_2_1_228(data []byte) bool {
	const old = `function Kbl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Kbl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_228(data []byte) bool {
	const replacement = `async function ZFe(){return Mu("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function ZFe(){", `var Rnb=`, replacement)
}

func patchModelPickerOptions_2_1_228(data []byte) bool {
	const replacement = `function CDX228(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(X.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(X.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(X.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function BB_(e=!1){let t=X,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function BB_(e=!1){", "function ixe(", replacement)
}

func patchModelPickerExtraOptions_2_1_228(data []byte) bool {
	const replacement = "function qB_(e,t){let r=BB_(e),n=X.ANTHROPIC_CUSTOM_MODEL_OPTION,o=CDX228(n);if(n&&o===n&&!r.some((c)=>c.value===n))r.push({value:n,label:X.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:X.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return r}"
	return replaceClaude208Function(data, "function qB_(e,t){", "function Rzs(", replacement)
}

func patchModelPickerSelectionValue_2_1_228(data []byte) bool {
	return replaceFirstFixed(data, `yjT=J7e===null?Eut:Rzs(X7e,J7e)??J7e`, `yjT=J7e===null?Eut:CDX228(J7e)`)
}

func patchAgentModelValidator_2_1_228(data []byte) bool {
	return replaceFirstFixed(data, `model:Dr(["sonnet","opus","haiku","fable"]).optional()`, `model:N().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_228(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function xc(){if(Vn()!=="firstParty")return!1;return!X.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function xc(){return!X.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function uV(){return"Opus 5"}`, `function uV(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function m4t(){return"opus"+(N2()?"[1m]":"")}`, `function m4t(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function dC(e){if(!xc())return!1;let t=e??p3(),r=as(t);if(zU(Do(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function dC(e){return xc()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_228(data []byte) bool {
	return replaceFirstFixed(data, "function UZe(e){return`${kvu(e.inputTokens)}/${kvu(e.outputTokens)} per Mtok`}", `function UZe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_228(data []byte) bool {
	return replaceClaude208Function(data, "function JgE(e,t,r){", "var bzh=", `function JgE(e,t,r){return null}`)
}

func patchResumeCommandHints_2_1_228(data []byte) bool {
	return patchResumeCommandHints_2_1_223(data)
}

func patchCompactProgressCurve_2_1_228(data []byte) bool {
	return replaceFirstFixed(data, `function iGf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function iGf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_228(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function cGe(){", "function Nap(", `function cGe(){return X.CLAUDE_BRIDGE_OAUTH_TOKEN}function uGe(){return}function X6(){return cGe()||Js()?.accessToken}function pRn(){return uGe()??Sa().BASE_API_URL}function xAt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||Lap.hostname();return Nap(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function FO(){if(OLo())return!0;if(Lzt())return!1;return!rK()&&Mzt()}`, `function FO(){return!!X.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function zBs(){if(OLo())return!0;if(Lzt())return!1;return Hrt()&&!rK()&&Fwr()&&await Ij("tengu_ccr_bridge")}`, `async function zBs(){return!Lzt()&&!rK()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function DLo(){", "function Rk_(){", "async function DLo(){if(Lzt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(rK())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
