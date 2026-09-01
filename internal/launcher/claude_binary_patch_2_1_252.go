package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_252 = claudeUIPatchSpec{
	Version: "2.1.252",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "b661c6a094fcc32656bf7c0071c5b45bf900b34d4f0a1ab3d78fd59aeba2c2c7",
	Apply:   applyClaudeUIPatches_2_1_252,
}

var claude252UIBrandingReplacements = claude251UIBrandingReplacements

func applyClaudeUIPatches_2_1_252(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude252UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_252(data, claudodexVersion, claudeVersion)
	headerBrandingPatched := patchActiveHeaderBrand_2_1_251(data)
	defaultTierLabelPatched := patchDefaultTierLabel_2_1_251(data)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_251(data)
	usagePatched := patchUsageFetchFunction_2_1_252(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_252(data, modelCfg)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_252(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_252(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_252(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_252(data)
	activeFastModeBrandPatched := patchActiveFastModeBrand_2_1_251(data)
	fastModePricingPatched := patchFastModePricing_2_1_251(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_251(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_251(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_251(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_252(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude252UIBrandingReplacements)

	changed := versionPatched || headerBrandingPatched || defaultTierLabelPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || activeFastModeBrandPatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !headerBrandingPatched || !defaultTierLabelPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !activeFastModeBrandPatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_252(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function ARe(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,r=Mkn(),t=a.DEMO_VERSION?"/code/claude":Fo(ee()),e=a.CLAUDE_CODE_HIDE_CWD?"":r?` + "`${t} in ${r.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",s=Je().agent;return{version:o,cwd:e,billingType:i,agentName:s}}`
	return replaceClaude208Function(data, "function ARe(){let o=a.DEMO_VERSION??", "function s7t(o,r,t){", replacement)
}

func patchUsageFetchFunction_2_1_252(data []byte) bool {
	const replacement = `async function dI(e,{atWall:r=!1}={}){return Hr(r?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let o=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=r?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",s=await fetch(o+t,{headers:{"Content-Type":"application/json"}});if(!s.ok)throw Error("Auth error: "+s.status);return await s.json()})}`
	return replaceClaude208Function(data, "async function dI(e,{atWall:r=!1}={}){", `var TDe=`, replacement)
}

func patchModelPickerOptions_2_1_252(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX252(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function tn(e=!1){let t=a,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function tn(e=!1){", "function S(e){", replacement)
}

func patchModelPickerExtraOptions_2_1_252(data []byte) bool {
	optionsPatched := replaceClaude208Function(data, "function iwe(e=!1,o=null){", "function ln(e,o){", "function iwe(e=!1,o=null){return tn(e)}")
	extraOptionsPatched := replaceClaude208Function(data, "function ln(e,o){let t=tn(e),", "function Smt(e,o){", "function ln(e,o){return tn(e)}")
	selectedOptionPatched := replaceFirstFixed(data, `Be=ye??n,Ve=Be===null?T:Smt(q,Be)??Be,`, `Be=ye??n,Ve=Be===null?T:CDX252(Be),`)
	selectedValuePatched := replaceFirstFixed(data, `defaultValue:Ve,selectedValue:Ve,`, ``)
	const sessionOptionsTarget = `Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:Smt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:wC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])`
	sessionOptionsPatched := replaceFirstFixed(data, sessionOptionsTarget, "Ho=q.slice(0,3)")
	return optionsPatched && extraOptionsPatched && selectedOptionPatched && selectedValuePatched && sessionOptionsPatched
}

func patchModelPickerSelectionValue_2_1_252(data []byte) bool {
	return replaceClaude208Function(data, "function Smt(e,o){if(e.some((r)=>r.value===o))return o;", "function Ce(){", `function Smt(e,o){let n=CDX252(o),t=e.find((r)=>r.value===n||CDX252(r.value)===n);return t?.value??n}`)
}

func patchAgentModelValidator_2_1_252(data []byte) bool {
	return replaceFirstFixed(data, `model:oe(["sonnet","opus","haiku","fable"]).optional()`, `model:i().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_252(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Yr(){if(Ne()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Yr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function yC(){return"Opus 5"}`, `function yC(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function CMe(){return"opus"+(YS()?"[1m]":"")}`, `function CMe(){return"opus"}`)
	capabilityPatched := replaceFirstFixed(data, `function Her(e,t){if(!Yr())return!1;return!!e&&(sn()||Zy()||t)}`, `function Her(e,t){return Yr()&&!!e}`)
	enabledPatched := replaceFirstFixed(data, `function NSt(e){if(!Yr())return!1;if(!Zy(e))return!1;if(!lf(e))return!1;return DSn(Je())}`, `function NSt(e){return Yr()&&(ye("flagSettings")?.fastMode===!0||DSn(Je()))}`)
	optInPatched := replaceFirstFixed(data, `function DSn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function DSn(e){return e.fastMode===!0}`)
	supportPatched := replaceFirstFixed(data, `function lf(e){if(!Yr())return!1;let t=e??eS(),r=Ot(t);if(hh(Ye(r),"fast_mode"))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function lf(e){return Yr()}`)
	statePatched := replaceFirstFixed(data, `function dw(e,t){if(sn()){if(e===null)return!!t;return!!t&&lf(e)}if(!lf(e))return!1;return!!t||NSt(e)}`, `function dw(e,t){return Yr()&&(t!==void 0?!!t:DSn(Je()))}`)
	initialStateGatePatched := replaceFirstFixed(data, `...Yr()&&{fastMode:NSt(et??null)}`, `fastMode:NSt(et??null)`)
	mainRequestGatePatched := replaceFirstFixed(data, `...Oe.gates.fastModeEnabled&&{fastMode:Ke.options.fastMode}`, `fastMode:Ke.options.fastMode`)
	retryStateGatePatched := replaceFirstFixed(data, `u={model:r.model,...Yr()&&{fastMode:r.fastMode}}`, `u={model:r.model,fastMode:r.fastMode}`)
	syncFallbackGatePatched := replaceFirstFixed(data, `...Yr()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`)
	if bytes.Count(data, []byte(`...Yr()&&{fastMode:Ar}`)) != 2 {
		return false
	}
	streamFallbackGatesPatched := replaceAllFixed(data, `...Yr()&&{fastMode:Ar}`, `fastMode:Ar`)
	streamPrimaryGatePatched := replaceFirstFixed(data, `...Yr()?{fastMode:Ar}:!1`, `fastMode:Ar`)
	initialRequestPatched := replaceFirstFixed(data, `let jn=[...Pt,...Jn],Ar=Yr()&&Zy()&&!R3()&&lf(fe)&&!!d.fastMode;`, `let jn=[...Pt,...Jn],Ar=!!d.fastMode;`)
	retryRequestPatched := replaceFirstFixed(data, `if(Yr()&&Zy()&&!R3()&&lf(fe)&&!!Yn.fastMode)cv="fast";`, `if(Yn.fastMode)cv="fast";`)
	return gatePatched && namePatched && modelPatched && capabilityPatched && enabledPatched && optInPatched && supportPatched && statePatched && initialStateGatePatched && mainRequestGatePatched && retryStateGatePatched && syncFallbackGatePatched && streamFallbackGatesPatched && streamPrimaryGatePatched && initialRequestPatched && retryRequestPatched
}

func patchRemoteControlRuntimeFunctions_2_1_252(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function _H(){return}function s3(){return}", "function N_r(e){", `function _H(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function s3(){return}function __(){return _H()||Yt()?.accessToken}async function uC(e){return __()}function Yue(){return s3()??zt().BASE_API_URL}function Lne(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return N_r(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function ch(){if(u())return!0;if(TEe())return!1;return!eA()&&aMe()}`, `function ch(){return!!_H()}`)
	availablePatched := replaceFirstFixed(data, `function wyn(){if(u())return!0;return!TEe()&&!eA()&&h6()}`, `function wyn(){return!!_H()}`)
	enabledPatched := replaceFirstFixed(data, `async function Tyn(){if(u())return!0;if(TEe())return!1;return h6()&&!eA()&&i()&&await Mp("tengu_ccr_bridge")}`, `async function Tyn(){return!TEe()&&!eA()&&!!_H()}`)
	errorPatched := replaceClaude208Function(data, "async function h9t(){", "function Yvr(){", "async function h9t(){if(TEe())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(eA())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	commandEnabledPatched := replaceFirstFixed(data, `function e(){if(ch())return!0;try{return h6()&&!eA()&&!TEe()&&Fl().source==="none"&&py({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!upn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
	commandVisiblePatched := replaceFirstFixed(data, `get isHidden(){return!ch()}`, `get isHidden(){return!1}`)
	return tokenPatched && visiblePatched && availablePatched && enabledPatched && errorPatched && commandEnabledPatched && commandVisiblePatched
}
