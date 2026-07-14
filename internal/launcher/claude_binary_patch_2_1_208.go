package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_208 = claudeUIPatchSpec{
	Version: "2.1.208",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "051c7f28871b158132ac03a6140f2f2ab4046b18ecc4f7a91a2ac4d54774551e",
	Apply:   applyClaudeUIPatches_2_1_208,
}

func applyClaudeUIPatches_2_1_208(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	versionPatched := patchLogoDisplayDataFunction_2_1_208(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_208(data)
	usagePatched := patchUsageFetchFunction_2_1_208(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_208(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_208(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_208(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_208(data)
	fastModePricingPatched := patchFastModePricing_2_1_208(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_208(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_208(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_208(data)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched {
		return false
	}
	return changed
}

func applyClaudeUIFixedReplacements_2_1_208(data []byte, modelCfg modelconfig.Config) bool {
	changed := false
	changed = replaceAllFixed(data, "Check the Claude Code changelog for updates", claudodexInfoLine()) || changed
	changed = replaceAllFixed(data, "What's new", "Info") || changed
	changed = replaceAllFixed(data, "Welcome back!", "Welcome back") || changed
	changed = replaceAllFixed(data, "Set the AI model for Claude Code", "Set the AI model for Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Code will be able to read files in this directory and make edits when auto-accept edits is on.", "Claudodex can read files here and edit when auto-accept edits is on.") || changed
	changed = replaceAllFixed(data, "WARNING: Claude Code running in Bypass Permissions mode", "WARNING: Claudodex running in Bypass Permissions mode") || changed
	changed = replaceAllFixed(data, "In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.", "In Bypass Permissions mode, Claudodex will not ask for your approval before running potentially dangerous commands.") || changed
	changed = replaceAllFixed(data, "No, exit Claude Code", "No, exit Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Max", "Codex Plan") || changed
	changed = replaceAllFixed(data, "Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.", "Switch between Codex-backed models. Your pick becomes the default for new sessions. For direct model names, use --model.") || changed
	changed = replaceAllFixed(data, "Select model", "Codex model") || changed
	changed = replaceAllFixed(data, "Default (recommended)", "Default (Claudodex)") || changed
	changed = replaceAllFixed(data, "Best for everyday, complex tasks", "default Codex work") || changed
	changed = replaceAllFixed(data, "best for everyday, complex tasks", "default Codex work") || changed
	changed = replaceAllFixed(data, "Efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday")) || changed
	changed = replaceAllFixed(data, "efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed
	changed = replaceAllFixed(data, "/upgrade to keep using Claude Code", "/usage to inspect Codex usage") || changed
	changed = replaceAllFixed(data, "Fast mode (research preview)", "Fast mode (Codex priority)") || changed
	changed = replaceAllFixed(data, "Draws from usage credits at a higher rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Billed as extra usage at a premium rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Draws from usage credits", "Codex priority") || changed
	changed = replaceAllFixed(data, "Requires usage credits", "Codex priority") || changed
	changed = replaceAllFixed(data, "Learn more:", "Claudodex:") || changed
	changed = replaceAllFixed(data, "https://code.claude.com/docs/en/fast-mode", "https://github.com/bassner/claudodex") || changed
	changed = replaceAllFixed(data, "Opus 4.8", "Codex AI") || changed
	changed = replaceAllPatternString(data, `children:"Claude Code"`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	return changed
}

func replaceClaude208Function(data []byte, startMarker, endMarker, replacement string) bool {
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte(endMarker))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchLogoDisplayDataFunction_2_1_208(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Zdt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=mVo(),r=process.env.DEMO_VERSION?"/code/claude":Zd(Ct()),n=Se.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=Kn().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function Zdt(){", "function Ata(", replacement)
}

func patchWhatsNewFeedFunction_2_1_208(data []byte) bool {
	const old = `function Lta(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Lta(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_208(data []byte) bool {
	const replacement = `function _we(){return Pc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "function _we(){", "var $kg=", replacement)
}

func patchModelPickerOptions_2_1_208(data []byte) bool {
	const replacement = `function CDX208(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function Llh(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function Llh(e=!1){", "function qEe(", replacement)
}

func patchModelPickerExtraOptions_2_1_208(data []byte) bool {
	const replacement = "function Flh(e){let t=Llh(e),r=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX208(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function Flh(e){", "function dSi(", replacement)
}

func patchModelPickerSelectionValue_2_1_208(data []byte) bool {
	return replaceFirstFixed(data, `mWy=yNe===null?N8e:dSi(_Ne,yNe)??yNe`, `mWy=yNe===null?N8e:CDX208(yNe)`)
}

func patchFastModeRuntimeFunctions_2_1_208(data []byte) bool {
	tlPatched := replaceFirstFixed(data, `function tl(){if(vn()!=="firstParty")return!1;return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function tl(){return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	e9Patched := replaceFirstFixed(data, `function e9(){return"Opus 4.8"}`, `function e9(){return"Codex AI"}`)
	wbtPatched := replaceFirstFixed(data, `function Wbt(){return"opus"+(z1()?"[1m]":"")}`, `function Wbt(){return"opus"}`)
	jtPatched := replaceFirstFixed(data, `function jT(e){if(!tl())return!1;let t=e??QO(),r=Zo(t);if(NW(co(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function jT(e){return tl()}`)
	return tlPatched && e9Patched && wbtPatched && jtPatched
}

func patchFastModePricing_2_1_208(data []byte) bool {
	return replaceFirstFixed(data, "function qUe(e){return`${uVl(e.inputTokens)}/${uVl(e.outputTokens)} per Mtok`}", `function qUe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_208(data []byte) bool {
	return replaceClaude208Function(data, "function iJi(e,t,r){", "function QQn(", `function iJi(e,t,r){return null}`)
}

func patchCompactProgressCurve_2_1_208(data []byte) bool {
	return replaceFirstFixed(data, `function cod(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function cod(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_208(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function RCt(){return OOt()&&Opr()&&Qe("tengu_ccr_bridge",!1)}`, `function RCt(){return!!Se.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `function g7s(){if(EHo())return!0;if(LOt())return!1;return OOt()&&!r2()&&Opr()&&await Oq("tengu_ccr_bridge")}`, `function g7s(){return!LOt()&&!r2()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "function vHo(){", "function Bdp(", "function vHo(){if(LOt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(r2())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
