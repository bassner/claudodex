package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_207 = claudeUIPatchSpec{
	Version: "2.1.207",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "1397a062c6889675055e3314dd956376ac51262a7734ad9e819c26975d71547a",
	Apply:   applyClaudeUIPatches_2_1_207,
}

func applyClaudeUIPatches_2_1_207(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_207(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_207(data)
	changed = whatsNewPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_207(data)
	changed = usagePatched || changed
	modelOptionsPatched := patchModelPickerOptions_2_1_207(data)
	changed = modelOptionsPatched || changed
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_207(data)
	changed = modelExtraOptionsPatched || changed
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_207(data)
	changed = modelSelectionPatched || changed
	fastModePatched := patchFastModeRuntimeFunctions_2_1_207(data)
	changed = fastModePatched || changed
	fastModePricingPatched := patchFastModePricing_2_1_207(data)
	changed = fastModePricingPatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_207(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_207(data)
	changed = compactProgressPatched || changed
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_207(data)
	changed = remoteControlPatched || changed

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

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_207(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function fut(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function hGs("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function fut(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=E8o(),r=process.env.DEMO_VERSION?"/code/claude":Wd(Ct()),n=be.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=Wn().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchWhatsNewFeedFunction_2_1_207(data []byte) bool {
	const old = `function CGs(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function CGs(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_207(data []byte) bool {
	start := bytes.Index(data, []byte("async function hCe(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte(`var fhg=`))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function hCe(){return vc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerOptions_2_1_207(data []byte) bool {
	start := bytes.Index(data, []byte("function Beh(e=!1){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function XSe("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function CDX207(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function Beh(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerExtraOptions_2_1_207(data []byte) bool {
	start := bytes.Index(data, []byte("function jeh(e){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function p_i("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "function jeh(e){let t=Beh(e),r=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX207(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerSelectionValue_2_1_207(data []byte) bool {
	return replaceFirstFixed(data,
		`Nqy=wMe===null?c8e:p_i(AMe,wMe)??wMe`,
		`Nqy=wMe===null?c8e:CDX207(wMe)`,
	)
}

func patchFastModeRuntimeFunctions_2_1_207(data []byte) bool {
	ycPatched := replaceFirstFixed(data, `function yc(){if(xn()!=="firstParty")return!1;return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function yc(){return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	k4Patched := replaceFirstFixed(data, `function k4(){return"Opus 4.8"}`, `function k4(){return"Codex AI"}`)
	dtPatched := replaceFirstFixed(data, `function D_t(){return"opus"+(kR()?"[1m]":"")}`, `function D_t(){return"opus"}`)
	tyPatched := replaceFirstFixed(data, `function ty(e){if(!yc())return!1;let t=e??wO(),r=Zo(t);if(dW(ao(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function ty(e){return yc()}`)
	return ycPatched && k4Patched && dtPatched && tyPatched
}

func patchFastModePricing_2_1_207(data []byte) bool {
	return replaceFirstFixed(data,
		"function WBe(e){return`${D5l(e.inputTokens)}/${D5l(e.outputTokens)} per Mtok`}",
		`function WBe(e){return"Codex priority"}`,
	)
}

func patchRemoteControlRuntimeFunctions_2_1_207(data []byte) bool {
	visiblePatched := replaceFirstFixed(data,
		`function TH(){if(mMo())return!0;if(M1t())return!1;return!jN()&&qCt()}`,
		`function TH(){return!M1t()&&!jN()&&!!be.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
	)
	enabledPatched := replaceFirstFixed(data,
		`async function vfa(){if(mMo())return!0;if(M1t())return!1;return L1t()&&!jN()&&hhr()&&await uG("tengu_ccr_bridge")}`,
		`async function vfa(){return!M1t()&&!jN()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
	)
	start := bytes.Index(data, []byte("async function _Lo(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function xVn("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "async function _Lo(){if(M1t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(jN())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}"
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return visiblePatched && enabledPatched
}

func patchContextWarningHint_2_1_207(data []byte) bool {
	start := bytes.Index(data, []byte("function dWi(e,t,r){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function cKn("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function dWi(e,t,r){return null}`
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchCompactProgressCurve_2_1_207(data []byte) bool {
	const old = `function iZu(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`
	const replacement = `function iZu(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`
	return replaceFirstFixed(data, old, replacement)
}
