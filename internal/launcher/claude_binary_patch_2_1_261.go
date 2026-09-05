package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_261 = claudeUIPatchSpec{
	Version: "2.1.261",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "5efecaff231b798be3c66def9be54183623b328b80eaef17f93c43987024e82a",
	Apply:   applyClaudeUIPatches_2_1_261,
}

const (
	claude261ModelOptionsOverrideTarget = `wt=V(()=>Oe??pdn(Zt),[Oe,Zt])`
	claude261ModelSelectionSourceTarget = `At=V(()=>bne(I,w),[I,w,eo,jt]),to=At??w,zt=to===null?zw:cmt(wt,to)??to,`
	claude261ModelExtraOptionsTarget    = `Go=V(()=>{let Ki=[];for(let[As,Ls,va]of[[fo.current,fo.value,"Current model"],[fo.sessionOverride===null?null:w,w===null?zw:cmt(wt,w)??w,"Base model"]])if(As!==null&&!wt.some((Bi)=>Bi.value===Ls)&&!Ki.some((Bi)=>Bi.value===As)&&Rr(As))Ki.push({value:As,label:WC(As),description:va});if(Ki.length===0)return wt;let qi=wt.findIndex((As)=>As.disabled===!0);if(qi===-1)return[...wt,...Ki];return[...wt.slice(0,qi),...Ki,...wt.slice(qi)]},[wt,fo,w])`
	claude261ModelPickerValueTarget     = `defaultValue:zt,selectedValue:zt,defaultFocusValue:Qo,options:Fn,`
)

var claude261UIBrandingReplacements = claude260UIBrandingReplacements

func applyClaudeUIPatches_2_1_261(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude261UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude261EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude261SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude261ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude261EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude261EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude261EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX261(")) && !bytes.Contains(entrySource, []byte("function pdn(e=!1){")) {
			continue
		}
		if found != nil {
			return claude259BunModuleRecord{}, false
		}
		current := record
		found = &current
	}
	if found == nil {
		return claude259BunModuleRecord{}, false
	}
	return *found, true
}

