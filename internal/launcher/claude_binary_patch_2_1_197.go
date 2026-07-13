package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_197 = claudeUIPatchSpec{
	Version: "2.1.197",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "8cc0c4d1e4eb1dca3b0cc92ab02ee3505de764e023f8c901761c167b72041fb8",
	Apply:   applyClaudeUIPatches_2_1_197,
}

func applyClaudeUIPatches_2_1_197(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_197(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_197(data)
	changed = whatsNewPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_197(data)
	changed = usagePatched || changed
	modelOptionsPatched := patchModelPickerOptions_2_1_197(data)
	changed = modelOptionsPatched || changed
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_197(data)
	changed = modelExtraOptionsPatched || changed
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_197(data)
	changed = modelSelectionPatched || changed
	fastModePatched := patchFastModeRuntimeFunctions_2_1_197(data)
	changed = fastModePatched || changed
	fastModePricingPatched := patchFastModePricing_2_1_197(data)
	changed = fastModePricingPatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_197(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_197(data)
	changed = compactProgressPatched || changed
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_197(data)
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
	changed = replaceAllFixed(data, "Efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed
	changed = replaceAllFixed(data, "/upgrade to keep using Claude Code", "/usage to inspect Codex usage") || changed
	changed = replaceAllFixed(data, "Fast mode (research preview)", "Fast mode (Codex priority)") || changed
	changed = replaceAllFixed(data, "Draws from usage credits at a higher rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Billed as extra usage at a premium rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Draws from usage credits", "Codex priority") || changed
	changed = replaceAllFixed(data, "$10/$50 per Mtok", "Codex priority") || changed
	changed = replaceAllFixed(data, "$30/$150 per Mtok", "Codex priority") || changed
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

func patchLogoDisplayDataFunction_2_1_197(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function yAt(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function OGl("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function yAt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=lSr(),n=process.env.DEMO_VERSION?"/code/claude":Ed(Mt()),r=Ne.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${n} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:n,o=_r(),s=o!=="firstParty"?$te[o]:Eo()?Akn():"API Usage Billing",i=Dr().agent;return{version:e,cwd:r,billingType:s,agentName:i}}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchWhatsNewFeedFunction_2_1_197(data []byte) bool {
	const old = `function GGl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function GGl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_197(data []byte) bool {
	start := bytes.Index(data, []byte("async function Cde(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var Lmp="))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function Cde(){return Cl("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerOptions_2_1_197(data []byte) bool {
	start := bytes.Index(data, []byte("function Amp(e=!1){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function Xre("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function CDX197(e){if(e==null||e==="")return"opus";let t=String(e).replace(/(\[1m\])+$/i,"").trim();return t===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":t===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":e}function Amp(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerExtraOptions_2_1_197(data []byte) bool {
	start := bytes.Index(data, []byte("function Rmp(e){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function yda("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "function Rmp(e){let t=Amp(e),n=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,r=CDX197(n);if(n&&r===n&&!t.some((l)=>l.value===n))t.push({value:n,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return VDe(t)}"
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerSelectionValue_2_1_197(data []byte) bool {
	return replaceFirstFixed(data,
		`g=n===null?rXt:yda(h,n)??n`,
		`g=n===null?rXt:CDX197(n)`,
	)
}

func patchFastModeRuntimeFunctions_2_1_197(data []byte) bool {
	ucPatched := replaceFirstFixed(data, `function uc(){if(_r()!=="firstParty")return!1;return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function uc(){return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	c6Patched := replaceFirstFixed(data, `function c6(){return"Opus 4.8"}`, `function c6(){return"Codex AI"}`)
	n9Patched := replaceFirstFixed(data, `function N9e(){return"opus"+(dC()?"[1m]":"")}`, `function N9e(){return"opus"}`)
	ihPatched := replaceFirstFixed(data, `function ih(e){if(!uc())return!1;let t=e??Gv(),n=Bo(t);if(hU(oo(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-7")||r.includes("opus-4-8")}`, `function ih(e){return uc()}`)
	return ucPatched && c6Patched && n9Patched && ihPatched
}

func patchFastModePricing_2_1_197(data []byte) bool {
	return replaceFirstFixed(data,
		"function IN(e){return`${jai(e.inputTokens)}/${jai(e.outputTokens)} per Mtok`}",
		`function IN(e){return"Codex priority"}`,
	)
}

func patchRemoteControlRuntimeFunctions_2_1_197(data []byte) bool {
	tokenStart := bytes.Index(data, []byte("function Mme(){return}function Nme(){return}function nB(){"))
	if tokenStart < 0 {
		return false
	}
	tokenEndRel := bytes.Index(data[tokenStart:], []byte("function G7t("))
	if tokenEndRel < 0 {
		return false
	}
	tokenOld := data[tokenStart : tokenStart+tokenEndRel]
	tokenReplacement := `function Mme(){return Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}function Nme(){return}function nB(){return Mme()||Gs()?.accessToken}function W7t(){return Nme()??Fs().BASE_API_URL}`
	if len([]byte(tokenReplacement)) > len(tokenOld) {
		return false
	}
	tokenBytes, ok := fitReplacement(tokenOld, tokenReplacement)
	if !ok {
		return false
	}
	copy(tokenOld, tokenBytes)

	visiblePatched := replaceFirstFixed(data,
		`function Pw(){if(gdr())return!0;if(Jen())return!1;return!W2()&&$Ve()}`,
		`function Pw(){return!Jen()&&!W2()}`,
	)
	enabledPatched := replaceFirstFixed(data,
		`async function eVo(){if(gdr())return!0;if(Jen())return!1;return Yen()&&!W2()&&aRt()&&await UU("tengu_ccr_bridge")}`,
		`async function eVo(){return!Jen()&&!W2()&&!!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
	)
	start := bytes.Index(data, []byte("async function Dlr(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function Edf()"))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "async function Dlr(){if(Jen())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(W2())return\"Remote Control is not available inside a cloud session.\";if(!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}"
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return visiblePatched && enabledPatched
}

func patchContextWarningHint_2_1_197(data []byte) bool {
	start := bytes.Index(data, []byte("function jda(e,t,n){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function Qlo("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function jda(e,t,n){return null}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchCompactProgressCurve_2_1_197(data []byte) bool {
	const old = `function Wtl(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	const replacement = `function Wtl(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	return replaceFirstFixed(data, old, replacement)
}
