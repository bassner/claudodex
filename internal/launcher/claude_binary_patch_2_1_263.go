package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_263 = claudeUIPatchSpec{
	Version: "2.1.263",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "ef5d2909c8af49f31ab6d5487e90316777bc2fac170adfe8160716caa8aaf4f9",
	Apply:   applyClaudeUIPatches_2_1_263,
}

const (
	claude263ModelOptionsOverrideTarget = `Ct=V(()=>Oe??fdn(Zt),[Oe,Zt])`
	claude263ModelSelectionSourceTarget = `Tt=V(()=>Tne(I,w),[I,w,eo,jt]),to=Tt??w,zt=to===null?Qw:umt(Ct,to)??to,`
	claude263ModelExtraOptionsTarget    = `Go=V(()=>{let Ki=[];for(let[As,Ls,va]of[[fo.current,fo.value,"Current model"],[fo.sessionOverride===null?null:w,w===null?Qw:umt(Ct,w)??w,"Base model"]])if(As!==null&&!Ct.some((Bi)=>Bi.value===Ls)&&!Ki.some((Bi)=>Bi.value===As)&&Rr(As))Ki.push({value:As,label:WC(As),description:va});if(Ki.length===0)return Ct;let qi=Ct.findIndex((As)=>As.disabled===!0);if(qi===-1)return[...Ct,...Ki];return[...Ct.slice(0,qi),...Ki,...Ct.slice(qi)]},[Ct,fo,w])`
	claude263ModelPickerValueTarget     = `defaultValue:zt,selectedValue:zt,defaultFocusValue:Qo,options:Fn,`
)

var claude263UIBrandingReplacements = claude261UIBrandingReplacements

