package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_212 = claudeUIPatchSpec{
	Version: "2.1.212",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "09ecba2ab2df9b6ee5b0695e26f65dea60fb3b6af3d3542ee09f466838d1e574",
	Apply:   applyClaudeUIPatches_2_1_212,
}

var claude212UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude211UIBrandingReplacements)-3)
	for _, replacement := range claude211UIBrandingReplacements {
		switch replacement.old {
		case `You can grant Claude access to additional directories without changing your current working directory.`,
			`You can hit Enter while Claude is working to queue a follow-up or steer it mid-turn \u2014 no need to wait for it to finish.`,
			`Setting a 200K auto-compact window keeps sessions trimmed automatically \u2014 Claude summarizes earlier so each turn stays cheaper without manual /compact.`:
			continue
		default:
			replacements = append(replacements, replacement)
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_212(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude212UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_212(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_212(data)
	usagePatched := patchUsageFetchFunction_2_1_212(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_212(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_212(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_212(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_212(data)
	fastModePricingPatched := patchFastModePricing_2_1_212(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_212(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_212(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_212(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_212(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude212UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_212(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Omt(){let e=Z.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=yei(),r=Z.DEMO_VERSION?"/code/claude":qd(wt()),n=Z.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=Kn().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function Omt(){", "function Xsa(", replacement)
}

func patchWhatsNewFeedFunction_2_1_212(data []byte) bool {
	const old = `function yaa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function yaa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_212(data []byte) bool {
	const replacement = `async function rwe(){return qc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "async function rwe(){", "var OEg=", replacement)
}

func patchModelPickerOptions_2_1_212(data []byte) bool {
	const replacement = `function CDX212(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(Z.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(Z.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(Z.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function fkh(e=!1){let t=Z,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function fkh(e=!1){", "function mCe(", replacement)
}

func patchModelPickerExtraOptions_2_1_212(data []byte) bool {
	const replacement = "function gkh(e){let t=fkh(e),r=Z.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX212(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:Z.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function gkh(e){", "function Xki(", replacement)
}

func patchModelPickerSelectionValue_2_1_212(data []byte) bool {
	return replaceFirstFixed(data, `a2y=lBe===null?ize:Xki(cBe,lBe)??lBe`, `a2y=lBe===null?ize:CDX212(lBe)`)
}

func patchFastModeRuntimeFunctions_2_1_212(data []byte) bool {
	ulPatched := replaceFirstFixed(data, `function ul(){if(xn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ul(){return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	o6Patched := replaceFirstFixed(data, `function o6(){return"Opus 4.8"}`, `function o6(){return"Codex AI"}`)
	vvtPatched := replaceFirstFixed(data, `function vvt(){return"opus"+(C1()?"[1m]":"")}`, `function vvt(){return"opus"}`)
	gsPatched := replaceFirstFixed(data, `function gS(e){if(!ul())return!1;let t=e??OF(),r=oi(t);if(D9(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function gS(e){return ul()}`)
	return ulPatched && o6Patched && vvtPatched && gsPatched
}

func patchFastModePricing_2_1_212(data []byte) bool {
	return replaceFirstFixed(data, "function o9e(e){return`${$oc(e.inputTokens)}/${$oc(e.outputTokens)} per Mtok`}", `function o9e(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_212(data []byte) bool {
	return replaceClaude208Function(data, "function IVi(e,t,r){", "function DQn(", `function IVi(e,t,r){return null}`)
}

func patchResumeCommandHints_2_1_212(data []byte) bool {
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

func patchCompactProgressCurve_2_1_212(data []byte) bool {
	return replaceFirstFixed(data, `function ZWd(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function ZWd(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_212(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function w0(){if(Mwo())return!0;if(dPt())return!1;return!B6()&&VRt()}`, `function w0(){return!!Z.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function Zks(){if(Mwo())return!0;if(dPt())return!1;return uPt()&&!B6()&&ccr()&&await Oj("tengu_ccr_bridge")}`, `async function Zks(){return!dPt()&&!B6()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function Nwo(){", "function Gq_(", "async function Nwo(){if(dPt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(B6())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
