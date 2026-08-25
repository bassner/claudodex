package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_245 = claudeUIPatchSpec{
	Version: "2.1.245",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "9f7c2260251765a18d0b35198669dacc1912f6e8129a3b01f6b58d93365ff1f1",
	Apply:   applyClaudeUIPatches_2_1_245,
}

var claude245UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	removed := map[string]bool{
		`You can grant Claude access to additional directories without changing your current working directory.`:                                                                                                                 true,
		`You can hit Enter while Claude is working to queue a follow-up or steer it mid-turn \u2014 no need to wait for it to finish.`:                                                                                           true,
		`Setting a 200K auto-compact window keeps sessions trimmed automatically \u2014 Claude summarizes earlier so each turn stays cheaper without manual /compact.`:                                                           true,
		`Approving lets Claude write to ANY file in this project without another prompt for up to 4 hours (new and changed file contents are not shown for approval). Deletes and CLAUDE.md/.claude paths still ask every time.`: true,
		`Approving lets Claude write to ANY file in the project "`: true,
	}
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude234UIBrandingReplacements))
	for _, replacement := range claude234UIBrandingReplacements {
		if removed[replacement.old] {
			continue
		}
		if replacement.old == `Hit Enter to queue up additional messages while Claude is working.` {
			replacement.expectedCount = 3
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_245(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude245UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_245(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_245(data)
	usagePatched := patchUsageFetchFunction_2_1_245(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_245(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_245(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_245(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_245(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_245(data)
	fastModePricingPatched := patchFastModePricing_2_1_245(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_245(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_245(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_245(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_245(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude245UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_245(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function gt(){let r=p.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,n=I(),t=p.DEMO_VERSION?"/code/claude":_(R()),i=p.CLAUDE_CODE_HIDE_CWD?"":n?` + "`${t} in ${n.replace(/^https?:\\/\\//,\"\")}`" + `:t,o="Codex Plan",s=y().agent;return{version:r,cwd:i,billingType:o,agentName:s}}`
	return replaceClaude208Function(data, "function gt(){let r=p.DEMO_VERSION??", "function mt(r,n,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_245(data []byte) bool {
	const old = `function Qa(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Qa(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_245(data []byte) bool {
	const replacement = `async function J(t){return c("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),n=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!n.ok)throw Error("Auth error: "+n.status);return await n.json()})}`
	return replaceClaude208Function(data, "async function J(t){", `var P=`, replacement)
}

func patchModelPickerOptions_2_1_245(data []byte) bool {
	const replacement = `function CDX245(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(g.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(g.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(g.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function mt(e=!1){let t=g,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function mt(e=!1){", "function h(e){", replacement)
}

func patchModelPickerExtraOptions_2_1_245(data []byte) bool {
	const replacement = "function vt(e,n){let o=mt(e),t=g.ANTHROPIC_CUSTOM_MODEL_OPTION,i=CDX245(t);if(t&&i===t&&!o.some((l)=>l.value===t))o.push({value:t,label:g.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??t,description:g.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${t})`});return o}"
	return replaceClaude208Function(data, "function vt(e,n){", "function _t(e){", replacement)
}

func patchModelPickerSelectionValue_2_1_245(data []byte) bool {
	return replaceClaude208Function(data, "function ks(e,n){if(e.some((i)=>i.value===n))return n;", "function mo(){", `function ks(e,n){return CDX245(n)}`)
}

func patchAgentModelValidator_2_1_245(data []byte) bool {
	return replaceFirstFixed(data, `model:Cn(["sonnet","opus","haiku","fable"]).optional()`, `model:D().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_245(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Dt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Dt(){return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function QO(){return"Opus 5"}`, `function QO(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function ZO(){return"opus"+(wr()?"[1m]":"")}`, `function ZO(){return"opus"}`)
	supportPatched := replaceFirstFixed(data, `function Ar(e){if(!Dt())return!1;let t=e??Qa(),n=q(t);if(en(R(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`, `function Ar(e){return Dt()}`)
	return gatePatched && namePatched && modelPatched && supportPatched
}

func patchFastModePricing_2_1_245(data []byte) bool {
	return replaceFirstFixed(data, "function nu(e){return`${$_(e.inputTokens)}/${$_(e.outputTokens)} per Mtok`}", `function nu(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_245(data []byte) bool {
	return replaceClaude208Function(data, "function bL(r1t){", "vD();", `function bL(r1t){return null}`)
}

func patchResumeCommandHints_2_1_245(data []byte) bool {
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

func patchCompactProgressCurve_2_1_245(data []byte) bool {
	return replaceFirstFixed(data, `function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`, `function ao(t){let n=Math.max(0,t)/2000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_245(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function f(){return}function E(){return}", "function _(e){", `function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function E(){return}function l(){return f()||o()?.accessToken}async function R(e){return l()}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function ZF(){if(Rl())return!0;if(As())return!1;return!Jr()&&lp()}`, `function ZF(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function eL(){if(Rl())return!0;if(As())return!1;return Cs()&&!Jr()&&qo()&&await Ko("tengu_ccr_bridge")}`, `async function eL(){return!As()&&!Jr()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function VC(){", "function tL(){", "async function VC(){if(As())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(Jr())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
