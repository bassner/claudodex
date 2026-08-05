package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_222 = claudeUIPatchSpec{
	Version: "2.1.222",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "c66a6cc6fa2e8145bb1a6e77831f2caf4b83690ff04650500dfa6e2c05ca997c",
	Apply:   applyClaudeUIPatches_2_1_222,
}

var claude222UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude221UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_222(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude222UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_222(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_222(data)
	usagePatched := patchUsageFetchFunction_2_1_222(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_222(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_222(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_222(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_222(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_222(data)
	fastModePricingPatched := patchFastModePricing_2_1_222(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_222(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_222(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_222(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_222(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude222UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_222(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function d0t(){let e=te.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=vLi(),r=te.DEMO_VERSION?"/code/claude":cp(Mt()),n=te.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=co().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function d0t(){", "function LKa(", replacement)
}

func patchWhatsNewFeedFunction_2_1_222(data []byte) bool {
	const old = `function nYa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function nYa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_222(data []byte) bool {
	const replacement = `async function UPe(){return xu("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function UPe(){", `var KA_=`, replacement)
}

func patchModelPickerOptions_2_1_222(data []byte) bool {
	const replacement = `function CDX222(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(te.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(te.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(te.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function E7g(e=!1){let t=te,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function E7g(e=!1){", "function QTe(", replacement)
}

func patchModelPickerExtraOptions_2_1_222(data []byte) bool {
	const replacement = "function A7g(e){let t=E7g(e),r=te.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX222(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:te.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:te.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function A7g(e){", "function Nos(", replacement)
}

func patchModelPickerSelectionValue_2_1_222(data []byte) bool {
	return replaceFirstFixed(data, `tMS=a5e===null?wrt:Nos(l5e,a5e)??a5e`, `tMS=a5e===null?wrt:CDX222(a5e)`)
}

func patchAgentModelValidator_2_1_222(data []byte) bool {
	return replaceFirstFixed(data, `model:E.enum(["sonnet","opus","haiku","fable"]).optional()`, `model:E.string().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_222(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Ql(){if(Ln()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Ql(){return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function QW(){return"Opus 5"}`, `function QW(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function SMt(){return"opus"+(r$()?"[1m]":"")}`, `function SMt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function EE(e){if(!Ql())return!1;let t=e??sF(),r=$i(t);if(B2(ao(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function EE(e){return Ql()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_222(data []byte) bool {
	return replaceFirstFixed(data, "function lVe(e){return`${W7c(e.inputTokens)}/${W7c(e.outputTokens)} per Mtok`}", `function lVe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_222(data []byte) bool {
	return replaceClaude208Function(data, "function iAi(BSP){", "var nIT,", `function iAi(BSP){return null}`)
}

func patchResumeCommandHints_2_1_222(data []byte) bool {
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

func patchCompactProgressCurve_2_1_222(data []byte) bool {
	return replaceFirstFixed(data, `function ucf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function ucf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_222(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function F4e(){", "function Mpn(", `function F4e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}function B4e(){return}function d8(){return F4e()||Fs()?.accessToken}function Hpn(){return B4e()??Qs().BASE_API_URL}`)
	visiblePatched := replaceFirstFixed(data, `function $x(){if(OSo())return!0;if(BNt())return!1;return!NG()&&FNt()}`, `function $x(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function Bys(){if(OSo())return!0;if(BNt())return!1;return _Ke()&&!NG()&&dpr()&&await LG("tengu_ccr_bridge")}`, `async function Bys(){return!BNt()&&!NG()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function DSo(){", "function xR_(){", "async function DSo(){if(BNt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(NG())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
