package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_154 = claudeUIPatchSpec{
	Version: "2.1.154",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "bc9881b107d7be1743c64c8b72dd66798f5d0947dbc48ed0d77964c473661fd4",
	Apply:   applyClaudeUIPatches_2_1_154,
}

func applyClaudeUIPatches_2_1_154(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_154(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_154(data)
	changed = usagePatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_154(data)
	changed = whatsNewPatched || changed
	defaultDescriptionPatched := patchDefaultModelDescriptionFunction_2_1_154(data)
	changed = defaultDescriptionPatched || changed
	fastFooterPatched := patchFastModeModelPickerFooter_2_1_154(data)
	changed = fastFooterPatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_154(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_154(data)
	changed = resumeHintsPatched || changed
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
	changed = replaceAllFixed(data, "Best for everyday tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed
	changed = replaceAllFixed(data, "Fast mode for Claude Code uses Claude Opus with faster output (it does not downgrade to a smaller model). It can be toggled with /fast and is available on Opus 4.8/4.7/4.6.", "Fast mode for Claudodex uses the selected Codex-backed Opus route with faster output. It can be toggled with /fast.") || changed
	changed = replaceAllFixed(data, "Use /fast to turn on Fast mode (Opus 4.8).", "Use /fast to toggle Fast mode.") || changed

	changed = replaceFirstFixed(data, `r=kN6?Y4.createElement(kN6.Title,null):Y4.createElement(V,{bold:!0},"Claude Code")`, `r=Y4.createElement(V,{bold:!0},"Claudodex")`) || changed
	changed = replaceAllPatternString(data, `Y4.createElement(V,{bold:!0},"Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Pq("claude",U)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Pq("claude",U)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Pq("claude",d)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Pq("claude",d)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	if !versionPatched {
		changed = replaceFirstFixed(data, "Pq(\"inactive\",d)(`v${h}`)", quotedVersion(claudodexVersion)) || changed
		changed = replaceFirstFixed(data, `Y4.createElement(V,{dimColor:!0},"v",E)`, quotedVersion(claudodexVersion)) || changed
	}
	changed = replaceFirstFixed(data, "w_=h4()?", "w_=0?") || changed
	modelPickerPatched := patchMaxModelPickerBase_2_1_154(data)
	changed = modelPickerPatched || changed
	modelPickerSelectionPatched := patchModelPickerSelectionValue_2_1_154(data)
	changed = modelPickerSelectionPatched || changed
	modelPickerOptionsPatched := patchModelPickerOptions_2_1_154(data)
	changed = modelPickerOptionsPatched || changed
	if !versionPatched || !usagePatched || !whatsNewPatched || !defaultDescriptionPatched || !fastFooterPatched || !contextWarningHintPatched || !resumeHintsPatched || !modelPickerPatched || !modelPickerSelectionPatched || !modelPickerOptionsPatched {
		return false
	}
	return changed
}

func patchUsageFetchFunction_2_1_154(data []byte) bool {
	start := bytes.Index(data, []byte("async function WXH(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var bR_=R("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function WXH(){return DK("api_usage_fetch",async()=>{let H=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),_=await fetch(H+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!_.ok)throw Error("Auth error: "+_.status);return await _.json()})}`
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

func patchLogoDisplayDataFunction_2_1_154(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function L6_(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function gmK("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function L6_(){let H=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,_=Yu6(),q=process.env.DEMO_VERSION?"/code/claude":s5(b_()),K=xH(process.env.CLAUDE_CODE_HIDE_CWD)?"":_?` + "`${q} in ${_.replace(/^https?:\\/\\//,\"\")}`" + `:q,O=Zq(),T=O!=="firstParty"?wAH[O]:Lq()?VH6():"API Usage Billing",$=i8().agent;return{version:H,cwd:K,billingType:T,agentName:$}}`
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

func patchWhatsNewFeedFunction_2_1_154(data []byte) bool {
	const old = `function qpK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function qpK(H){return{title:"Claudodex Info",lines:"Thank you for using Claudodex!|Experimental - treat it as such.|If you run into issues, please file a report at|https://github.com/bassner/claudodex/issues".split("|").map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchDefaultModelDescriptionFunction_2_1_154(data []byte) bool {
	old := `function or_(H=!1){if(pe()||RAH()||UUH()){let q=LR(),K=HJ(NP(q))??"Opus",O=H&&Pj(q);if(VP())return` + "`" + `${K} with 1M context \xB7 Most capable for complex work${O?EKH(!0,q):""}` + "`" + `;return` + "`" + `${K} \xB7 Most capable for complex work${O?EKH(!0,q):""}` + "`" + `}return` + "`" + `${HJ(NP(EN()))??"Sonnet"} \xB7 Best for everyday tasks` + "`" + `}`
	const replacement = `function or_(H=!1){return"Default Codex route \xB7 default Codex work"}`
	return replaceFirstFixed(data, old, replacement)
}

func patchFastModeModelPickerFooter_2_1_154(data []byte) bool {
	const old = `X4.createElement(V,{dimColor:!0},"Use ",X4.createElement(V,{bold:!0},"/fast")," to turn on Fast mode (",Bp(),").")`
	const replacement = `X4.createElement(V,{dimColor:!0},"Use ",X4.createElement(V,{bold:!0},"/fast")," to toggle Fast mode.")`
	return replaceFirstFixed(data, old, replacement)
}

func patchContextWarningHint_2_1_154(data []byte) bool {
	start := bytes.Index(data, []byte("function d44(H){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function NZO(H){"))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	const old = `let j=w,J=Y,M=!Gc()&&!VH_(K,O),D=!1;`
	const replacement = `let j=0,J=Y,M=!Gc()&&!VH_(K,O),D=!1;`
	return replaceFirstFixed(window, old, replacement)
}

func patchResumeCommandHints_2_1_154(data []byte) bool {
	const shutdownOld = `Resume this session with:
claude ${O}--resume ${q}
`
	const shutdownReplacement = `Resume with:
claudodex ${O}--resume ${q}
`
	shutdownPatched := replaceFirstFixed(data, shutdownOld, shutdownReplacement)

	changed := shutdownPatched
	changed = replaceAllFixed(data, " resume with: claude --resume ", " resume: claudodex --resume ") || changed
	changed = replaceAllFixed(data, "Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation") || changed
	changed = replaceAllFixed(data, "Open `claude agents` to attach to it, or stop it there first to resume here.", "Open `claudodex agents` to attach, or stop it there first to resume here.") || changed
	changed = replaceAllFixed(data, "). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", "). Use `claudodex agents` to attach, or add --fork-session to branch off a copy.") || changed
	changed = replaceAllFixed(data, "command:`cd ${AK([H.projectPath])} ${ce8()} claude --resume ${T}`", "command:`cd ${AK([H.projectPath])}; claudodex --resume ${T}`") || changed
	changed = replaceAllFixed(data, `kO.default.createElement(V,{bold:!0},"claude agents")," to attach to it, or run:"`, `kO.default.createElement(V,{bold:!0},"claudodex agents")," or run:"`) || changed
	changed = replaceAllFixed(data, `kO.default.createElement(V,null," ",$,"claude --resume ",q," --fork-session")`, `kO.default.createElement(V,null,$,"claudodex --resume ",q," --fork-session")`) || changed
	return shutdownPatched && changed
}

func patchMaxModelPickerBase_2_1_154(data []byte) bool {
	start := bytes.Index(data, []byte("function ki3(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function Vi3("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	replacement := `function ki3(H=!1){let _=[],q=qNK();if(q!==void 0)_.push(q);let K=HNK();if(K!==void 0)_.push(K);let O=TNK();if(O!==void 0)_.push(O);return _}function ZX(H){return H===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":H===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":H===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":H}`
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

func patchModelPickerSelectionValue_2_1_154(data []byte) bool {
	const old = `j=$q(),J=q===null?KN_:q,[M,D]`
	const replacement = `j=$q(),J=(q=ZX(q))??KN_,[M,D]`
	if len(replacement) != len(old) {
		return false
	}
	return replaceFirstFixed(data, old, replacement)
}

func patchModelPickerOptions_2_1_154(data []byte) bool {
	start := bytes.Index(data, []byte("function v__(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function MPH("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	replacement := `function v__(H=!1){let _=process.env,q=(O,T,$)=>({value:O,label:T,description:$});return[q("opus",_.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??_.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.5",_.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),q("sonnet",_.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??_.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.4",_.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),q("haiku",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??_.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.4-mini",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
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