func applyClaudeUIPatches_2_1_263(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude263UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude263EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude263SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude263ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude263EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude263EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude263EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX263(")) && !bytes.Contains(entrySource, []byte("function fdn(e=!1){")) {
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

func claude263EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude263ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude263EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude263EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func patchLogoDisplayDataFunction_2_1_263(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function PHe(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,b=uOn(),x=a.DEMO_VERSION?"/code/claude":Ao(Q()),O=a.CLAUDE_CODE_HIDE_CWD?"":b?` + "`${x} in ${b.replace(/^https?:\\/\\//,\"\")}`" + `:x,R="Codex Plan",v=Ge().agent;return{version:l,cwd:O,billingType:R,agentName:v}}`
	return replaceClaude208Function(data, "function PHe(){let l=a.DEMO_VERSION??", "function VQt(l,b,x){", replacement)
}

func patchUsageFetchFunction_2_1_263(data []byte) bool {
	const replacement = `async function kO(e,{atWall:t=!1}={}){return Sr(t?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),o=t?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",d=await fetch(r+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()})}`
	return replaceClaude208Function(data, "async function kO(e,{atWall:t=!1}={}){", "var aVe=", replacement)
}

func patchModelPickerOptions_2_1_263(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX263(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function fdn(e=!1){return CDXOpts263(e)}function Rue(e=!1,t=null){return CDXOpts263(e,t)}function CDXOpts263(e=!1,t=null){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}`
	return replaceClaude208Function(data, "function fdn(e=!1){", "function lto(e,t){", replacement)
}

func patchModelPickerResolver_2_1_263(data []byte) bool {
	return replaceClaude208Function(data, "function fto(e,t){", "function mdn(e){", `function fto(e,t){return CDXOpts263(e,t).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_263(data []byte) bool {
	if bytes.Count(data, []byte(claude263ModelOptionsOverrideTarget)) != 1 ||
		bytes.Count(data, []byte(claude263ModelSelectionSourceTarget)) != 1 ||
		bytes.Count(data, []byte(claude263ModelExtraOptionsTarget)) != 1 ||
		bytes.Count(data, []byte(claude263ModelPickerValueTarget)) != 1 {
		return false
	}
	return replaceFirstFixed(data, claude263ModelOptionsOverrideTarget, `Ct=V(()=>fdn(Zt),[Zt])`) &&
		replaceFirstFixed(data, claude263ModelSelectionSourceTarget, claude263ModelSelectionSourceTarget) &&
		replaceFirstFixed(data, claude263ModelExtraOptionsTarget, `Go=Ct.slice(0,3)`) &&
		replaceFirstFixed(data, claude263ModelPickerValueTarget, `defaultValue:zt,defaultFocusValue:Qo,options:Fn.slice(0,3),`)
}

func patchModelPickerSelectionValue_2_1_263(data []byte) bool {
	const replacement = `function umt(e,t){let r=CDX263(t),o=e.find((d)=>d.value===r||CDX263(d.value)===r);return o?.value??r}`
	return replaceClaude208Function(data, "function umt(e,t){if(e.some((d)=>d.value===t))return t;", "function drn(){", replacement)
}

func patchAgentModelValidator_2_1_263(data []byte) bool {
	return replaceFirstFixed(data, `model:X(["sonnet","opus","haiku","fable"]).optional()`, `model:s().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_263(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Mr(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Mr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function RR(){return"Opus 5"}`, `function RR(){return"Codex"}`),
		replaceFirstFixed(data, `function Q$e(){return"opus"+(vw()?"[1m]":"")}`, `function Q$e(){return"opus"}`),
		replaceFirstFixed(data, `function FCn(e,t){if(!Mr())return!1;return!!e&&(Pt()||Jy()||t)}`, `function FCn(e,t){return Mr()&&!!e}`),
		replaceFirstFixed(data, `function NAt(e){if(!Mr())return!1;if(!Jy(e))return!1;if(!af(e))return!1;return $Cn(Ge())}`, `function NAt(e){return Mr()&&(ye("flagSettings")?.fastMode===!0||$Cn(Ge()))}`),
		replaceFirstFixed(data, `function $Cn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function $Cn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function af(e){if(!Mr())return!1;let t=e??dh(),r=wt(t);if(dm(Ue(r),"fast_mode",r))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function af(e){return Mr()}`),
		replaceFirstFixed(data, `function db(e,t){if(Pt()){if(e===null)return!!t;return!!t&&af(e)}if(!af(e))return!1;return!!t||NAt(e)}`, `function db(e,t){return Mr()&&(t!==void 0?!!t:$Cn(Ge()))}`),
		replaceFirstFixed(data, `...Mr()&&{fastMode:NAt(je??null)}`, `fastMode:NAt(je??null)`),
		replaceFirstFixed(data, `...He.gates.fastModeEnabled&&{fastMode:At.options.fastMode}`, `fastMode:At.options.fastMode`),
		replaceFirstFixed(data, `d={model:r.model,...Mr()&&{fastMode:r.fastMode}}`, `d={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Mr()&&{fastMode:t.fastMode}`, `fastMode:t.fastMode`),
	}
	if bytes.Count(data, []byte(`...Mr()&&{fastMode:ba}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...Mr()&&{fastMode:ba}`, `fastMode:ba`),
		replaceFirstFixed(data, `...Mr()?{fastMode:ba}:!1`, `fastMode:ba`),
		replaceFirstFixed(data, `if(Mr()&&_(()=>Jy())&&!Sse()&&_(()=>af(_e))&&!!et.fastMode)eW="fast";`, `if(et.fastMode)eW="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_263(data []byte) bool {
	return replaceFirstFixed(data, "function Ese(e){return`${jx(e.inputTokens)}/${jx(e.outputTokens)} per Mtok`}", `function Ese(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_263(data []byte) bool {
	return replaceClaude208Function(data, "function YS(w,I,O){", "var XS=", `function YS(w,I,O){return null}`)
}

func patchRemoteControlRuntimeFunctions_2_1_263(data []byte) bool {
	for _, transformation := range claude263RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude263RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function RH(){return}function oG(){return}", "function t(e){", `function RH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function oG(){return}function m_(){return RH()||Yt()?.accessToken}async function wC(e){return m_()}function Ype(){return oG()??Vt().BASE_API_URL}function qre(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function lb(){if(u())return!0;if(KG())return!1;return!UC()&&R$e()}`, `function lb(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function RAn(){if(u())return!0;return!KG()&&!UC()&&u6()}`, `function RAn(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function kAn(){if(u())return!0;if(KG())return!1;return u6()&&!UC()&&l()&&await od("tengu_ccr_bridge")}`, `async function kAn(){return!KG()&&!UC()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function T4t(){", "function T(){", `async function T4t(){if(KG())return"Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).";if(UC())return"Remote Control is not available inside a cloud session.";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.";return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(lb())return!0;try{return u6()&&!UC()&&!KG()&&Gl().source==="none"&&kp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!Myn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!lb()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude263Transformations(version string) []claude258Transformation {
	return claude263TransformationsForConfig(version, "2.1.263", modelconfig.Default())
}

func claude263TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude263SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude263EmbeddedPatchedModuleBytecode})
}

func claude263SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_263(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_261},
		{"usage", patchUsageFetchFunction_2_1_263},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_263(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_263},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_263},
		{"model-selection", patchModelPickerSelectionValue_2_1_263},
		{"agent-model-validator", patchAgentModelValidator_2_1_263},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_263},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_258},
		{"fast-mode-pricing", patchFastModePricing_2_1_263},
		{"context-warning", patchContextWarningHint_2_1_263},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_261},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_263},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude263UIBrandingReplacements)
		}},
	}
}

func claude263ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX263("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function lto("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
