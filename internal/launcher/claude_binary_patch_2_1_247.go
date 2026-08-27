package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_247 = claudeUIPatchSpec{
	Version: "2.1.247",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "5086b9b64d8bb842e1f599cdd3767ab08c6b2266e462fcc5686ae4b019cca8f7",
	Apply:   applyClaudeUIPatches_2_1_247,
}

const (
	claude247ActiveHeaderBrandTarget   = "\x0b\x00\x00\x80\xc9/\x86\x00Claude Code\x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar"
	claude247ActiveFastModeBrandTarget = "\x06\x00\x00\x80\x6c\x06\x46\x00Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
)

var claude247UIBrandingReplacements = func() []claude209UIBrandingReplacement {
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

func applyClaudeUIPatches_2_1_247(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude247UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_247(data, claudodexVersion, claudeVersion)
	headerBrandingPatched := patchActiveHeaderBrand_2_1_247(data)
	defaultTierLabelPatched := patchDefaultTierLabel_2_1_247(data)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_247(data)
	usagePatched := patchUsageFetchFunction_2_1_247(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_247(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_247(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_247(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_247(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_247(data)
	activeFastModeBrandPatched := patchActiveFastModeBrand_2_1_247(data)
	fastModePricingPatched := patchFastModePricing_2_1_247(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_247(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_247(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_247(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_247(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude247UIBrandingReplacements)

	changed := versionPatched || headerBrandingPatched || defaultTierLabelPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || activeFastModeBrandPatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !headerBrandingPatched || !defaultTierLabelPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !activeFastModeBrandPatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchDefaultTierLabel_2_1_247(data []byte) bool {
	const target = "Default (recommended)"
	if bytes.Count(data, []byte(target)) != 4 {
		return false
	}
	return replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_247(data []byte) bool {
	const target = "Claude Code\x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar"
	if bytes.Count(data, []byte(claude247ActiveHeaderBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x16\x00\x00\x80d\xf1\x0e\x00tengu_terminal_sidebar")
}

func patchLogoDisplayDataFunction_2_1_247(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Z(){let o=f.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,r=E(),t=f.DEMO_VERSION?"/code/claude":I(P()),s=f.CLAUDE_CODE_HIDE_CWD?"":r?` + "`${t} in ${r.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",n=b().agent;return{version:o,cwd:s,billingType:i,agentName:n}}`
	return replaceClaude208Function(data, "function Z(){let o=f.DEMO_VERSION??", "function L(o,r,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_247(data []byte) bool {
	const replacement = `var ue=async(o,e)=>{return g("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",e.applyMessageOp,o),null};`
	return replaceClaude208Function(data, "var ue=async(o,e)=>{try{", "function z(X){", replacement)
}

func patchUsageFetchFunction_2_1_247(data []byte) bool {
	const replacement = `async function J(t){return u("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),n=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!n.ok)throw Error("Auth error: "+n.status);return await n.json()})}`
	return replaceClaude208Function(data, "async function J(t){", `var P=`, replacement)
}

func patchModelPickerOptions_2_1_247(data []byte) bool {
	const replacement = `function CDX247(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(g.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(g.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(g.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function mt(e=!1){let t=g,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function mt(e=!1){", "function h(e){", replacement)
}

func patchModelPickerExtraOptions_2_1_247(data []byte) bool {
	const replacement = "function vt(e,n){return mt(e)}"
	return replaceClaude208Function(data, "function vt(e,n){let o=mt(e),", "function _t(e){", replacement)
}

func patchModelPickerSelectionValue_2_1_247(data []byte) bool {
	return replaceClaude208Function(data, "function ks(e,n){if(e.some((i)=>i.value===n))return n;", "function mo(){", `function ks(e,n){return CDX247(n)}`)
}

func patchAgentModelValidator_2_1_247(data []byte) bool {
	return replaceFirstFixed(data, `model:un(["sonnet","opus","haiku","fable"]).optional()`, `model:I().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_247(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Kt(){if(k()!=="firstParty")return!1;return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Kt(){return!c.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function dI(){return"Opus 5"}`, `function dI(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function fI(){return"opus"+(zr()?"[1m]":"")}`, `function fI(){return"opus"}`)
	optInPatched := replaceFirstFixed(data, `function gI(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(H("policySettings")?.fastModePerSessionOptIn===!0)return!1;return H("flagSettings")?.fastMode===!0}`, `function gI(e){return e.fastMode===!0}`)
	capabilityPatched := replaceFirstFixed(data, `function r2(e,t){if(!Kt())return!1;return!!e&&(yf()||Ef()||t)}`, `function r2(e,t){return Kt()&&!!e}`)
	enabledPatched := replaceFirstFixed(data, `function pI(e){if(!Kt())return!1;if(!Ef(e))return!1;if(!Gr(e))return!1;return gI(Se())}`, `function pI(e){return Kt()&&(H("flagSettings")?.fastMode===!0||gI(Se()))}`)
	statePatched := replaceFirstFixed(data, `function o2(e,t){if(yf()){if(e===null)return!!t;return!!t&&Gr(e)}if(!Gr(e))return!1;return!!t||pI(e)}`, `function o2(e,t){return Kt()&&(t!==void 0?!!t:gI(Se()))}`)
	headlessSettingsSyncPatched := replaceFirstFixed(data, `if(GP(W,r,l.storageV5),Tw())r((Ct)=>{let sn=Rw(Ct.settings);return Ct.fastMode===sn?Ct:{...Ct,fastMode:sn}});`, `if(GP(W,r,l.storageV5),Tw())r((Ct)=>({...Ct,fastMode:Ct.settings.fastMode===!0}));`)
	supportPatched := replaceFirstFixed(data, `function Gr(e){if(!Kt())return!1;let t=e??Lu(),n=J(t);if(fn(O(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-8")||r.includes("opus-5")}`, `function Gr(e){return Kt()}`)
	initialStateGatePatched := replaceFirstFixed(data, `...dl()&&{fastMode:Pr(Ne??null)}`, `fastMode:Pr(Ne??null)`)
	mainRequestGatePatched := replaceFirstFixed(data, `...T.gates.fastModeEnabled&&{fastMode:W.options.fastMode}`, `fastMode:W.options.fastMode`)
	streamRetryGatePatched := replaceFirstFixed(data, `o={model:n.model,...Am()&&{fastMode:n.fastMode}}`, `o={model:n.model,fastMode:n.fastMode}`)
	syncFallbackGatePatched := replaceFirstFixed(data, `...Am()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`)
	if bytes.Count(data, []byte(`...Am()&&{fastMode:nt}`)) != 2 {
		return false
	}
	streamFallbackGatesPatched := replaceAllFixed(data, `...Am()&&{fastMode:nt}`, `fastMode:nt`)
	initialRequestPatched := replaceFirstFixed(data, `let Xe=[...L,...Ae],nt=Am()&&nGt()&&!dhe()&&rGt(y)&&!!s.fastMode;`, `let Xe=[...L,...Ae],nt=!!s.fastMode;`)
	retryRequestPatched := replaceFirstFixed(data, `if(Am()&&nGt()&&!dhe()&&rGt(y)&&!!et.fastMode)Ov="fast";`, `if(et.fastMode)Ov="fast";`)
	return gatePatched && namePatched && modelPatched && optInPatched && capabilityPatched && enabledPatched && statePatched && headlessSettingsSyncPatched && supportPatched && initialStateGatePatched && mainRequestGatePatched && streamRetryGatePatched && syncFallbackGatePatched && streamFallbackGatesPatched && initialRequestPatched && retryRequestPatched
}

func patchActiveFastModeBrand_2_1_247(data []byte) bool {
	const target = "Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	if bytes.Count(data, []byte(claude247ActiveFastModeBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok")
}

func patchFastModePricing_2_1_247(data []byte) bool {
	return replaceFirstFixed(data, "function Hu(e){return`${Eb(e.inputTokens)}/${Eb(e.outputTokens)} per Mtok`}", `function Hu(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_247(data []byte) bool {
	return replaceClaude208Function(data, "function fN(_9t){", "JF();Se();ta();r3();He();wt();Vr();hN();Cot();ne();", `function fN(_9t){return null}`)
}

func patchResumeCommandHints_2_1_247(data []byte) bool {
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

func patchCompactProgressCurve_2_1_247(data []byte) bool {
	return replaceFirstFixed(data, `function ao(t){let n=Math.max(0,t)/1000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`, `function ao(t){let n=Math.max(0,t)/2000,r=1-Math.exp(-n/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_247(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function f(){return}function E(){return}", "function _(e){", `function f(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function E(){return}function l(){return f()||o()?.accessToken}async function R(e){return l()}function U(){return E()??i().BASE_API_URL}function x(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||m();return _(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function SB(){if(fc())return!0;if(ra())return!1;return!go()&&wg()}`, `function SB(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function vB(){if(fc())return!0;if(ra())return!1;return na()&&!go()&&hi()&&await ui("tengu_ccr_bridge")}`, `async function vB(){return!ra()&&!go()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function CT(){", "function CB(){", "async function CT(){if(ra())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(go())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return tokenPatched && visiblePatched && enabledPatched && errorPatched
}
