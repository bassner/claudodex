package launcher

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_259 = claudeUIPatchSpec{
	Version: "2.1.259",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "884baa38fe1a624be25c4a91568bf5a08b5cf4e7d7acf29b7760e3525d964898",
	Apply:   applyClaudeUIPatches_2_1_259,
}

const (
	claude259ModelOptionsOverrideTarget = `ct=K(()=>Pe??Ufn(rt),[Pe,rt])`
	claude259ModelSelectionSourceTarget = `Rt=Ye??w,pt=Rt===null?Cw:wgt(ct,Rt)??Rt,`
	claude259ModelExtraOptionsTarget    = `po=K(()=>{let Ji=[];for(let[hi,us,Ps]of[[so.current,so.value,"Current model"],[so.sessionOverride===null?null:w,w===null?Cw:wgt(ct,w)??w,"Base model"]])if(hi!==null&&!ct.some((ps)=>ps.value===us)&&!Ji.some((ps)=>ps.value===hi)&&Mr(hi))Ji.push({value:hi,label:bv(hi),description:Ps});if(Ji.length===0)return ct;let Ss=ct.findIndex((hi)=>hi.disabled===!0);if(Ss===-1)return[...ct,...Ji];return[...ct.slice(0,Ss),...Ji,...ct.slice(Ss)]},[ct,so,w])`
	claude259ModelPickerValueTarget     = `defaultValue:pt,selectedValue:pt,defaultFocusValue:Bo,options:wn,`
)

var claude259UIBrandingReplacements = claude258UIBrandingReplacements

