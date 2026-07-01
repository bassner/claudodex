package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_196 = claudeUIPatchSpec{
	Version: "2.1.196",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "6fc6e61ab7582c2bf241225ff90d9f79e91d69380cb9589fc9dedd3a30070f5a",
	Apply:   applyClaudeUIPatches_2_1_196,
}

func applyClaudeUIPatches_2_1_196(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	changed := false
	versionPatched := patchLogoDisplayDataFunction_2_1_196(data, claudodexVersion, claudeVersion)
	changed = versionPatched || changed
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_196(data)
	changed = whatsNewPatched || changed
	usagePatched := patchUsageFetchFunction_2_1_196(data)
	changed = usagePatched || changed
	modelOptionsPatched := patchModelPickerOptions_2_1_196(data)
	changed = modelOptionsPatched || changed
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_196(data)
	changed = modelExtraOptionsPatched || changed
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_196(data)
	changed = modelSelectionPatched || changed
	fastModePatched := patchFastModeRuntimeFunctions_2_1_196(data)
	changed = fastModePatched || changed
	contextWarningHintPatched := patchContextWarningHint_2_1_196(data)
	changed = contextWarningHintPatched || changed
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	changed = resumeHintsPatched || changed
	compactProgressPatched := patchCompactProgressCurve_2_1_196(data)
	changed = compactProgressPatched || changed
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_196(data)
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
	changed = replaceAllFixed(data, "best for everyday, complex tasks", "default Codex work") || changed
	changed = replaceAllFixed(data, "efficient for routine tasks", modelDescriptionPatch(modelCfg.Sonnet, "everyday coding")) || changed
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

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_196(data []byte, claudodexVersion, claudeVersion string) bool {
	start := bytes.Index(data, []byte("function yAt(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function AGl("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function yAt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=sSr(),n=process.env.DEMO_VERSION?"/code/claude":Sd(Mt()),r=Ne.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${n} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:n,o=_r(),s=o!=="firstParty"?$te[o]:Eo()?Ekn():"API Usage Billing",i=Dr().agent;return{version:e,cwd:r,billingType:s,agentName:i}}`
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

func patchWhatsNewFeedFunction_2_1_196(data []byte) bool {
	const old = `function OGl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function OGl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_196(data []byte) bool {
	start := bytes.Index(data, []byte("async function bde(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("var Cmp="))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `async function bde(){return Cl("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
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

func patchModelPickerOptions_2_1_196(data []byte) bool {
	start := bytes.Index(data, []byte("function pmp(e=!1){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function QSe("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function CDX196(e){if(e==null||e==="")return"opus";let t=String(e).replace(/(\[1m\])+$/i,"").trim();return t===process.env.ANTHROPIC_DEFAULT_FABLE_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":t===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":e}function pmp(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.5",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.4",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.4-mini",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
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

func patchModelPickerExtraOptions_2_1_196(data []byte) bool {
	start := bytes.Index(data, []byte("function fmp(e){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function VDe("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "function fmp(e){let t=pmp(e),n=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,r=CDX196(n);if(n&&r===n&&!t.some((l)=>l.value===n))t.push({value:n,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return t}"
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

func patchModelPickerSelectionValue_2_1_196(data []byte) bool {
	currentRowPatched := replaceFirstFixed(data,
		`let H=x,P;if(t[5]!==n||t[6]!==H){e:{if(n!==null&&!H.some((Ee)=>Ee.value===n)&&Ua(n)){let Ee={value:n,label:O9(n),description:"Current model"},we=H.findIndex(Z1m);if(we===-1){P=[...H,Ee];break e}P=[...H.slice(0,we),Ee,...H.slice(we)];break e}P=H}t[5]=n,t[6]=H,t[7]=P}else P=t[7];let O=P,N;`,
		`let H=x,P,n2=CDX196(n);P=n2!==null&&!H.some((e)=>e.value===n2)&&Ua(n2)?[...H,{value:n2,label:O9(n2),description:"Current model"}]:H;let O=P,N;`,
	)
	selectionPatched := replaceFirstFixed(data,
		`if(t[13]!==p||t[14]!==D)M=D.some((Ee)=>Ee.value===p)?p:D[0]?.value??void 0,t[13]=p,t[14]=D,t[15]=M;else M=t[15];`,
		`if(t[13]!==p||t[14]!==D)p=CDX196(p),M=D.some((e)=>e.value===p)?p:D[0]?.value,t[15]=M;else M=t[15];`,
	)
	return currentRowPatched && selectionPatched
}

func patchFastModeRuntimeFunctions_2_1_196(data []byte) bool {
	ucPatched := replaceFirstFixed(data, `function uc(){if(_r()!=="firstParty")return!1;return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function uc(){return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	c6Patched := replaceFirstFixed(data, `function c6(){return"Opus 4.8"}`, `function c6(){return"Codex AI"}`)
	n9Patched := replaceFirstFixed(data, `function N9e(){return"opus"+(uC()?"[1m]":"")}`, `function N9e(){return"opus"}`)
	ihPatched := replaceFirstFixed(data, `function ih(e){if(!uc())return!1;let t=e??Wv(),n=Wo(t);if(mU(io(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-7")||r.includes("opus-4-8")}`, `function ih(e){return uc()}`)
	return ucPatched && c6Patched && n9Patched && ihPatched
}

func patchRemoteControlRuntimeFunctions_2_1_196(data []byte) bool {
	tokenOverridePatched := replaceFirstFixed(data,
		`function Pme(){return}function Ome(){return}function eB(){let e=Pme();if(e!==void 0)return e;if(!kc()||!Eo())return;return Gs()?.accessToken}function j7t(){return Ome()??Fs().BASE_API_URL}`,
		`function Pme(){return Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}function Ome(){return}function eB(){return Pme()||Gs()?.accessToken}function j7t(){return Ome()??Fs().BASE_API_URL}`,
	)
	visiblePatched := replaceFirstFixed(data,
		`function Pw(){if(mdr())return!0;if(Yen())return!1;return!W2()&&$Ve()}`,
		`function Pw(){return!Yen()&&!W2()}`,
	)
	enabledPatched := replaceFirstFixed(data,
		`async function zGo(){if(mdr())return!0;if(Yen())return!1;return zen()&&!W2()&&aRt()&&await UU("tengu_ccr_bridge")}`,
		`async function zGo(){return!Yen()&&!W2()&&!!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
	)
	start := bytes.Index(data, []byte("async function klr(){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function ddf()"))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := "async function klr(){if(Yen())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(W2())return\"Remote Control is not available inside a cloud session.\";if(!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}"
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

func patchContextWarningHint_2_1_196(data []byte) bool {
	start := bytes.Index(data, []byte("function Dda(e,t,n){"))
	if start < 0 {
		return false
	}
	endRel := bytes.Index(data[start:], []byte("function Vlo("))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	replacement := `function Dda(e,t,n){return null}`
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

func patchResumeCommandHints_2_1_196(data []byte) bool {
	changed := false
	changed = replaceAllFixed(data, "\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ") || changed
	changed = replaceAllFixed(data, "Previous session saved \xB7 resume with: claude --resume ", "Previous session saved \xB7 resume: claudodex --resume ") || changed
	changed = replaceAllFixed(data, "Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation") || changed
	changed = replaceAllFixed(data, "Open `claude agents` to attach to it, or stop it there first to resume here.", "Open `claudodex agents` to attach, or stop it there first to resume here.") || changed
	changed = replaceAllFixed(data, "). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", "). Use `claudodex agents` to attach, or add --fork-session to branch off a copy.") || changed
	return changed
}

func patchCompactProgressCurve_2_1_196(data []byte) bool {
	const old = `function Ptl(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	const replacement = `function Ptl(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`
	return replaceFirstFixed(data, old, replacement)
}
