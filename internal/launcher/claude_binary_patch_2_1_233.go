package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_233 = claudeUIPatchSpec{
	Version: "2.1.233",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "bc466b6cde63edafc773f471a1fb98787fabb31f52240c8616ce7e1f587b212d",
	Apply:   applyClaudeUIPatches_2_1_233,
}

var claude233UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude229UIBrandingReplacements...)
	for i := range replacements {
		if replacements[i].old == `Claude needs your permission` {
			replacements[i].expectedCount = 4
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_233(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude233UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_233(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_233(data)
	usagePatched := patchUsageFetchFunction_2_1_233(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_233(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_233(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_233(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_233(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_233(data)
	fastModePricingPatched := patchFastModePricing_2_1_233(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_233(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_233(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_233(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_233(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude233UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_233(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function S$t(){let e=V.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=Xcs(),r=V.DEMO_VERSION?"/code/claude":Xf(Yt()),n=V.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=Wo().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function S$t(){", "function tFl(", replacement)
}

func patchWhatsNewFeedFunction_2_1_233(data []byte) bool {
	const old = `function TFl(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function TFl(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_233(data []byte) bool {
	const replacement = `async function s4e(){return bd("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function s4e(){", `var F8b=`, replacement)
}

func patchModelPickerOptions_2_1_233(data []byte) bool {
	const replacement = `function CDX233(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(V.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(V.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(V.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function jcb(e=!1){let t=V,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function jcb(e=!1){", "function UPe(", replacement)
}

func patchModelPickerExtraOptions_2_1_233(data []byte) bool {
	const replacement = "function Wcb(e,t){let r=jcb(e),n=V.ANTHROPIC_CUSTOM_MODEL_OPTION,o=CDX233(n);if(n&&o===n&&!r.some((c)=>c.value===n))r.push({value:n,label:V.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:V.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${n})`});return r}"
	return replaceClaude208Function(data, "function Wcb(e,t){", "function YZs(", replacement)
}

func patchModelPickerSelectionValue_2_1_233(data []byte) bool {
	return replaceClaude208Function(data, "function YZs(e,t){", "function cNd(", `function YZs(e,t){return CDX233(t)}`)
}

func patchAgentModelValidator_2_1_233(data []byte) bool {
	return replaceFirstFixed(data, `model:Mr(["sonnet","opus","haiku","fable"]).optional()`, `model:F().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_233(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function ku(){if(Jn()!=="firstParty")return!1;return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ku(){return!V.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function q7(){return"Opus 5"}`, `function q7(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function OWt(){return"opus"+(wU()?"[1m]":"")}`, `function OWt(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function OC(e){if(!ku())return!1;let t=e??Z3(),r=cs(t);if(hN(zo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-8")||n.includes("opus-5")}`, `function OC(e){return ku()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_233(data []byte) bool {
	return replaceFirstFixed(data, "function Kit(e){return`${lju(e.inputTokens)}/${lju(e.outputTokens)} per Mtok`}", `function Kit(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_233(data []byte) bool {
	return replaceClaude208Function(data, "function tYi(t1M){", "var YRw,", `function tYi(t1M){return null}`)
}

func patchResumeCommandHints_2_1_233(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ", 2},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
		{"Open `claude agents` to attach to it, or stop it there first to resume here.", "Open `claudodex agents` to attach, or stop it there first to resume here.", 2},
		{"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.", "). Use `claudodex agents` to attach, or add --fork-session to branch off a copy.", 2},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount {
			return false
		}
	}
	for _, target := range required {
		if !replaceAllFixed(data, target.old, target.replacement) {
			return false
		}
	}
	for _, target := range required {
		if bytes.Contains(data, []byte(target.old)) {
			return false
		}
	}
	return true
}

func patchCompactProgressCurve_2_1_233(data []byte) bool {
	return replaceFirstFixed(data, `function HSm(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function HSm(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_233(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function sYe(){", "function cUp(", `function sYe(){return V.CLAUDE_BRIDGE_OAUTH_TOKEN}function aYe(){return}function qj(){return sYe()||aa()?.accessToken}function ltr(){return aYe()??Ua().BASE_API_URL}function nDt(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||lUp.hostname();return cUp(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function WD(){if(OFo())return!0;if(FGt())return!1;return!iY()&&NGt()}`, `function WD(){return!!V.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function p3s(){if(OFo())return!0;if(FGt())return!1;return Ust()&&!iY()&&_Ar()&&await _5("tengu_ccr_bridge")}`, `async function p3s(){return!FGt()&&!iY()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function DFo(){", "function E1_(){", "async function DFo(){if(FGt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(iY())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
