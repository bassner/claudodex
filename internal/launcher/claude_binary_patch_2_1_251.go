package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_251 = claudeUIPatchSpec{
	Version: "2.1.251",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "625869b01e0050f260b2980fac248fd9cef9e462612bded4ec9d3d49ff8969a5",
	Apply:   applyClaudeUIPatches_2_1_251,
}

const (
	claude251ActiveHeaderBrandTarget   = "\x0b\x00\x00\x80\xc9/\x86\x00Claude Code\x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle"
	claude251ActiveFastModeBrandTarget = "\x06\x00\x00\x80\x6c\x06\x46\x00Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
)

var claude251UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	counts := map[string]int{
		`Claude is waiting for your input`: 3,
		`Claude needs your permission`:     4,
	}
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude247UIBrandingReplacements))
	for _, replacement := range claude247UIBrandingReplacements {
		if count, ok := counts[replacement.old]; ok {
			replacement.expectedCount = count
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_251(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude251UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_251(data, claudodexVersion, claudeVersion)
	headerBrandingPatched := patchActiveHeaderBrand_2_1_251(data)
	defaultTierLabelPatched := patchDefaultTierLabel_2_1_251(data)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_251(data)
	usagePatched := patchUsageFetchFunction_2_1_251(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_251(data, modelCfg)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_251(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_251(data)
	agentModelValidatorPatched := patchAgentModelValidator_2_1_251(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_251(data)
	activeFastModeBrandPatched := patchActiveFastModeBrand_2_1_251(data)
	fastModePricingPatched := patchFastModePricing_2_1_251(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_251(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_251(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_251(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_251(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude251UIBrandingReplacements)

	changed := versionPatched || headerBrandingPatched || defaultTierLabelPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || agentModelValidatorPatched || fastModePatched || activeFastModeBrandPatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed
	if !versionPatched || !headerBrandingPatched || !defaultTierLabelPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !agentModelValidatorPatched || !fastModePatched || !activeFastModeBrandPatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func patchDefaultTierLabel_2_1_251(data []byte) bool {
	const target = "Default (recommended)"
	if bytes.Count(data, []byte(target)) != 4 {
		return false
	}
	return replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_251(data []byte) bool {
	const target = "Claude Code\x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle"
	if bytes.Count(data, []byte(claude251ActiveHeaderBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle")
}

func patchLogoDisplayDataFunction_2_1_251(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function ARe(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,r=Nkn(),t=a.DEMO_VERSION?"/code/claude":Fo(ee()),e=a.CLAUDE_CODE_HIDE_CWD?"":r?` + "`${t} in ${r.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",s=Je().agent;return{version:o,cwd:e,billingType:i,agentName:s}}`
	return replaceClaude208Function(data, "function ARe(){let o=a.DEMO_VERSION??", "function a7t(o,r,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_251(data []byte) bool {
	const replacement = `var W=async(i,n)=>{return y("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var W=async(i,n)=>{try{", "function x(L){", replacement)
}

func patchUsageFetchFunction_2_1_251(data []byte) bool {
	const replacement = `async function uI(e,{atWall:r=!1}={}){return Hr(r?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let o=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=r?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",s=await fetch(o+t,{headers:{"Content-Type":"application/json"}});if(!s.ok)throw Error("Auth error: "+s.status);return await s.json()})}`
	return replaceClaude208Function(data, "async function uI(e,{atWall:r=!1}={}){", `var bDe=`, replacement)
}

func patchModelPickerOptions_2_1_251(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX251(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function tn(e=!1){let t=a,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function tn(e=!1){", "function S(e){", replacement)
}

func patchModelPickerExtraOptions_2_1_251(data []byte) bool {
	optionsPatched := replaceClaude208Function(data, "function iwe(e=!1,o=null){", "function ln(e,o){", "function iwe(e=!1,o=null){return tn(e)}")
	extraOptionsPatched := replaceClaude208Function(data, "function ln(e,o){let t=tn(e),", "function npn(e){", "function ln(e,o){return tn(e)}")
	const sessionOptionsTarget = `Ho=z(()=>{let i=[];for(let[a,v,R]of[[re.current,re.value,"Current model"],[re.sessionOverride===null?null:n,n===null?T:fmt(q,n)??n,"Base model"]])if(a!==null&&!q.some((L)=>L.value===v)&&!i.some((L)=>L.value===a)&&kr(a))i.push({value:a,label:bC(a),description:R});if(i.length===0)return q;let c=q.findIndex((a)=>a.disabled===!0);if(c===-1)return[...q,...i];return[...q.slice(0,c),...i,...q.slice(c)]},[q,re,n])`
	sessionOptionsPatched := replaceFirstFixed(data, sessionOptionsTarget, "Ho=q.slice(0,3)")
	return optionsPatched && extraOptionsPatched && sessionOptionsPatched
}

func patchModelPickerSelectionValue_2_1_251(data []byte) bool {
	return replaceClaude208Function(data, "function fmt(e,o){if(e.some((r)=>r.value===o))return o;", "function Ce(){", `function fmt(e,o){let n=CDX251(o),t=e.find((r)=>r.value===n||CDX251(r.value)===n);return t?.value??n}`)
}

func patchAgentModelValidator_2_1_251(data []byte) bool {
	return replaceFirstFixed(data, `model:ie(["sonnet","opus","haiku","fable"]).optional()`, `model:i().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_251(data []byte) bool {
	gatePatched := replaceFirstFixed(data, `function Yr(){if(Ne()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Yr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`)
	namePatched := replaceFirstFixed(data, `function _C(){return"Opus 5"}`, `function _C(){return"Codex"}`)
	modelPatched := replaceFirstFixed(data, `function CMe(){return"opus"+(YS()?"[1m]":"")}`, `function CMe(){return"opus"}`)
	capabilityPatched := replaceFirstFixed(data, `function ker(e,t){if(!Yr())return!1;return!!e&&(sn()||Zy()||t)}`, `function ker(e,t){return Yr()&&!!e}`)
	enabledPatched := replaceFirstFixed(data, `function NSt(e){if(!Yr())return!1;if(!Zy(e))return!1;if(!af(e))return!1;return OSn(Je())}`, `function NSt(e){return Yr()&&(ye("flagSettings")?.fastMode===!0||OSn(Je()))}`)
	optInPatched := replaceFirstFixed(data, `function OSn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function OSn(e){return e.fastMode===!0}`)
	supportPatched := replaceFirstFixed(data, `function af(e){if(!Yr())return!1;let t=e??eS(),r=Ot(t);if(hh(Ye(r),"fast_mode"))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function af(e){return Yr()}`)
	statePatched := replaceFirstFixed(data, `function dw(e,t){if(sn()){if(e===null)return!!t;return!!t&&af(e)}if(!af(e))return!1;return!!t||NSt(e)}`, `function dw(e,t){return Yr()&&(t!==void 0?!!t:OSn(Je()))}`)
	initialStateGatePatched := replaceFirstFixed(data, `...Yr()&&{fastMode:NSt(et??null)}`, `fastMode:NSt(et??null)`)
	mainRequestGatePatched := replaceFirstFixed(data, `...Fe.gates.fastModeEnabled&&{fastMode:ct.options.fastMode}`, `fastMode:ct.options.fastMode`)
	retryStateGatePatched := replaceFirstFixed(data, `u={model:r.model,...Yr()&&{fastMode:r.fastMode}}`, `u={model:r.model,fastMode:r.fastMode}`)
	syncFallbackGatePatched := replaceFirstFixed(data, `...Yr()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`)
	if bytes.Count(data, []byte(`...Yr()&&{fastMode:xr}`)) != 2 {
		return false
	}
	streamFallbackGatesPatched := replaceAllFixed(data, `...Yr()&&{fastMode:xr}`, `fastMode:xr`)
	streamPrimaryGatePatched := replaceFirstFixed(data, `...Yr()?{fastMode:xr}:!1`, `fastMode:xr`)
	initialRequestPatched := replaceFirstFixed(data, `let Yn=[...At,...mr],xr=Yr()&&Zy()&&!v3()&&af(fe)&&!!d.fastMode;`, `let Yn=[...At,...mr],xr=!!d.fastMode;`)
	retryRequestPatched := replaceFirstFixed(data, `if(Yr()&&Zy()&&!v3()&&af(fe)&&!!sr.fastMode)lv="fast";`, `if(sr.fastMode)lv="fast";`)
	return gatePatched && namePatched && modelPatched && capabilityPatched && enabledPatched && optInPatched && supportPatched && statePatched && initialStateGatePatched && mainRequestGatePatched && retryStateGatePatched && syncFallbackGatePatched && streamFallbackGatesPatched && streamPrimaryGatePatched && initialRequestPatched && retryRequestPatched
}

func patchActiveFastModeBrand_2_1_251(data []byte) bool {
	const target = "Opus 5\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	if bytes.Count(data, []byte(claude251ActiveFastModeBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex+\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok")
}

func patchFastModePricing_2_1_251(data []byte) bool {
	return replaceFirstFixed(data, "function _re(e){return`${Ck(e.inputTokens)}/${Ck(e.outputTokens)} per Mtok`}", `function _re(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_251(data []byte) bool {
	return replaceClaude208Function(data, "function Dy(S,P,T){", "var Oy=", `function Dy(S,P,T){return null}`)
}

func patchResumeCommandHints_2_1_251(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ", 2},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
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

func patchCompactProgressCurve_2_1_251(data []byte) bool {
	return replaceFirstFixed(data, `function Cn(n){let i=Math.max(0,n)/1000,l=1-Math.exp(-i/90);return Math.min(95,Math.round(l*100))}`, `function Cn(n){let i=Math.max(0,n)/2000,l=1-Math.exp(-i/90);return Math.min(95,Math.round(l*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_251(data []byte) bool {
	tokenPatched := replaceClaude208Function(data, "function gH(){return}function s3(){return}", "function ayr(e){", `function gH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function s3(){return}function __(){return gH()||Yt()?.accessToken}async function cC(e){return __()}function Kue(){return s3()??zt().BASE_API_URL}function Dne(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return ayr(e)||"remote-control"}`)
	visiblePatched := replaceFirstFixed(data, `function ch(){if(u())return!0;if(TEe())return!1;return!eA()&&aMe()}`, `function ch(){return!!gH()}`)
	availablePatched := replaceFirstFixed(data, `function Tyn(){if(u())return!0;return!TEe()&&!eA()&&g6()}`, `function Tyn(){return!!gH()}`)
	enabledPatched := replaceFirstFixed(data, `async function Eyn(){if(u())return!0;if(TEe())return!1;return g6()&&!eA()&&i()&&await Lp("tengu_ccr_bridge")}`, `async function Eyn(){return!TEe()&&!eA()&&!!gH()}`)
	errorPatched := replaceClaude208Function(data, "async function _9t(){", "function Xvr(){", "async function _9t(){if(TEe())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(eA())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	commandEnabledPatched := replaceFirstFixed(data, `function e(){if(ch())return!0;try{return g6()&&!eA()&&!TEe()&&Nl().source==="none"&&py({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!dpn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
	commandVisiblePatched := replaceFirstFixed(data, `get isHidden(){return!ch()}`, `get isHidden(){return!1}`)
	return tokenPatched && visiblePatched && availablePatched && enabledPatched && errorPatched && commandEnabledPatched && commandVisiblePatched
}
