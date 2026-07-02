package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_198 = claudeUIPatchSpec{
	Version: "2.1.198",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "ab6f7ee109816ede414f7c285446633f805b623aa609f425609a64266451d61e",
	Apply:   applyClaudeUIPatches_2_1_198,
}

func applyClaudeUIPatches_2_1_198(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_198(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_198(data)
	changed = whatsNewPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_198(data)
	changed = usagePatched || changed
	modelOptionsPatched := patchModelPickerOptions_2_1_198(data)
	changed = modelOptionsPatched || changed
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_198(data)
	changed = modelExtraOptionsPatched || changed
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_198(data)
	changed = modelSelectionPatched || changed
	fastModePatched := patchFastModeRuntimeFunctions_2_1_198(data)
	changed = fastModePatched || changed
	fastModePricingPatched := patchFastModePricing_2_1_198(data)
	changed = fastModePricingPatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_198(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_198(data)
	changed = compactProgressPatched || changed
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_198(data)
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
	changed = replaceAllFixed(data, "Efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
	changed = replaceAllFixed(data, "efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
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
	changed = replaceAllFixed(data, "Fast mode for Claude Code uses Claude Opus with faster output (it does not downgrade to a smaller model). It can be toggled with /fast and is available on Opus 4.8/4.7.", "Fast mode for Claudodex requests Codex priority service tier. It can be toggled with /fast.") || changed

	changed = replaceAllPatternString(data, `children:"Claude Code"`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `(" Claude Code ")`, "Claude Code", "Claudodex") || changed

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_198(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function PCt(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function yYl("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function PCt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=qEr(),n=process.env.DEMO_VERSION?"/code/claude":Hd(Lt()),r=Oe.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${n} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:n,s="Codex Plan",i=kr().agent;return{version:e,cwd:r,billingType:s,agentName:i}}`
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

func patchWhatsNewFeedFunction_2_1_198(data []byte) bool {
	const old = `function kYl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function kYl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_198(data []byte) bool {
	start := bytes.Index(data, []byte("async function Dde(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var JTp="))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function Dde(){return Al("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
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

func patchModelPickerOptions_2_1_198(data []byte) bool {
	start := bytes.Index(data, []byte("function NTp(e=!1){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function _oe("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function CDX198(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_FABLE_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function NTp(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.5",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.4",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.4-mini",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
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

func patchModelPickerExtraOptions_2_1_198(data []byte) bool {
	start := bytes.Index(data, []byte("function UTp(e){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function uha("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "function UTp(e){let t=NTp(e),n=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,r=CDX198(n);if(n&&r===n&&!t.some((l)=>l.value===n))t.push({value:n,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return t}"
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

func patchModelPickerSelectionValue_2_1_198(data []byte) bool {
	selectionPatched := replaceFirstFixed(data,
		`g=n===null?OQt:uha(h,n)??n`,
		`g=n===null?OQt:CDX198(n)`,
	)
	focusPatched := replaceFirstFixed(data,
		`$=N.some((ve)=>ve.value===_)?_:N[0]?.value??void 0`,
		`_=CDX198(_),$=N.some((ve)=>ve.value===_)?_:N[0]?.value`,
	)
	return selectionPatched && focusPatched
}

func patchFastModeRuntimeFunctions_2_1_198(data []byte) bool {
	lcPatched := replaceFirstFixed(data, `function lc(){if(fr()!=="firstParty")return!1;return!st(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function lc(){return!st(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	v6Patched := replaceFirstFixed(data, `function v6(){return"Opus 4.8"}`, `function v6(){return"Codex AI"}`)
	t3ePatched := replaceFirstFixed(data, `function T3e(){return"opus"+(iC()?"[1m]":"")}`, `function T3e(){return"opus"}`)
	hPatched := replaceFirstFixed(data, `function _h(e){if(!lc())return!1;let t=e??Qv(),n=Uo(t);if(lB(so(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-7")||r.includes("opus-4-8")}`, `function _h(e){return lc()}`)
	return lcPatched && v6Patched && t3ePatched && hPatched
}

func patchFastModePricing_2_1_198(data []byte) bool {
	return replaceFirstFixed(data,
		"function PN(e){return`${Vdi(e.inputTokens)}/${Vdi(e.outputTokens)} per Mtok`}",
		`function PN(e){return"Codex priority"}`,
	)
}

func patchRemoteControlRuntimeFunctions_2_1_198(data []byte) bool {
	tokenOverridePatched := replaceFirstFixed(data,
		`function ife(){return}function afe(){return}function ZF(){let e=ife();if(e!==void 0)return e;if(!xc()||!So())return;return Ks()?.accessToken}function aJt(){return afe()??qs().BASE_API_URL}`,
		`function ife(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function afe(){return}function ZF(){return ife()||Ks()?.accessToken}function aJt(){return afe()??qs().BASE_API_URL}`,
	)
	visiblePatched := replaceFirstFixed(data,
		`function Uw(){if(Mmr())return!0;if(knn())return!1;return!e$()&&d7e()}`,
		`function Uw(){return!knn()&&!e$()}`,
	)
	enabledPatched := replaceFirstFixed(data,
		`async function dzo(){if(Mmr())return!0;if(knn())return!1;return xnn()&&!e$()&&Svt()&&await OB("tengu_ccr_bridge")}`,
		`async function dzo(){return!knn()&&!e$()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
	)
	start := bytes.Index(data, []byte("async function gdr(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function x_f()"))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "async function gdr(){if(knn())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(e$())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}"
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return tokenOverridePatched && visiblePatched && enabledPatched
}

func patchContextWarningHint_2_1_198(data []byte) bool {
	start := bytes.Index(data, []byte("function Mha(e,t,n){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function qdo("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function Mha(e,t,n){return null}`
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

func patchCompactProgressCurve_2_1_198(data []byte) bool {
	const old = `function Pml(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	const replacement = `function Pml(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	return replaceFirstFixed(data, old, replacement)
}