func applyClaudeUIPatches_2_1_259(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude259UIBrandingReplacements) {
		return false
	}
	transformations := []bool{
		patchLogoDisplayDataFunction_2_1_259(data, claudodexVersion, claudeVersion),
		patchActiveHeaderBrand_2_1_258(data),
		patchDefaultTierLabel_2_1_258(data),
		patchWhatsNewFeedFunction_2_1_259(data),
		patchUsageFetchFunction_2_1_259(data),
		patchModelPickerOptions_2_1_259(data, modelCfg),
		patchModelPickerResolver_2_1_259(data),
		patchModelPickerExtraOptions_2_1_259(data),
		patchModelPickerSelectionValue_2_1_259(data),
		patchAgentModelValidator_2_1_258(data),
		patchFastModeRuntimeFunctions_2_1_259(data),
		patchActiveFastModeBrand_2_1_258(data),
		patchFastModePricing_2_1_259(data),
		patchContextWarningHint_2_1_259(data),
		patchResumeCommandHints_2_1_258(data),
		patchCompactProgressCurve_2_1_259(data),
		patchRemoteControlRuntimeFunctions_2_1_259(data),
		applyClaude209UIBrandingReplacements(data, claude259UIBrandingReplacements),
		disableClaude259EmbeddedPatchedModuleBytecode(data),
	}
	for _, transformed := range transformations {
		if !transformed {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return true
}

const (
	claude259BunModuleRecordSize = 52
	claude259BunOffsetsSize      = 32
)

type claude259BunModuleRecord struct {
	bytecodeOffset int
	bytecodeLength int
	contentOffset  int
	contentLength  int
}

func claude259EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return claude259BunModuleRecord{}, false
	}
	defer file.Close()

	var section *macho.Section
	for _, candidate := range file.Sections {
		if candidate.Seg == "__BUN" && candidate.Name == "__bun" {
			section = candidate
			break
		}
	}
	if section == nil || section.Size < 8+claude259BunOffsetsSize {
		return claude259BunModuleRecord{}, false
	}
	sectionStart := int(section.Offset)
	sectionEnd := sectionStart + int(section.Size)
	if sectionStart < 0 || sectionEnd < sectionStart || sectionEnd > len(data) {
		return claude259BunModuleRecord{}, false
	}
	const trailer = "\n---- Bun! ----\n"
	trailerRelative := bytes.LastIndex(data[sectionStart:sectionEnd], []byte(trailer))
	if trailerRelative < claude259BunOffsetsSize {
		return claude259BunModuleRecord{}, false
	}
	trailerOffset := sectionStart + trailerRelative
	offsetsOffset := trailerOffset - claude259BunOffsetsSize
	modulesOffset := int(binary.LittleEndian.Uint32(data[offsetsOffset+8 : offsetsOffset+12]))
	modulesLength := int(binary.LittleEndian.Uint32(data[offsetsOffset+12 : offsetsOffset+16]))
	if modulesLength == 0 || modulesLength%claude259BunModuleRecordSize != 0 {
		return claude259BunModuleRecord{}, false
	}
	dataStart := sectionStart + 8
	moduleTableStart := dataStart + modulesOffset
	if moduleTableStart < dataStart || moduleTableStart+modulesLength > offsetsOffset {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for moduleOffset := moduleTableStart; moduleOffset < moduleTableStart+modulesLength; moduleOffset += claude259BunModuleRecordSize {
		contentOffset := int(binary.LittleEndian.Uint32(data[moduleOffset+8 : moduleOffset+12]))
		contentLength := int(binary.LittleEndian.Uint32(data[moduleOffset+12 : moduleOffset+16]))
		bytecodeOffset := int(binary.LittleEndian.Uint32(data[moduleOffset+24 : moduleOffset+28]))
		bytecodeLength := int(binary.LittleEndian.Uint32(data[moduleOffset+28 : moduleOffset+32]))
		contentStart := dataStart + contentOffset
		if contentLength <= 0 || contentStart < dataStart || contentStart+contentLength > offsetsOffset {
			continue
		}
		entrySource := data[contentStart : contentStart+contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX259(")) && !bytes.Contains(entrySource, []byte("function Ufn(e=!1){")) {
			continue
		}
		if bytecodeLength > 0 && (dataStart+bytecodeOffset < dataStart || dataStart+bytecodeOffset+bytecodeLength > offsetsOffset) {
			return claude259BunModuleRecord{}, false
		}
		if found != nil {
			return claude259BunModuleRecord{}, false
		}
		record := claude259BunModuleRecord{
			bytecodeOffset: moduleOffset + 24,
			bytecodeLength: moduleOffset + 28,
			contentOffset:  contentStart,
			contentLength:  contentLength,
		}
		found = &record
	}
	if found == nil {
		return claude259BunModuleRecord{}, false
	}
	return *found, true
}

func claude259EmbeddedPatchedModuleBytecodeLength(data []byte) (uint32, bool) {
	record, ok := claude259EmbeddedPatchedModuleRecord(data)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint32(data[record.bytecodeLength : record.bytecodeLength+4]), true
}

func disableClaude259EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude259EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func patchLogoDisplayDataFunction_2_1_259(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function x0e(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,b=qDn(),O=a.DEMO_VERSION?"/code/claude":Lo(ne()),T=a.CLAUDE_CODE_HIDE_CWD?"":b?` + "`${O} in ${b.replace(/^https?:\\/\\//,\"\")}`" + `:O,w="Codex Plan",S=Je().agent;return{version:l,cwd:T,billingType:w,agentName:S}}`
	return replaceClaude208Function(data, "function x0e(){let l=a.DEMO_VERSION??", "function LZt(l,b,O){", replacement)
}

func patchWhatsNewFeedFunction_2_1_259(data []byte) bool {
	const replacement = `var ee=async(s,t)=>{return h("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,s),null};`
	return replaceClaude208Function(data, "var ee=async(s,t)=>{try{", "function D(X){", replacement)
}

func patchUsageFetchFunction_2_1_259(data []byte) bool {
	const replacement = `async function ZO(e,{atWall:n=!1}={}){return Cr(n?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),o=n?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",f=await fetch(r+o,{headers:{"Content-Type":"application/json"}});if(!f.ok)throw Error("Auth error: "+f.status);return await f.json()})}`
	return replaceClaude208Function(data, "async function ZO(e,{atWall:n=!1}={}){", "var eKe=", replacement)
}

func patchModelPickerOptions_2_1_259(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX259(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function Ufn(e=!1){return vTe(e)}function vTe(e=!1,n=null){let t=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}`
	return replaceClaude208Function(data, "function Ufn(e=!1){", "function _ro(e,n){", replacement)
}

func patchModelPickerResolver_2_1_259(data []byte) bool {
	return replaceClaude208Function(data, "function wro(e,n){", "function Bfn(e){", `function wro(e,n){return vTe(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_259(data []byte) bool {
	if bytes.Count(data, []byte(claude259ModelOptionsOverrideTarget)) != 1 ||
		bytes.Count(data, []byte(claude259ModelSelectionSourceTarget)) != 1 ||
		bytes.Count(data, []byte(claude259ModelExtraOptionsTarget)) != 1 ||
		bytes.Count(data, []byte(claude259ModelPickerValueTarget)) != 1 {
		return false
	}
	return replaceFirstFixed(data, claude259ModelOptionsOverrideTarget, `ct=K(()=>Ufn(rt),[rt])`) &&
		replaceFirstFixed(data, claude259ModelSelectionSourceTarget, `Rt=Ye??w,pt=CDX259(Rt===null?Cw:Rt),`) &&
		replaceFirstFixed(data, claude259ModelExtraOptionsTarget, "po=ct.slice(0,3)") &&
		replaceFirstFixed(data, claude259ModelPickerValueTarget, `options:wn.slice(0,3),`)
}

func patchModelPickerSelectionValue_2_1_259(data []byte) bool {
	const replacement = `function wgt(e,n){let r=CDX259(n),o=e.find((f)=>f.value===r||CDX259(f.value)===r);return o?.value??r}`
	return replaceClaude208Function(data, "function wgt(e,n){if(e.some((f)=>f.value===n))return n;", "function Esn(){", replacement)
}

func patchFastModeRuntimeFunctions_2_1_259(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function $r(){if(Me()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function $r(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function FR(){return"Opus 5"}`, `function FR(){return"Codex"}`),
		replaceFirstFixed(data, `function cUe(){return"opus"+(Uw()?"[1m]":"")}`, `function cUe(){return"opus"}`),
		replaceFirstFixed(data, `function wvn(e,n){if(!$r())return!1;return!!e&&(qt()||bS()||n)}`, `function wvn(e,n){return $r()&&!!e}`),
		replaceFirstFixed(data, `function DAt(e){if(!$r())return!1;if(!bS(e))return!1;if(!hf(e))return!1;return Tvn(Je())}`, `function DAt(e){return $r()&&(be("flagSettings")?.fastMode===!0||Tvn(Je()))}`),
		replaceFirstFixed(data, `function Tvn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(be("policySettings")?.fastModePerSessionOptIn===!0)return!1;return be("flagSettings")?.fastMode===!0}`, `function Tvn(e){return e.fastMode===!0}`),
		replaceFirstFixed(data, `function hf(e){if(!$r())return!1;let n=e??xh(),r=Et(n);if(jf(ze(r),"fast_mode",r))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function hf(e){return $r()}`),
		replaceFirstFixed(data, `function Hb(e,n){if(qt()){if(e===null)return!!n;return!!n&&hf(e)}if(!hf(e))return!1;return!!n||DAt(e)}`, `function Hb(e,n){return $r()&&(n!==void 0?!!n:Tvn(Je()))}`),
		replaceFirstFixed(data, `...$r()&&{fastMode:DAt(ot??null)}`, `fastMode:DAt(ot??null)`),
		replaceFirstFixed(data, `...Le.gates.fastModeEnabled&&{fastMode:Ot.options.fastMode}`, `fastMode:Ot.options.fastMode`),
		replaceFirstFixed(data, `f={model:r.model,...$r()&&{fastMode:r.fastMode}}`, `f={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...$r()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
	}
	if bytes.Count(data, []byte(`...$r()&&{fastMode:Vr}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...$r()&&{fastMode:Vr}`, `fastMode:Vr`),
		replaceFirstFixed(data, `...$r()?{fastMode:Vr}:!1`, `fastMode:Vr`),
		replaceFirstFixed(data, `if($r()&&y(()=>bS())&&!qie()&&y(()=>hf(_e))&&!!Br.fastMode)ch="fast";`, `if(Br.fastMode)ch="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_259(data []byte) bool {
	return replaceFirstFixed(data, "function Xie(e){return`${lD(e.inputTokens)}/${lD(e.outputTokens)} per Mtok`}", `function Xie(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_259(data []byte) bool {
	return replaceClaude208Function(data, "function Q_(w,T,I){", "var eb=", `function Q_(w,T,I){return null}`)
}

func patchCompactProgressCurve_2_1_259(data []byte) bool {
	return replaceFirstFixed(data, `function wo(t){let i=Math.max(0,t)/1000,c=1-Math.exp(-i/90);return Math.min(95,Math.round(c*100))}`, `function wo(t){let i=Math.max(0,t)/2000,c=1-Math.exp(-i/90);return Math.min(95,Math.round(c*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_259(data []byte) bool {
	for _, transformation := range claude259RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude259RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function p0(){return}function jG(){return}", "function kCr(e){", `function p0(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function jG(){return}function O_(){return p0()||tn()?.accessToken}async function uv(e){return O_()}function mme(){return jG()??Jt().BASE_API_URL}function oie(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return kCr(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function vb(){if(s())return!0;if(iq())return!1;return!UA()&&W$e()}`, `function vb(){return!!p0()}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function pCn(){if(s())return!0;return!iq()&&!UA()&&Bj()}`, `function pCn(){return!!p0()}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function fCn(){if(s())return!0;if(iq())return!1;return Bj()&&!UA()&&u()&&await Ku("tengu_ccr_bridge")}`, `async function fCn(){return!iq()&&!UA()&&!!p0()}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Xzt(){", "function T(){", `async function Xzt(){if(iq())return"Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).";if(UA())return"Remote Control is not available inside a cloud session.";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.";return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(vb())return!0;try{return Bj()&&!UA()&&!iq()&&Yl().source==="none"&&Gp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!Cbn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!vb()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude259Transformations(version string) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_259(data, version, "2.1.259") }},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_259},
		{"usage", patchUsageFetchFunction_2_1_259},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_259(data, modelconfig.Default()) }},
		{"model-resolver", patchModelPickerResolver_2_1_259},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_259},
		{"model-selection", patchModelPickerSelectionValue_2_1_259},
		{"agent-model-validator", patchAgentModelValidator_2_1_258},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_259},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_258},
		{"fast-mode-pricing", patchFastModePricing_2_1_259},
		{"context-warning", patchContextWarningHint_2_1_259},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_259},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_259},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude259UIBrandingReplacements)
		}},
		{"patched-module-bytecode", disableClaude259EmbeddedPatchedModuleBytecode},
	}
}

func claude259ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX259("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function _ro("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