func claude261EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude261ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude261EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude261EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func patchLogoDisplayDataFunction_2_1_261(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function PHe(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,b=lOn(),x=a.DEMO_VERSION?"/code/claude":Ao(Q()),O=a.CLAUDE_CODE_HIDE_CWD?"":b?` + "`${x} in ${b.replace(/^https?:\\/\\//,\"\")}`" + `:x,R="Codex Plan",v=Ge().agent;return{version:l,cwd:O,billingType:R,agentName:v}}`
	return replaceClaude208Function(data, "function PHe(){let l=a.DEMO_VERSION??", "function zQt(l,b,x){", replacement)
}

func patchWhatsNewFeedFunction_2_1_261(data []byte) bool {
	const replacement = `var ee=async(s,n)=>{return y("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,s),null};`
	return replaceClaude208Function(data, "var ee=async(s,n)=>{try{", "function x(L){", replacement)
}

func patchUsageFetchFunction_2_1_261(data []byte) bool {
	const replacement = `async function RO(e,{atWall:t=!1}={}){return Sr(t?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),o=t?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",d=await fetch(r+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()})}`
	return replaceClaude208Function(data, "async function RO(e,{atWall:t=!1}={}){", "var sVe=", replacement)
}

func patchModelPickerOptions_2_1_261(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX261(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function pdn(e=!1){return vue(e)}function vue(e=!1,t=null){return CDXOpts261(e,t)}function CDXOpts261(e=!1,t=null){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}`
	return replaceClaude208Function(data, "function pdn(e=!1){", "function rto(e,t){", replacement)
}

func patchModelPickerResolver_2_1_261(data []byte) bool {
	return replaceClaude208Function(data, "function ato(e,t){", "function fdn(e){", `function ato(e,t){return CDXOpts261(e,t).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_261(data []byte) bool {
	if bytes.Count(data, []byte(claude261ModelOptionsOverrideTarget)) != 1 ||
		bytes.Count(data, []byte(claude261ModelSelectionSourceTarget)) != 1 ||
		bytes.Count(data, []byte(claude261ModelExtraOptionsTarget)) != 1 ||
		bytes.Count(data, []byte(claude261ModelPickerValueTarget)) != 1 {
		return false
	}
	return replaceFirstFixed(data, claude261ModelOptionsOverrideTarget, `wt=V(()=>pdn(Zt),[Zt])`) &&
		replaceFirstFixed(data, claude261ModelSelectionSourceTarget, claude261ModelSelectionSourceTarget) &&
		replaceFirstFixed(data, claude261ModelExtraOptionsTarget, `Go=wt.slice(0,3)`) &&
		replaceFirstFixed(data, claude261ModelPickerValueTarget, `defaultValue:zt,defaultFocusValue:Qo,options:Fn.slice(0,3),`)
}

func patchModelPickerSelectionValue_2_1_261(data []byte) bool {
	const replacement = `function cmt(e,t){let r=CDX261(t),o=e.find((d)=>d.value===r||CDX261(d.value)===r);return o?.value??r}`
	return replaceClaude208Function(data, "function cmt(e,t){if(e.some((d)=>d.value===t))return t;", "function lrn(){", replacement)
}

func patchAgentModelValidator_2_1_261(data []byte) bool {
	return replaceFirstFixed(data, `model:Y(["sonnet","opus","haiku","fable"]).optional()`, `model:s().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_261(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Mr(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Mr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function RR(){return"Opus 5"}`, `function RR(){return"Codex"}`),
		replaceFirstFixed(data, `function J$e(){return"opus"+(vw()?"[1m]":"")}`, `function J$e(){return"opus"}`),
		replaceFirstFixed(data, `function NCn(e,t){if(!Mr())return!1;return!!e&&(Pt()||Jy()||t)}`, `function NCn(e,t){return Mr()&&!!e}`),
		replaceFirstFixed(data, `function MAt(e){if(!Mr())return!1;if(!Jy(e))return!1;if(!af(e))return!1;return FCn(Ge())}`, `function MAt(e){return Mr()&&(ye("flagSettings")?.fastMode===!0||FCn(Ge()))}`),
		replaceFirstFixed(data, `function FCn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function FCn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function af(e){if(!Mr())return!1;let t=e??dh(),r=bt(t);if(dm(Ue(r),"fast_mode",r))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function af(e){return Mr()}`),
		replaceFirstFixed(data, `function ub(e,t){if(Pt()){if(e===null)return!!t;return!!t&&af(e)}if(!af(e))return!1;return!!t||MAt(e)}`, `function ub(e,t){return Mr()&&(t!==void 0?!!t:FCn(Ge()))}`),
		replaceFirstFixed(data, `...Mr()&&{fastMode:MAt(je??null)}`, `fastMode:MAt(je??null)`),
		replaceFirstFixed(data, `...He.gates.fastModeEnabled&&{fastMode:Ct.options.fastMode}`, `fastMode:Ct.options.fastMode`),
		replaceFirstFixed(data, `d={model:r.model,...Mr()&&{fastMode:r.fastMode}}`, `d={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Mr()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`),
	}
	if bytes.Count(data, []byte(`...Mr()&&{fastMode:Pa}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...Mr()&&{fastMode:Pa}`, `fastMode:Pa`),
		replaceFirstFixed(data, `...Mr()?{fastMode:Pa}:!1`, `fastMode:Pa`),
		replaceFirstFixed(data, `if(Mr()&&_(()=>Jy())&&!_se()&&_(()=>af(_e))&&!!Ur.fastMode)jD="fast";`, `if(Ur.fastMode)jD="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_261(data []byte) bool {
	return replaceFirstFixed(data, "function wse(e){return`${jx(e.inputTokens)}/${jx(e.outputTokens)} per Mtok`}", `function wse(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_261(data []byte) bool {
	return replaceClaude208Function(data, "function XS(w,I,O){", "var JS=", `function XS(w,I,O){return null}`)
}

func patchCompactProgressCurve_2_1_261(data []byte) bool {
	return replaceFirstFixed(data, `function Ro(n){let s=Math.max(0,n)/1000,c=1-Math.exp(-s/90);return Math.min(95,Math.round(c*100))}`, `function Ro(n){let s=Math.max(0,n)/2000,c=1-Math.exp(-s/90);return Math.min(95,Math.round(c*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_261(data []byte) bool {
	for _, transformation := range claude261RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude261RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function RH(){return}function rG(){return}", "function t(e){", `function RH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function rG(){return}function f_(){return RH()||Yt()?.accessToken}async function wC(e){return f_()}function Xpe(){return rG()??Vt().BASE_API_URL}function Wre(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function ab(){if(u())return!0;if(zG())return!1;return!UC()&&v$e()}`, `function ab(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function vAn(){if(u())return!0;return!zG()&&!UC()&&c6()}`, `function vAn(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function RAn(){if(u())return!0;if(zG())return!1;return c6()&&!UC()&&l()&&await rd("tengu_ccr_bridge")}`, `async function RAn(){return!zG()&&!UC()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function w4t(){", "function T(){", `async function w4t(){if(zG())return"Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).";if(UC())return"Remote Control is not available inside a cloud session.";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.";return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(ab())return!0;try{return c6()&&!UC()&&!zG()&&Gl().source==="none"&&kp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!Lyn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!ab()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude261Transformations(version string) []claude258Transformation {
	return claude261TransformationsForConfig(version, "2.1.261", modelconfig.Default())
}

func claude261TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude261SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude261EmbeddedPatchedModuleBytecode})
}

func claude261SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_261(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_261},
		{"usage", patchUsageFetchFunction_2_1_261},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_261(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_261},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_261},
		{"model-selection", patchModelPickerSelectionValue_2_1_261},
		{"agent-model-validator", patchAgentModelValidator_2_1_261},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_261},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_258},
		{"fast-mode-pricing", patchFastModePricing_2_1_261},
		{"context-warning", patchContextWarningHint_2_1_261},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_261},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_261},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude261UIBrandingReplacements)
		}},
	}
}

func claude261ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX261("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function rto("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
