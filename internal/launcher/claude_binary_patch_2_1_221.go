package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_221 = claudeUIPatchSpec{
	Version: "2.1.221",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "7a181f36ed0fc4fbac6cee4ecf2b615eff93d8b434221fff5d7c878dc5ebf380",
	Apply:   applyClaudeUIPatches_2_1_221,
}

var claude221UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude220UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_221(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude221UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_221(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_221(data)
	usagePatched := patchUsageFetchFunction_2_1_221(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_221(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_221(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_221(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_221(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_221(data)
	fastModePricingPatched := patchFastModePricing_2_1_221(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_221(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_221(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_221(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_221(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude221UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_221(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function AAt(){let e=re.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=CHi(),r=re.DEMO_VERSION?"/code/claude":np(Lt()),n=re.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=lo().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function AAt(){", "function Fza(", replacement)
}

func patchWhatsNewFeedFunction_2_1_221(data []byte) bool {
	const old = `function sVa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function sVa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_221(data []byte) bool {
	const replacement = `async function qPe(){return vu("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function qPe(){", `var rB_=`, replacement)
}

func patchModelPickerOptions_2_1_221(data []byte) bool {
	const replacement = `function CDX221(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(re.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(re.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(re.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function xGg(e=!1){let t=re,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function xGg(e=!1){", "function wTe(", replacement)
}

func patchModelPickerExtraOptions_2_1_221(data []byte) bool {
	const replacement = "function OGg(e){let t=xGg(e),r=re.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX221(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:re.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:re.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function OGg(e){", "function $rs(", replacement)
}

func patchModelPickerSelectionValue_2_1_221(data []byte) bool {
	return replaceFirstFixed(data, `NxS=Mje===null?Wtt:$rs(Lje,Mje)??Mje`, `NxS=Mje===null?Wtt:CDX221(Mje)`)
}

func patchAgentModelValidator_2_1_221(data []byte) bool {
	return replaceFirstFixed(data, `model:E.enum(["sonnet","opus","haiku","fable"]).optional()`, `model:E.string().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_221(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Wl(){if(Ln()!=="firstParty")return!1;return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Wl(){return!re.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function LW(){return"Opus 5"}`, `function LW(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function DHt(){return"opus"+(GN()?"[1m]":"")}`, `function DHt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function gE(e){if(!Wl())return!1;let t=e??z2(),r=Oi(t);if(R2(co(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function gE(e){return Wl()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_221(data []byte) bool {
	return replaceFirstFixed(data, "function Mze(e){return`${pzc(e.inputTokens)}/${pzc(e.outputTokens)} per Mtok`}", `function Mze(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_221(data []byte) bool {
	return replaceClaude208Function(data, "function JEi(LpP){", "var BCT,", `function JEi(LpP){return null}`)
}

func patchResumeCommandHints_2_1_221(data []byte) bool {
	required := []struct {
		old           string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", 2},
		{"Previous session saved \xB7 resume with: claude --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", 2},
		{"Open `claude agents` to attach to it, or stop it there first to resume here.", 2},
		{"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", 2},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount {
			return false
		}
	}
	if !patchResumeCommandHints_2_1_196(data) {
		return false
	}
	for _, target := range required {
		if bytes.Contains(data, []byte(target.old)) {
			return false
		}
	}
	return true
}

func patchCompactProgressCurve_2_1_221(data []byte) bool {
	return replaceFirstFixed(data, `function Nif(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function Nif(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_221(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function V4e(){", "function bhn(", `function V4e(){return re.CLAUDE_BRIDGE_OAUTH_TOKEN}function K4e(){return}function WG(){return V4e()||As()?.accessToken}function yhn(){return K4e()??Us().BASE_API_URL}`)
	visiblePatched := replaceFirstFixed(data, `function jx(){if(gJo())return!0;if(Fjt())return!1;return!pG()&&T$t()}`, `function jx(){return!!re.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function Qma(){if(gJo())return!0;if(Fjt())return!1;return kKe()&&!pG()&&kRr()&&await VW("tengu_ccr_bridge")}`, `async function Qma(){return!Fjt()&&!pG()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function _Jo(){", "function SNb(){", "async function _Jo(){if(Fjt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(pG())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
