package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_195 = claudeUIPatchSpec{
	Version: "2.1.195",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "8b45adad93f336ab95f33e714494b19fd3377a494eb05c122c8677bc895876ad",
	Apply:   applyClaudeUIPatches_2_1_195,
}

func applyClaudeUIPatches_2_1_195(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_195(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_195(data)
	changed = whatsNewPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_195(data)
	changed = usagePatched || changed
	modelOptionsPatched := patchModelPickerOptions_2_1_195(data)
	changed = modelOptionsPatched || changed
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_195(data)
	changed = modelSelectionPatched || changed
	fastModePatched := patchFastModeRuntimeFunctions_2_1_195(data)
	changed = fastModePatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_195(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_195(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_195(data)
	changed = compactProgressPatched || changed

	changed = replaceAllFixed(data, "Check the Claude Code changelog for updates", claudodexInfoLine()) || changed
	changed = replaceAllFixed(data, "What's new", "Info") || changed
	changed = replaceAllFixed(data, "Welcome back!", "Welcome back") || changed
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
	changed = replaceAllFixed(data, "Best for everyday, complex tasks", "default Codex work") || changed
	changed = replaceAllFixed(data, "Efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
	changed = replaceAllFixed(data, "Fastest for quick answers", modelDescriptionPatch(modelCfg.Haiku, "quick code")) || changed
	changed = replaceAllFixed(data, ` with 1M context \xB7 `, ` via Codex model \xB7 `) || changed
	changed = replaceAllFixed(data, "/upgrade to keep using Claude Code", "/usage to inspect Codex usage") || changed
	changed = replaceAllFixed(data, "Fast mode (research preview)", "Fast mode (Codex priority)") || changed
	changed = replaceAllFixed(data, "Draws from usage credits at a higher rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "Billed as extra usage at a premium rate. Separate rate limits apply.", "Uses Codex priority service tier when available.") || changed
	changed = replaceAllFixed(data, "$10/$50 per Mtok", "Codex priority") || changed
	changed = replaceAllFixed(data, "$30/$150 per Mtok", "Codex priority") || changed
	changed = replaceAllFixed(data, "Learn more:", "Claudodex:") || changed
	changed = replaceAllFixed(data, "https://code.claude.com/docs/en/fast-mode", "https://github.com/bassner/claudodex") || changed
	changed = replaceAllFixed(data, "Opus 4.8", "Codex AI") || changed

	changed = replaceAllPatternString(data, `_b.jsx(v,{bold:!0,children:"Claude Code"})`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `xo("claude",O)("Claude Code")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `xo("claude",O)(" Claude Code ")`, "Claude Code", "Claudodex") || changed
	changed = replaceAllPatternString(data, `children:["Claude Code"," "]`, "Claude Code", "Claudodex") || changed

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelSelectionPatched || !fastModePatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_195(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function pEt(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function ejl("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function pEt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=syr(),n=process.env.DEMO_VERSION?"/code/claude":Hd(Mt()),r=Ne.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${n} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:n,o=mr(),s=o!=="firstParty"?lte[o]:To()?Ywn():"API Usage Billing",i=Pr().agent;return{version:e,cwd:r,billingType:s,agentName:i}}`
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

func patchWhatsNewFeedFunction_2_1_195(data []byte) bool {
	const old = `function djl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function djl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_195(data []byte) bool {
	start := bytes.Index(data, []byte("async function Vue(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var olp="))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function Vue(){return _l("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
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

func patchModelPickerOptions_2_1_195(data []byte) bool {
	start := bytes.Index(data, []byte("function Kap(e){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function zap("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function CDX195(e){if(e==null||e==="")return"opus";let t=String(e).replace(/(\[1m\])+$/i,"").trim();return t===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":t===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":e}function Kap(e){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return iDe([n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.5",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.4",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.4-mini",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")])}`
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

func patchModelPickerSelectionValue_2_1_195(data []byte) bool {
	const old = `d=Ao(),p=n===null?vYt:n,[m,f]=M1e.useState(p)`
	const replacement = `d=Ao(),p=CDX195(n)??vYt,[m,f]=M1e.useState(p)`
	selectionPatched := replaceFirstFixed(data, old, replacement)
	currentRowPatched := replaceFirstFixed(data, `!H.some((Dn)=>Dn.value===n)&&ka(n)`, `!H.some((Dn)=>Dn.value===m)&&ka(n)`)
	return selectionPatched && currentRowPatched
}

func patchFastModeRuntimeFunctions_2_1_195(data []byte) bool {
	scPatched := replaceFirstFixed(data, `function sc(){if(mr()!=="firstParty")return!1;return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function sc(){return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	w6Patched := replaceFirstFixed(data, `function W6(){return"Opus 4.8"}`, `function W6(){return"Codex AI"}`)
	yPatched := replaceFirstFixed(data, `function Y$e(){return"opus"+(oC()?"[1m]":"")}`, `function Y$e(){return"opus"}`)
	costLabelPatched := replaceFirstFixed(data, `f=oU(znt(!0,F)),t[1]=f`, `f="Codex",t[1]=f`)
	fastCommandCostLabelPatched := replaceFirstFixed(data, `d=oU(znt(!0,u)),p=j9o()`, `d="Codex",p=j9o()`)
	old := `function rh(e){if(!sc())return!1;let t=e??$v(),n=Ko(t);if(tU(fo(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-6")||r.includes("opus-4-7")||r.includes("opus-4-8")}`
	replacement := `function rh(e){return sc()}`
	if len(replacement) > len(old) {
		return false
	}
	rhPatched := replaceFirstFixed(data, old, replacement)
	return scPatched && w6Patched && yPatched && costLabelPatched && fastCommandCostLabelPatched && rhPatched
}

func patchContextWarningHint_2_1_195(data []byte) bool {
	start := bytes.Index(data, []byte("function Naa(e,t,n){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function Hio("))
	if endRel < 0 {
		return false
	}
	window := data[start : start+endRel]
	replacement := `function Naa(e,t,n){return null}`
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

func patchResumeCommandHints_2_1_195(data []byte) bool {
	start := bytes.Index(data, []byte("function _go(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function ygo("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function _go(){if(m3n)return;try{let e=It(),t=jh(e),n=t?JSON.stringify(t):e,r=fNa(),o=r?` + "`" + `--worktree ${r} ` + "`" + `:"";process.stdout.isTTY&&Ck()&&!n6()&&Hqt(e)&&(Abe.writeSync(1,vt.dim(` + "`" + ` Resume this session with: claudodex ${o}--resume ${n} ` + "`" + `)),m3n=!0)}catch{}}`
	if len([]byte(replacement)) > len(old) {
		return false
	}
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	shutdownPatched := true
	changed := shutdownPatched
	changed = replaceAllFixed(data, "Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation") || changed
	changed = replaceAllFixed(data, "Open `claude agents` to attach to it, or stop it there first to resume here.", "Open `claudodex agents` to attach, or stop it there first to resume here.") || changed
	changed = replaceAllFixed(data, "). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", "). Use `claudodex agents` to attach, or add --fork-session to branch off a copy.") || changed
	return shutdownPatched && changed
}

func patchCompactProgressCurve_2_1_195(data []byte) bool {
	const old = `function CXa(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	const replacement = `function CXa(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	return replaceFirstFixed(data, old, replacement)
}
