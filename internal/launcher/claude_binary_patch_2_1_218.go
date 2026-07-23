package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_218 = claudeUIPatchSpec{
	Version: "2.1.218",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "71abaff59312c9a9b6a1d818365048b42e4e95cc521a823660eded3e0880d9b7",
	Apply:   applyClaudeUIPatches_2_1_218,
}

var claude218UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude216UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_218(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude218UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_218(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_218(data)
	usagePatched := patchUsageFetchFunction_2_1_218(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_218(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_218(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_218(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_218(data)
	fastModePricingPatched := patchFastModePricing_2_1_218(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_218(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_218(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_218(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_218(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude218UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_218(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Abt(){let e=Z.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=Ugi(),r=Z.DEMO_VERSION?"/code/claude":Cd(kt()),n=Z.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=oo().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function Abt(){", "function v0a(", replacement)
}

func patchWhatsNewFeedFunction_2_1_218(data []byte) bool {
	const old = `function W0a(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function W0a(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_218(data []byte) bool {
	const replacement = `async function fke(){return Uc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function fke(){", "var FVg=", replacement)
}

func patchModelPickerOptions_2_1_218(data []byte) bool {
	const replacement = `function CDX218(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(Z.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(Z.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(Z.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function png(e=!1){let t=Z,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function png(e=!1){", "function vRe(", replacement)
}

func patchModelPickerExtraOptions_2_1_218(data []byte) bool {
	const replacement = "function hng(e){let t=png(e),r=Z.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX218(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function hng(e){", "function H9i(", replacement)
}

func patchModelPickerSelectionValue_2_1_218(data []byte) bool {
	return replaceFirstFixed(data, `JAb=Q3e===null?tJe:H9i(Z3e,Q3e)??Q3e`, `JAb=Q3e===null?tJe:CDX218(Q3e)`)
}

func patchFastModeRuntimeFunctions_2_1_218(data []byte) bool {
	mlPatched := replaceFirstFixed(data, `function ml(){if(Dn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ml(){return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	g5Patched := replaceFirstFixed(data, `function G5(){return"Opus 4.8"}`, `function G5(){return"Codex AI"}`)
	u0tPatched := replaceFirstFixed(data, `function U0t(){return"opus"+(NM()?"[1m]":"")}`, `function U0t(){return"opus"}`)
	aePatched := replaceFirstFixed(data, `function aE(e){if(!ml())return!1;let t=e??MN(),r=Si(t);if(F2(yo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function aE(e){return ml()}`)
	return mlPatched && g5Patched && u0tPatched && aePatched
}

func patchFastModePricing_2_1_218(data []byte) bool {
	return replaceFirstFixed(data, "function Zqe(e){return`${uRc(e.inputTokens)}/${uRc(e.outputTokens)} per Mtok`}", `function Zqe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_218(data []byte) bool {
	return replaceClaude208Function(data, "function yii(Fqx){", "var fST,", `function yii(Fqx){return null}`)
}

func patchResumeCommandHints_2_1_218(data []byte) bool {
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

func patchCompactProgressCurve_2_1_218(data []byte) bool {
	return replaceFirstFixed(data, `function lSp(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function lSp(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_218(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function iFe(){", "function r7r(", `function iFe(){return Z.CLAUDE_BRIDGE_OAUTH_TOKEN}function sFe(){return}function Aq(){return iFe()||ms()?.accessToken}`)
	visiblePatched := replaceFirstFixed(data, `function Tk(){if(lBo())return!0;if(I2t())return!1;return!xq()&&AOt()}`, `function Tk(){return!!Z.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function Zjs(){if(lBo())return!0;if(I2t())return!1;return K8e()&&!xq()&&O_r()&&await sq("tengu_ccr_bridge")}`, `async function Zjs(){return!I2t()&&!xq()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function cBo(){", "function tLy(", "async function cBo(){if(I2t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(xq())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
