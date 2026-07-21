package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_216 = claudeUIPatchSpec{
	Version: "2.1.216",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "d01b49210d72ecbe277a2665d104bacccddf2d22185be99446d2929e0edfc48d",
	Apply:   applyClaudeUIPatches_2_1_216,
}

var claude216UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude212UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_216(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude216UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_216(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_216(data)
	usagePatched := patchUsageFetchFunction_2_1_216(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_216(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_216(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_216(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_216(data)
	fastModePricingPatched := patchFastModePricing_2_1_216(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_216(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_216(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_216(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_216(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude216UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_216(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function jgt(){let e=Z.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=oli(),r=Z.DEMO_VERSION?"/code/claude":qd(St()),n=Z.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=zn().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function jgt(){", "function B_a(", replacement)
}

func patchWhatsNewFeedFunction_2_1_216(data []byte) bool {
	const old = `function aya(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function aya(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_216(data []byte) bool {
	const replacement = `function NRe(){return Yc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "function NRe(){", "var qLg=", replacement)
}

func patchModelPickerOptions_2_1_216(data []byte) bool {
	const replacement = `function CDX216(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(Z.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(Z.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(Z.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function V9h(e=!1){let t=Z,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function V9h(e=!1){", "function tAe(", replacement)
}

func patchModelPickerExtraOptions_2_1_216(data []byte) bool {
	const replacement = "function Y9h(e){let t=V9h(e),r=Z.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX216(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function Y9h(e){", "function tNi(", replacement)
}

func patchModelPickerSelectionValue_2_1_216(data []byte) bool {
	return replaceFirstFixed(data, `hZy=VBe===null?j7e:tNi(zBe,VBe)??VBe`, `hZy=VBe===null?j7e:CDX216(VBe)`)
}

func patchFastModeRuntimeFunctions_2_1_216(data []byte) bool {
	elPatched := replaceFirstFixed(data, `function El(){if(Tn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function El(){return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	j6Patched := replaceFirstFixed(data, `function J6(){return"Opus 4.8"}`, `function J6(){return"Codex AI"}`)
	zwtPatched := replaceFirstFixed(data, `function Zwt(){return"opus"+(iN()?"[1m]":"")}`, `function Zwt(){return"opus"}`)
	ktPatched := replaceFirstFixed(data, `function kT(e){if(!El())return!1;let t=e??m2(),r=mi(t);if(r2(po(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function kT(e){return El()}`)
	return elPatched && j6Patched && zwtPatched && ktPatched
}

func patchFastModePricing_2_1_216(data []byte) bool {
	return replaceFirstFixed(data, "function z6e(e){return`${$hc(e.inputTokens)}/${$hc(e.outputTokens)} per Mtok`}", `function z6e(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_216(data []byte) bool {
	return replaceClaude208Function(data, "function JJo(Huk){", "var d7b", `function JJo(Huk){return null}`)
}

func patchResumeCommandHints_2_1_216(data []byte) bool {
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

func patchCompactProgressCurve_2_1_216(data []byte) bool {
	return replaceFirstFixed(data, `function Mnp(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function Mnp(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_216(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function P0(){if(VHo())return!0;if(g1t())return!1;return!Mq()&&Fkt()}`, `function P0(){return!!Z.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function I2s(){if(VHo())return!0;if(g1t())return!1;return Kje()&&!Mq()&&rfr()&&await mq("tengu_ccr_bridge")}`, `async function I2s(){return!g1t()&&!Mq()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function zHo(){", "function Ily(", "async function zHo(){if(g1t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(Mq())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
