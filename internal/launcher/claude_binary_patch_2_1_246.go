package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_246 = claudeUIPatchSpec{
	Version: "2.1.246",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "7b09f01cb76a38e0e3a7c47c5d698d382162a5ff26538fc778683770caf9218b",
	Apply:   applyClaudeUIPatches_2_1_246,
}

const (
	claude246ActiveHeaderBrandTarget   = "\x0b\x00\x00\x80\xc9/\x86\x00Claude Code\x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar"
	claude246ActiveFastModeBrandTarget = "\x06\x00\x00\x80\x6c\x06\x46\x00Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
)

var claude246UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	counts := map[string]int{
		`Welcome to Claude Code`:                                                       7,
		`No, and tell Claude what to do differently `:                                  3,
		`and tell Claude what to do differently`:                                       9,
		`and tell Claude what to do next`:                                              4,
		`To add hooks, edit settings.json directly or ask Claude`:                      3,
		`Claude ended this conversation. Start a new session (or /clear) to continue.`: 7,
		`Claude Code needs your input`:                                                 3,
		`[Image data detected and sent to Claude]`:                                     3,
		`Push when Claude decides`:                                                     4,
		`Generate a report analyzing your Claude Code sessions`:                        3,
		`Claude Max`: 3,
		`Hit Enter to queue up additional messages while Claude is working.`: 2,
		`Claude needs your permission`:                                       5,
		`Claude wants to use your browser`:                                   3,
	}
	removed := map[string]bool{
		`Check the Claude Code changelog for updates`: true,
	}
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude245UIBrandingReplacements))
	for _, replacement := range claude245UIBrandingReplacements {
		if removed[replacement.old] {
			continue
		}
		if count, ok := counts[replacement.old]; ok {
			replacement.expectedCount = count
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_246(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude246UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_246(data, claudodexVersion, claudeVersion)
	headerBrandingPatched := patchActiveHeaderBrand_2_1_246(data)
	defaultTierLabelPatched := patchDefaultTierLabel_2_1_246(data)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_246(data)
	usagePatched := patchUsageFetchFunction_2_1_246(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_246(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_246(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_246(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_246(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_246(data)
	activeFastModeBrandPatched := patchActiveFastModeBrand_2_1_246(data)
	fastModePricingPatched := patchFastModePricing_2_1_246(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_246(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_246(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_246(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_246(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude246UIBrandingReplacements)

	changed := versionPatched || headerBrandingPatched || defaultTierLabelPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || activeFastModeBrandPatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !headerBrandingPatched || !defaultTierLabelPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !activeFastModeBrandPatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchDefaultTierLabel_2_1_246(data []byte) bool {
	const target = "Default (recommended)"
	if bytes.Count(data, []byte(target)) != 4 {
		return false
	}
	return replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_246(data []byte) bool {
	const target = "Claude Code\x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar"
	if bytes.Count(data, []byte(claude246ActiveHeaderBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar")
}

func patchLogoDisplayDataFunction_2_1_246(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Z(){let o=f.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,r=E(),t=f.DEMO_VERSION?"/code/claude":I(P()),s=f.CLAUDE_CODE_HIDE_CWD?"":r?` + "`${t} in ${r.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",n=b().agent;return{version:o,cwd:s,billingType:i,agentName:n}}`
	return replaceClaude208Function(data, "function Z(){let o=f.DEMO_VERSION??", "function L(o,r,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_246(data []byte) bool {
	const replacement = `var ue=async(o,e)=>{return g("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",e.applyMessageOp,o),null};`
	return replaceClaude208Function(data, "var ue=async(o,e)=>{try{", "function z(X){", replacement)
}

func patchUsageFetchFunction_2_1_246(data []byte) bool {
	const replacement = `async function J(t){return u("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),n=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!n.ok)throw Error("Auth error: "+n.status);return await n.json()})}`
	return replaceClaude208Function(data, "async function J(t){", `var P=`, replacement)
}

func patchModelPickerOptions_2_1_246(data []byte) bool {
	const replacement = `function CDX246(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(g.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(g.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(g.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function mt(e=!1){let t=g,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function mt(e=!1){", "function h(e){", replacement)
}

func patchModelPickerExtraOptions_2_1_246(data []byte) bool {
	const replacement = "function vt(e,n){let o=mt(e),t=g.ANTHROPIC_CUSTOM_MODEL_OPTION,i=CDX246(t);if(t&&i===t&&!o.some((l)=>l.value===t))o.push({value:t,label:g.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??t,description:g.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${t})`});return o}"
	return replaceClaude208Function(data, "function vt(e,n){let o=mt(e),", "function _t(e){", replacement)
}

func patchModelPickerSelectionValue_2_1_246(data []byte) bool {
	return replaceClaude208Function(data, "function ks(e,n){if(e.some((i)=>i.value===n))return n;", "function mo(){", `function ks(e,n){return CDX246(n)}`)
}

func patchAgentModelValidator_2_1_246(data []byte) bool {
	return replaceFirstFixed(data, `model:ln(["sonnet","opus","haiku","fable"]).optional()`, `model:I().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_246(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Bt(){if(C()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Bt(){return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function LD(){return"Opus 5"}`, `function LD(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function UD(){return"opus"+(Mr()?"[1m]":"")}`, `function UD(){return"opus"}`)
	optInPatched := replaceFirstFixed(data, `function HD(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(G("policySettings")?.fastModePerSessionOptIn===!0)return!1;return G("flagSettings")?.fastMode===!0}`, `function HD(e){return e.fastMode===!0}`)
	capabilityPatched := replaceFirstFixed(data, `function bY(e,t){if(!Bt())return!1;return!!e&&(uf()||cf()||t)}`, `function bY(e,t){return Bt()&&!!e}`)
	enabledPatched := replaceFirstFixed(data, `function BD(e){if(!Bt())return!1;if(!cf(e))return!1;if(!Pr(e))return!1;return HD(ve())}`, `function BD(e){return Bt()&&(G("flagSettings")?.fastMode===!0||HD(ve()))}`)
	statePatched := replaceFirstFixed(data, `function EY(e,t){if(uf()){if(e===null)return!!t;return!!t&&Pr(e)}if(!Pr(e))return!1;return!!t||BD(e)}`, `function EY(e,t){return Bt()&&(t!==void 0?!!t:HD(ve()))}`)
	headlessSettingsSyncPatched := replaceFirstFixed(data, `if(sE(B,r,a.storageV5),xC())r((gt)=>{let At=FC(gt.settings);return gt.fastMode===At?gt:{...gt,fastMode:At}});`, `if(sE(B,r,a.storageV5),xC())r((gt)=>({...gt,fastMode:gt.settings.fastMode===!0}));`)
	supportPatched := replaceFirstFixed(data, `function Pr(e){if(!Bt())return!1;let t=e??Iu(),n=X(t);if(un(P(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`, `function Pr(e){return Bt()}`)
	initialStateGatePatched := replaceFirstFixed(data, `...dl()&&{fastMode:Pr(Ne??null)}`, `fastMode:Pr(Ne??null)`)
	mainRequestGatePatched := replaceFirstFixed(data, `...k.gates.fastModeEnabled&&{fastMode:z.options.fastMode}`, `fastMode:z.options.fastMode`)
	streamRetryGatePatched := replaceFirstFixed(data, `o={model:n.model,...hm()&&{fastMode:n.fastMode}}`, `o={model:n.model,fastMode:n.fastMode}`)
	syncFallbackGatePatched := replaceFirstFixed(data, `...hm()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`)
	if bytes.Count(data, []byte(`...hm()&&{fastMode:et}`)) != 2 {
		return false
	}
	streamFallbackGatesPatched := replaceAllFixed(data, `...hm()&&{fastMode:et}`, `fastMode:et`)
	initialRequestPatched := replaceFirstFixed(data, `let qe=[...O,...Xe],et=hm()&&xWt()&&!wge()&&IWt(y)&&!!s.fastMode;`, `let qe=[...O,...Xe],et=!!s.fastMode;`)
	retryRequestPatched := replaceFirstFixed(data, `if(hm()&&xWt()&&!wge()&&IWt(y)&&!!He.fastMode)wv="fast";`, `if(He.fastMode)wv="fast";`)
	return gatePatched && namePatched && modelPatched && optInPatched && capabilityPatched && enabledPatched && statePatched && headlessSettingsSyncPatched && supportPatched && initialStateGatePatched && mainRequestGatePatched && streamRetryGatePatched && syncFallbackGatePatched && streamFallbackGatesPatched && initialRequestPatched && retryRequestPatched
}

func patchActiveFastModeBrand_2_1_246(data []byte) bool {
	const target = "Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	if bytes.Count(data, []byte(claude246ActiveFastModeBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok")
}

func patchFastModePricing_2_1_246(data []byte) bool {
	return replaceFirstFixed(data, "function Lu(e){return`${sb(e.inputTokens)}/${sb(e.outputTokens)} per Mtok`}", `function Lu(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_246(data []byte) bool {
	return replaceClaude208Function(data, "function gL($Xt){", "nN();_e();Ts();", `function gL($Xt){return null}`)
}

func patchResumeCommandHints_2_1_246(data []byte) bool {
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

func patchCompactProgressCurve_2_1_246(data []byte) bool {
	return replaceFirstFixed(data, `function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`, `function ao(t){let n=Math.max(0,t)/2000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_246(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function f(){return}function E(){return}", "function _(e){", `function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function E(){return}function l(){return f()||o()?.accessToken}async function R(e){return l()}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function W0(){if(sc())return!0;if(qs())return!1;return!ro()&&pg()}`, `function W0(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function j0(){if(sc())return!0;if(qs())return!1;return Ys()&&!ro()&&fi()&&await oi("tengu_ccr_bridge")}`, `async function j0(){return!qs()&&!ro()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function eT(){", "function Y0(){", "async function eT(){if(qs())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(ro())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
