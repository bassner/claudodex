package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_153 = claudeUIPatchSpec{
	Version: "2.1.153",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254",
	Apply:   applyClaudeUIPatches_2_1_153,
}

func applyClaudeUIPatches_2_1_153(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_153(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	changed = patchWhatsNewFeedFunction_2_1_153(data) || changed
	changed = replaceAllFixed(data, "Check the Claude Code changelog for updates", claudodexInfoLine()) || changed
	changed = replaceAllFixed(data, "What's new", "Info") || changed
	changed = replaceAllFixed(data, "Welcome back!", "Welcome back") || changed
	changed = replaceAllFixed(data, "Set the AI model for Claude Code", "Set the AI model for Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Code'll be able to read, edit, and execute files here.", "Claudodex can read, edit, and execute files here.") || changed
	changed = replaceAllFixed(data, "WARNING: Claude Code running in Bypass Permissions mode", "WARNING: Claudodex running in Bypass Permissions mode") || changed
	changed = replaceAllFixed(data, "In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.", "In Bypass Permissions mode, Claudodex will not ask for your approval before running potentially dangerous commands.") || changed
	changed = replaceAllFixed(data, "No, exit Claude Code", "No, exit Claudodex") || changed
	changed = replaceAllFixed(data, "Claude Max", "Codex Plan") || changed
	changed = replaceAllFixed(data, "Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.", "Switch between Codex-backed models. Your pick becomes the default for new sessions. For direct model names, use --model.") || changed
	changed = replaceAllFixed(data, "Select model", "Codex model") || changed
	changed = replaceAllFixed(data, "Default (recommended)", "Default (Claudodex)") || changed
	changed = replaceAllFixed(data, "Most capable for complex work", "default Codex work") || changed
	changed = replaceAllFixed(data, "Best for everyday tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed

	changed = replaceAllPatternString(data, `j4.createElement(V,{bold:!0},"Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Lq("claude",d)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `Lq("claude",d)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	if !versionPatched {
		changed = replaceFirstFixed(data, "Lq(\"inactive\",d)(`v${h}`)", quotedVersion(claudodexVersion)) || changed
		changed = replaceFirstFixed(data, `j4.createElement(V,{dimColor:!0},"v",E)`, quotedVersion(claudodexVersion)) || changed
	}
	changed = replaceFirstFixed(data, "w_=h4()?", "w_=0?") || changed
	changed = patchMaxModelPickerBase_2_1_153(data) || changed
	return changed
}

func patchLogoDisplayDataFunction_2_1_153(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function P6_(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function RuK("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function P6_(){let H=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,_=Px6(),q=process.env.DEMO_VERSION?"/code/claude":Q5(S_()),K=xH(process.env.CLAUDE_CODE_HIDE_CWD)?"":_?` + "`${q} in ${_.replace(/^https?:\\/\\//,\"\")}`" + `:q,O=vq(),T=O!=="firstParty"?eYH[O]:Zq()?zH6():"API Usage Billing",z=o8().agent;return{version:H,cwd:K,billingType:T,agentName:z}}`
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

func patchWhatsNewFeedFunction_2_1_153(data []byte) bool {
	const old = `function muK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function muK(H){return{title:"Claudodex Info",lines:"Thank you for using Claudodex!|Experimental - treat it as such.|If you run into issues, please file a report at|https://github.com/bassner/claudodex/issues".split("|").map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchMaxModelPickerBase_2_1_153(data []byte) bool {
	start := bytes.Index(data, []byte("function jl3(H=!1){"))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte("function Jl3("))
	if end < 0 {
		return false
	}
	window := data[start : start+end]
	changed := false
	for _, patch := range []struct {
		old string
		new string
	}{
		{"let z=[ML6(H)]", "let z=[]"},
		{"z.push(lkK())", "void 0"},
		{"z.push(Al3)", "void 0"},
		{"z.push(ckK())", "void 0"},
		{"return z.push(nkK),z", "return z"},
	} {
		changed = replaceFirstFixed(window, patch.old, patch.new) || changed
	}
	return changed
}
