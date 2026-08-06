package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_223 = claudeUIPatchSpec{
	Version: "2.1.223",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "fcbe0b8d47570c501302dd1ad31cc26ac2810f022c45fa253936a6961dee32bf",
	Apply:   applyClaudeUIPatches_2_1_223,
}

var claude223UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude222UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_223(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude223UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_223(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_223(data)
	usagePatched := patchUsageFetchFunction_2_1_223(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_223(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_223(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_223(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_223(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_223(data)
	fastModePricingPatched := patchFastModePricing_2_1_223(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_223(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_223(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_223(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_223(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude223UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_223(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function NAt(){let e=te.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=RNi(),r=te.DEMO_VERSION?"/code/claude":mp(Mt()),n=te.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=eo().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function NAt(){", "function xXa(", replacement)
}

func patchWhatsNewFeedFunction_2_1_223(data []byte) bool {
	const old = `function JXa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function JXa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_223(data []byte) bool {
	const replacement = `async function yOe(){return Ou("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function yOe(){", `var NP_=`, replacement)
}

func patchModelPickerOptions_2_1_223(data []byte) bool {
	const replacement = `function CDX223(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(te.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(te.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(te.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function aZg(e=!1){let t=te,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function aZg(e=!1){", "function Cve(", replacement)
}

func patchModelPickerExtraOptions_2_1_223(data []byte) bool {
	const replacement = "function uZg(e){let t=aZg(e),r=te.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX223(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:te.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:te.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function uZg(e){", "function Wss(", replacement)
}

func patchModelPickerSelectionValue_2_1_223(data []byte) bool {
	return replaceFirstFixed(data, `pBS=M5e===null?Jrt:Wss(L5e,M5e)??M5e`, `pBS=M5e===null?Jrt:CDX223(M5e)`)
}

func patchAgentModelValidator_2_1_223(data []byte) bool {
	return replaceFirstFixed(data, `model:E.enum(["sonnet","opus","haiku","fable"]).optional()`, `model:E.string().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_223(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function rc(){if(Mn()!=="firstParty")return!1;return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function rc(){return!te.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function cG(){return"Opus 5"}`, `function cG(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function JMt(){return"opus"+(o$()?"[1m]":"")}`, `function JMt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function xE(e){if(!rc())return!1;let t=e??mF(),r=Bi(t);if(K2(to(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function xE(e){return rc()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_223(data []byte) bool {
	return replaceFirstFixed(data, "function $Ve(e){return`${sXc(e.inputTokens)}/${sXc(e.outputTokens)} per Mtok`}", `function $Ve(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_223(data []byte) bool {
	return replaceClaude208Function(data, "function fRi(FIP){", "var g1T,", `function fRi(FIP){return null}`)
}

func patchResumeCommandHints_2_1_223(data []byte) bool {
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

func patchCompactProgressCurve_2_1_223(data []byte) bool {
	return replaceFirstFixed(data, `function lmf(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function lmf(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_223(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function u9e(){", "function ZDd(", `function u9e(){return te.CLAUDE_BRIDGE_OAUTH_TOKEN}function d9e(){return}function E8(){return u9e()||Os()?.accessToken}function jfn(){return d9e()??Zs().BASE_API_URL}function Wfn(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||QDd.hostname();return ZDd(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function Xx(){if(mvo())return!0;if(S$t())return!1;return!zG()&&b$t()}`, `function Xx(){return!!te.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function XSs(){if(mvo())return!0;if(S$t())return!1;return jKe()&&!zG()&&efr()&&await GG("tengu_ccr_bridge")}`, `async function XSs(){return!S$t()&&!zG()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function hvo(){", "function yD_(){", "async function hvo(){if(S$t())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(zG())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
