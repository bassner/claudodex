package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_219 = claudeUIPatchSpec{
	Version: "2.1.219",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "a8e806faaefac53c7a0f26523d8a45c60dbef3407b14ef990c75765d08febc82",
	Apply:   applyClaudeUIPatches_2_1_219,
}

var claude219UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude218UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_219(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude219UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_219(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_219(data)
	usagePatched := patchUsageFetchFunction_2_1_219(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_219(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_219(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_219(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_219(data)
	fastModePricingPatched := patchFastModePricing_2_1_219(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_219(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_219(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_219(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_219(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude219UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_219(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function kSt(){let e=Z.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=$bi(),r=Z.DEMO_VERSION?"/code/claude":Ad(xt()),n=Z.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",s=eo().agent;return{version:e,cwd:n,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function kSt(){", "function CDa(", replacement)
}

func patchWhatsNewFeedFunction_2_1_219(data []byte) bool {
	const old = `function VDa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function VDa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_219(data []byte) bool {
	const replacement = `async function $ke(){return Vc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function $ke(){", "var TZg=", replacement)
}

func patchModelPickerOptions_2_1_219(data []byte) bool {
	const replacement = `function CDX219(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(Z.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(Z.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(Z.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function Bug(e=!1){let t=Z,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function Bug(e=!1){", "function Fye(", replacement)
}

func patchModelPickerExtraOptions_2_1_219(data []byte) bool {
	const replacement = "function qug(e){let t=Bug(e),r=Z.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX219(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function qug(e){", "function Nqi(", replacement)
}

func patchModelPickerSelectionValue_2_1_219(data []byte) bool {
	return replaceFirstFixed(data, `DLb=L4e===null?XJe:Nqi(M4e,L4e)??L4e`, `DLb=L4e===null?XJe:CDX219(L4e)`)
}

func patchFastModeRuntimeFunctions_2_1_219(data []byte) bool {
	elPatched := replaceFirstFixed(data, `function El(){if(xn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function El(){return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	pqPatched := replaceFirstFixed(data, `function pq(){return"Opus 5"}`, `function pq(){return"Codex"}`)
	jktPatched := replaceFirstFixed(data, `function jkt(){return"opus"+(KM()?"[1m]":"")}`, `function jkt(){return"opus"}`)
	fEPatched := replaceFirstFixed(data, `function fE(e){if(!El())return!1;let t=e??ZN(),r=Ei(t);if(HN(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")||n.includes("opus-5")}`, `function fE(e){return El()}`)
	return elPatched && pqPatched && jktPatched && fEPatched
}

func patchFastModePricing_2_1_219(data []byte) bool {
	return replaceFirstFixed(data, "function Mje(e){return`${qIc(e.inputTokens)}/${qIc(e.outputTokens)} per Mtok`}", `function Mje(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_219(data []byte) bool {
	return replaceClaude208Function(data, "function gli(XeI){", "var fIS,", `function gli(XeI){return null}`)
}

func patchResumeCommandHints_2_1_219(data []byte) bool {
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

func patchCompactProgressCurve_2_1_219(data []byte) bool {
	return replaceFirstFixed(data, `function G0p(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function G0p(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_219(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function q2e(){", "function nYr(", `function q2e(){return Z.CLAUDE_BRIDGE_OAUTH_TOKEN}function j2e(){return}function Gq(){return q2e()||ms()?.accessToken}`)
	visiblePatched := replaceFirstFixed(data, `function bk(){if(VUo())return!0;if(KBt())return!1;return!Kq()&&MPt()}`, `function bk(){return!!Z.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function tzs(){if(VUo())return!0;if(KBt())return!1;return MGe()&&!Kq()&&fbr()&&await Aq("tengu_ccr_bridge")}`, `async function tzs(){return!KBt()&&!Kq()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function zUo(){", "function k3y(){", "async function zUo(){if(KBt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(Kq())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
