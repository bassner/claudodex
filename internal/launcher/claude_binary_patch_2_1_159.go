package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_159 = claudeUIPatchSpec{
	Version: "2.1.159",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "5adf7b4d349f743d669cd5adf2ce76dbb5e146d8ab99b3a63c5aef2ef15595f9",
	Apply:   applyClaudeUIPatches_2_1_159,
}

func applyClaudeUIPatches_2_1_159(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_159(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_159(data)
	changed = usagePatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_159(data)
	changed = whatsNewPatched || changed
	defaultDescriptionPatched := patchDefaultModelDescriptionFunction_2_1_159(data)
	changed = defaultDescriptionPatched || changed
	fastFooterPatched := patchFastModeModelPickerFooter_2_1_159(data)
	changed = fastFooterPatched || changed
	fastNoticePatched := patchFastModeModelPickerNotice_2_1_159(data)
	changed = fastNoticePatched || changed
	fastModeRuntimePatched := patchFastModeRuntimeFunctions_2_1_159(data)
	changed = fastModeRuntimePatched || changed
	fastModePricingPatched := patchFastModePricing_2_1_159(data)
	changed = fastModePricingPatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_159(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_154(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_159(data)
	changed = compactProgressPatched || changed

	changed = replaceAllFixed(data, "Check the Claude Code changelog for updates", claudodexInfoLine()) || changed
	changed = replaceAllFixed(data, "What's new", "Info") || changed
	changed = replaceAllFixed(data, "Welcome back!", "Welcome back") || changed
	changed = replaceAllFixed(data, "Opus 4.8 is here!", "Codex backend on") || changed
	changed = replaceAllFixed(data, "Opus 4.8 is now available!", "Codex backend active") || changed
	changed = replaceAllFixed(data, "Set the AI model for Claude Code", "Set the AI model for Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Code'll be able to read, edit, and execute files here.", "Claudodex can read, edit, and execute files here.") || changed
	changed = replaceAllFixed(data, "Claude Code will be able to read files in this directory and make edits when auto-accept edits is on.", "Claudodex can read files here and edit when auto-accept edits is on.") || changed
	changed = replaceAllFixed(data, "WARNING: Claude Code running in Bypass Permissions mode", "WARNING: Claudodex running in Bypass Permissions mode") || changed
	changed = replaceAllFixed(data, "In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.", "In Bypass Permissions mode, Claudodex will not ask for your approval before running potentially dangerous commands.") || changed
	changed = replaceAllFixed(data, "No, exit Claude Code", "No, exit Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Max", "Codex Plan") || changed
	changed = replaceAllFixed(data, "Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.", "Switch between Codex-backed models. Your pick becomes the default for new sessions. For direct model names, use --model.") || changed
	changed = replaceAllFixed(data, "Select model", "Codex model") || changed
	changed = replaceAllFixed(data, "Default (recommended)", "Default (Claudodex)") || changed
	changed = replaceAllFixed(data, "Most capable for complex work", "default Codex work") || changed
	changed = replaceAllFixed(data, "Most capable for complex reasoning tasks", "default Codex reasoning tasks") || changed
	changed = replaceAllFixed(data, "Best for everyday tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed
	changed = replaceAllFixed(data, "Fast mode for Claude Code uses Claude Opus with faster output (it does not downgrade to a smaller model). It can be toggled with /fast and is available on Opus 4.8/4.7/4.6.", "Fast mode for Claudodex requests the Codex priority service tier for lower latency. It can be toggled with /fast.") || changed
	changed = replaceAllFixed(data, "Fast mode (research preview)", "Fast mode (Codex priority)") || changed
	changed = replaceAllFixed(data, "Draws from usage credits at a higher rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Billed as extra usage at a premium rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "$10/$50 per Mtok", "Codex priority") || changed
	changed = replaceAllFixed(data, "Learn more:", "Claudodex:") || changed
	changed = replaceAllFixed(data, "https://code.claude.com/docs/en/fast-mode", "https://github.com/bassner/claudodex") || changed
	changed = replaceAllFixed(data, "Use /fast to turn on Fast mode (Opus 4.8).", "Use /fast to toggle Fast mode.") || changed
	changed = replaceAllFixed(data, "Opus 4.8", "Codex AI") || changed

	changed = replaceAllPatternString(data, `y_.createElement(V,{bold:!0},"Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `jq("claude",b)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `jq("claude",b)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	if !versionPatched {
		changed = replaceFirstFixed(data, "jq(\"inactive\",b)(`v${X}`)", quotedVersion(claudodexVersion)) || changed
		changed = replaceFirstFixed(data, `y_.createElement(V,{dimColor:!0},"v",zg)`, quotedVersion(claudodexVersion)) || changed
	}

	modelPickerPatched := patchMaxModelPickerBase_2_1_159(data)
	changed = modelPickerPatched || changed
	modelPickerSelectionPatched := patchModelPickerSelectionValue_2_1_159(data)
	changed = modelPickerSelectionPatched || changed
	modelPickerOptionsPatched := patchModelPickerOptions_2_1_159(data)
	changed = modelPickerOptionsPatched || changed
	if !versionPatched || !usagePatched || !whatsNewPatched || !defaultDescriptionPatched || !fastFooterPatched || !fastNoticePatched || !fastModeRuntimePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !modelPickerPatched || !modelPickerSelectionPatched || !modelPickerOptionsPatched {
		return false
	}
	return changed
}

func patchUsageFetchFunction_2_1_159(data []byte) bool {
	start := bytes.Index(data, []byte("async function UXH(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var zL_=R("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function UXH(){return ZK("api_usage_fetch",async()=>{let H=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),_=await fetch(H+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!_.ok)throw Error("Auth error: "+_.status);return await _.json()})}`
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

func patchLogoDisplayDataFunction_2_1_159(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function K8_(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function PBK("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function K8_(){let H=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,_=nu6(),q=process.env.DEMO_VERSION?"/code/claude":G5(C_()),K=bH(process.env.CLAUDE_CODE_HIDE_CWD)?"":_?` + "`${q} in ${_.replace(/^https?:\\/\\//,\"\")}`" + `:q,O=Zq(),T=O!=="firstParty"?yAH[O]:Gq()?H_6():"API Usage Billing",$=U8().agent;return{version:H,cwd:K,billingType:T,agentName:$}}`
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

func patchWhatsNewFeedFunction_2_1_159(data []byte) bool {
	const old = `function hBK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function hBK(H){return{title:"Claudodex Info",lines:"Thank you for using Claudodex!|Experimental - treat it as such.|If you run into issues, please file a report at|https://github.com/bassner/claudodex/issues".split("|").map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchDefaultModelDescriptionFunction_2_1_159(data []byte) bool {
	old := `function xo_(H=!1){if(te()||FAH()||AFH()){let q=QZ(),K=oM(aM(q))??"Opus",O=H&&kj(q);if(yP())return` + "`" + `${K} with 1M context \xB7 Most capable for complex work${O?cKH(!0,q):""}` + "`" + `;return` + "`" + `${K} \xB7 Most capable for complex work${O?cKH(!0,q):""}` + "`" + `}return` + "`" + `${oM(aM(mN()))??"Sonnet"} \xB7 Best for everyday tasks` + "`" + `}`
	const replacement = `function xo_(H=!1){return"Default Codex route \xB7 default Codex work"}`
	return replaceFirstFixed(data, old, replacement)
}

func patchFastModeModelPickerFooter_2_1_159(data []byte) bool {
	const old = `W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Use ",W4.createElement(V,{bold:!0},"/fast")," to turn on Fast mode (",ip(),")."))`
	const replacement = `W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Use ",W4.createElement(V,{bold:!0},"/fast")," to toggle Fast mode."))`
	return replaceFirstFixed(data, old, replacement)
}

func patchFastModeModelPickerNotice_2_1_159(data []byte) bool {
	const old = `W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Fast mode is ",W4.createElement(V,{bold:!0},"ON")," and available with"," ",ip()," (/fast). Switching to other models turns off fast mode."))`
	const replacement = `W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Fast mode is ",W4.createElement(V,{bold:!0},"ON"),". Codex priority remains active while available."))`
	return replaceFirstFixed(data, old, replacement)
}

func patchFastModeRuntimeFunctions_2_1_159(data []byte) bool {
	const supportedOld = `function kj(H){if(!x4())return!1;let _=H??Z0(),K=qK(_).toLowerCase();if(ri())return K.includes("opus-4-6");return K.includes("opus-4-6")||K.includes("opus-4-7")||K.includes("opus-4-8")}`
	const supportedReplacement = `function kj(H){return x4()}`
	if len(supportedReplacement) > len(supportedOld) {
		return false
	}
	supportedPatched := replaceFirstFixed(data, supportedOld, supportedReplacement)

	const modelOld = `function ip(){return ri()?"Opus 4.6":"Opus 4.8"}`
	const modelReplacement = `function ip(){return"opus"}`
	if len(modelReplacement) > len(modelOld) {
		return false
	}
	modelPatched := replaceFirstFixed(data, modelOld, modelReplacement)
	return supportedPatched && modelPatched
}

func patchFastModePricing_2_1_159(data []byte) bool {
	const pickerOld = `let C=A7(),x=n9(C).includes("opus")?C:"claude-opus-4-8";M=Jk(gGH(!0,x)),_[1]=M`
	const pickerReplacement = `M="Codex priority",_[1]=M`
	pickerPatched := replaceFirstFixed(data, pickerOld, pickerReplacement)

	start := bytes.Index(data, []byte("async function Ah6(H,_,q,K){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var i_q=R("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function Ah6(H,_,q,K){let O=ce();if(O)return` + "`" + `Fast mode unavailable: ${O}` + "`" + `;Yh6(H,q);d("tengu_fast_mode_toggled",{enabled:H,source:K});return H?` + "`" + `${Z2H(!0)} Fast mode ON \xB7 Codex priority` + "`" + `:"Fast mode OFF"}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return pickerPatched
}

func patchContextWarningHint_2_1_159(data []byte) bool {
	start := bytes.Index(data, []byte("function h54(H){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function mGO("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	const old = `let j=w,J=Y,M=!uc()&&!K__(K,O),D=!1;`
	const replacement = `let j=0,J=Y,M=!uc()&&!K__(K,O),D=!1;`
	return replaceFirstFixed(window, old, replacement)
}

func patchMaxModelPickerBase_2_1_159(data []byte) bool {
	start := bytes.Index(data, []byte("function yr3(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function Er3("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	replacement := `function yr3(H=!1){let _=[],q=yvK();if(q!==void 0)_.push(q);let K=vvK();if(K!==void 0)_.push(K);let O=CvK();if(O!==void 0)_.push(O);return _}function ZX(H){return H===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":H===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":H===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":H}`
	if len([]byte(replacement)) > len(window) {
		return false
	}
	newBytes, ok := fitReplacement(window, replacement)
	if !ok {
		return false
	}
	copy(window, newBytes)
	return true
}

func patchModelPickerSelectionValue_2_1_159(data []byte) bool {
	const old = `j=Aq(),J=q===null?bN_:q,[M,D]`
	const replacement = `j=Aq(),J=(q=ZX(q))??bN_,[M,D]`
	if len(replacement) != len(old) {
		return false
	}
	return replaceFirstFixed(data, old, replacement)
}

func patchModelPickerOptions_2_1_159(data []byte) bool {
	start := bytes.Index(data, []byte("function $6_(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function T6_("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	replacement := `function $6_(H=!1){let _=process.env,q=(O,T,$)=>({value:O,label:T,description:$});return[q("opus",_.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??_.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",_.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),q("sonnet",_.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??_.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",_.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),q("haiku",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??_.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	if len([]byte(replacement)) > len(window) {
		return false
	}
	newBytes, ok := fitReplacement(window, replacement)
	if !ok {
		return false
	}
	copy(window, newBytes)
	return true
}

func patchCompactProgressCurve_2_1_159(data []byte) bool {
	const old = `function Ht7(H){let _=Math.max(0,H)/1000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`
	const replacement = `function Ht7(H){let _=Math.max(0,H)/2000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`
	return replaceFirstFixed(data, old, replacement)
}
