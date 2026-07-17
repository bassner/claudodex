package launcher

import (
	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_211 = claudeUIPatchSpec{
	Version: "2.1.211",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "5a728a76198b6eca7f3c7cdbff43bab44b77b48c2108f7a3107d889773382629",
	Apply:   applyClaudeUIPatches_2_1_211,
}

var claude211UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude209UIBrandingReplacements)-1)
	for _, replacement := range claude209UIBrandingReplacements {
		switch replacement.old {
		case `Approving lets Claude write to ANY file in this project without another prompt for up to 4 hours (new and changed file contents are not shown for approval). Deletes and CLAUDE.md/.claude paths still ask every time.`,
			`Approving lets Claude write to ANY file in the project "`:
			continue
		default:
			replacements = append(replacements, replacement)
		}
	}
	return append(replacements, claude209UIBrandingReplacement{
		old:           `lets Claude write to ANY file in the project "`,
		replacement:   `lets Codex write to ANY file in the project "`,
		expectedCount: 2,
	})
}()

func applyClaudeUIPatches_2_1_211(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude211UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_211(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_211(data)
	usagePatched := patchUsageFetchFunction_2_1_211(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_211(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_211(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_211(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_211(data)
	fastModePricingPatched := patchFastModePricing_2_1_211(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_211(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_211(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_211(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude211UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_211(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function rft(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=_Jo(),r=process.env.DEMO_VERSION?"/code/claude":Bd(xt()),n=ye.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=zn().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function rft(){", "function sta(", replacement)
}

func patchWhatsNewFeedFunction_2_1_211(data []byte) bool {
	const old = `function Rta(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Rta(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_211(data []byte) bool {
	const replacement = `async function hwe(){return Bc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function hwe(){", "var Jpg=", replacement)
}

func patchModelPickerOptions_2_1_211(data []byte) bool {
	const replacement = `function CDX211(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function KTh(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function KTh(e=!1){", "function Rve(", replacement)
}

func patchModelPickerExtraOptions_2_1_211(data []byte) bool {
	const replacement = "function XTh(e){let t=KTh(e),r=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX211(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function XTh(e){", "function AAi(", replacement)
}

func patchModelPickerSelectionValue_2_1_211(data []byte) bool {
	return replaceFirstFixed(data, `yk_=iFe===null?HGe:AAi(sFe,iFe)??iFe`, `yk_=iFe===null?HGe:CDX211(iFe)`)
}

func patchFastModeRuntimeFunctions_2_1_211(data []byte) bool {
	slPatched := replaceFirstFixed(data, `function sl(){if(An()!=="firstParty")return!1;return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function sl(){return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	r9Patched := replaceFirstFixed(data, `function R9(){return"Opus 4.8"}`, `function R9(){return"Codex AI"}`)
	hstPatched := replaceFirstFixed(data, `function HSt(){return"opus"+(sM()?"[1m]":"")}`, `function HSt(){return"opus"}`)
	nsPatched := replaceFirstFixed(data, `function nS(e){if(!sl())return!1;let t=e??m2(),r=ei(t);if(gG(ao(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function nS(e){return sl()}`)
	return slPatched && r9Patched && hstPatched && nsPatched
}

func patchFastModePricing_2_1_211(data []byte) bool {
	return replaceFirstFixed(data, "function X3e(e){return`${GXl(e.inputTokens)}/${GXl(e.outputTokens)} per Mtok`}", `function X3e(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_211(data []byte) bool {
	return replaceClaude208Function(data, "function iji(e,t,r){", "function CYn(", `function iji(e,t,r){return null}`)
}

func patchCompactProgressCurve_2_1_211(data []byte) bool {
	return replaceFirstFixed(data, `function nGd(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function nGd(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_211(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function TR(){if(dvo())return!0;if($Dt())return!1;return!PM()&&fAt()}`, `function TR(){return!!ye.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function rAs(){if(dvo())return!0;if($Dt())return!1;return FDt()&&!PM()&&lar()&&await hj("tengu_ccr_bridge")}`, `async function rAs(){return!$Dt()&&!PM()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function pvo(){", "function UUy(", "async function pvo(){if($Dt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(PM())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
