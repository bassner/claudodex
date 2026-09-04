package launcher

import (
	"bytes"
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_260 = claudeUIPatchSpec{
	Version: "2.1.260",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "3c269f66801028823e24a63ced9fdd3988cb86cf85fccd9f03f87e463b9d3e3c",
	Apply:   applyClaudeUIPatches_2_1_260,
}

const (
	claude260ModelOptionsOverrideTarget = `At=q(()=>Oe??scn(jt),[Oe,jt])`
	claude260ModelSelectionSourceTarget = `Yt=q(()=>qte(H,w),[H,w,io,wt]),ao=Yt??w,to=ao===null?Lv:vpt(At,ao)??ao,`
	claude260ModelExtraOptionsTarget    = `mn=q(()=>{let Li=[];for(let[Cs,Qs,Aa]of[[So.current,So.value,"Current model"],[So.sessionOverride===null?null:w,w===null?Lv:vpt(At,w)??w,"Base model"]])if(Cs!==null&&!At.some((Di)=>Di.value===Qs)&&!Li.some((Di)=>Di.value===Cs)&&Tr(Cs))Li.push({value:Cs,label:vC(Cs),description:Aa});if(Li.length===0)return At;let ks=At.findIndex((Cs)=>Cs.disabled===!0);if(ks===-1)return[...At,...Li];return[...At.slice(0,ks),...Li,...At.slice(ks)]},[At,So,w])`
	claude260ModelPickerValueTarget     = `defaultValue:to,selectedValue:to,defaultFocusValue:nn,options:Sr,`
)

var claude260UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude258UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "Claude needs your permission" {
			replacements[index].expectedCount = 6
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_260(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude260UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude260EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude260SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude260EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, false
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
		return nil, false
	}
	sectionStart := int(section.Offset)
	sectionEnd := sectionStart + int(section.Size)
	if sectionStart < 0 || sectionEnd < sectionStart || sectionEnd > len(data) {
		return nil, false
	}
	const trailer = "\n---- Bun! ----\n"
	trailerRelative := bytes.LastIndex(data[sectionStart:sectionEnd], []byte(trailer))
	if trailerRelative < claude259BunOffsetsSize {
		return nil, false
	}
	trailerOffset := sectionStart + trailerRelative
	offsetsOffset := trailerOffset - claude259BunOffsetsSize
	modulesOffset := int(binary.LittleEndian.Uint32(data[offsetsOffset+8 : offsetsOffset+12]))
	modulesLength := int(binary.LittleEndian.Uint32(data[offsetsOffset+12 : offsetsOffset+16]))
	if modulesLength == 0 || modulesLength%claude259BunModuleRecordSize != 0 {
		return nil, false
	}
	dataStart := sectionStart + 8
	moduleTableStart := dataStart + modulesOffset
	if moduleTableStart < dataStart || moduleTableStart+modulesLength > offsetsOffset {
		return nil, false
	}
	records := make([]claude259BunModuleRecord, 0, modulesLength/claude259BunModuleRecordSize)
	for moduleOffset := moduleTableStart; moduleOffset < moduleTableStart+modulesLength; moduleOffset += claude259BunModuleRecordSize {
		contentOffset := int(binary.LittleEndian.Uint32(data[moduleOffset+8 : moduleOffset+12]))
		contentLength := int(binary.LittleEndian.Uint32(data[moduleOffset+12 : moduleOffset+16]))
		bytecodeOffset := int(binary.LittleEndian.Uint32(data[moduleOffset+24 : moduleOffset+28]))
		bytecodeLength := int(binary.LittleEndian.Uint32(data[moduleOffset+28 : moduleOffset+32]))
		contentStart := dataStart + contentOffset
		if contentLength <= 0 || contentStart < dataStart || contentStart+contentLength > offsetsOffset {
			continue
		}
		if bytecodeLength > 0 && (dataStart+bytecodeOffset < dataStart || dataStart+bytecodeOffset+bytecodeLength > offsetsOffset) {
			return nil, false
		}
		records = append(records, claude259BunModuleRecord{
			bytecodeOffset: moduleOffset + 24,
			bytecodeLength: moduleOffset + 28,
			contentOffset:  contentStart,
			contentLength:  contentLength,
		})
	}
	if len(records) == 0 {
		return nil, false
	}
	return records, true
}

func claude260EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude260EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX260(")) && !bytes.Contains(entrySource, []byte("function scn(e=!1){")) {
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

func claude260EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	records, ok := claude260EmbeddedBunModuleRecords(data)
	if !ok {
		return nil, nil, false
	}
	hashes := make([][sha256.Size]byte, len(records))
	for index, record := range records {
		hashes[index] = sha256.Sum256(data[record.contentOffset : record.contentOffset+record.contentLength])
	}
	return records, hashes, true
}

func disableClaude260ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	if len(records) == 0 || len(records) != len(hashes) {
		return false
	}
	changed := 0
	for index, record := range records {
		current := sha256.Sum256(data[record.contentOffset : record.contentOffset+record.contentLength])
		if current == hashes[index] {
			continue
		}
		changed++
		if binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
			continue
		}
		binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
		binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	}
	return changed > 0
}

func disableClaude260EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude260EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func patchLogoDisplayDataFunction_2_1_260(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Lxe(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,b=B0n(),x=a.DEMO_VERSION?"/code/claude":Ao(Q()),O=a.CLAUDE_CODE_HIDE_CWD?"":b?` + "`${x} in ${b.replace(/^https?:\\/\\//,\"\")}`" + `:x,R="Codex Plan",v=qe().agent;return{version:l,cwd:O,billingType:R,agentName:v}}`
	return replaceClaude208Function(data, "function Lxe(){let l=a.DEMO_VERSION??", "function LYt(l,b,x){", replacement)
}

func patchWhatsNewFeedFunction_2_1_260(data []byte) bool {
	const replacement = `var ee=async(s,t)=>{return y("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,s),null};`
	return replaceClaude208Function(data, "var ee=async(s,t)=>{try{", "function x(L){", replacement)
}

func patchUsageFetchFunction_2_1_260(data []byte) bool {
	const replacement = `async function hO(e,{atWall:n=!1}={}){return Sr(n?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),o=n?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",d=await fetch(r+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()})}`
	return replaceClaude208Function(data, "async function hO(e,{atWall:n=!1}={}){", "var u4e=", replacement)
}

func patchModelPickerOptions_2_1_260(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX260(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function scn(e=!1){return Gce(e)}function Gce(e=!1,n=null){return CDXOpts260(e,n)}function CDXOpts260(e=!1,n=null){let t=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}`
	return replaceClaude208Function(data, "function scn(e=!1){", "function dQr(e,n){", replacement)
}

func patchModelPickerResolver_2_1_260(data []byte) bool {
	return replaceClaude208Function(data, "function gQr(e,n){", "function icn(e){", `function gQr(e,n){return CDXOpts260(e,n).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_260(data []byte) bool {
	if bytes.Count(data, []byte(claude260ModelOptionsOverrideTarget)) != 1 ||
		bytes.Count(data, []byte(claude260ModelSelectionSourceTarget)) != 1 ||
		bytes.Count(data, []byte(claude260ModelExtraOptionsTarget)) != 1 ||
		bytes.Count(data, []byte(claude260ModelPickerValueTarget)) != 1 {
		return false
	}
	return replaceFirstFixed(data, claude260ModelOptionsOverrideTarget, `At=q(()=>scn(jt),[jt])`) &&
		replaceFirstFixed(data, claude260ModelSelectionSourceTarget, `Yt=q(()=>qte(H,w),[H,w,io,wt]),ao=Yt??w,to=ao===null?Lv:vpt(At,ao)??ao,`) &&
		replaceFirstFixed(data, claude260ModelExtraOptionsTarget, "mn=At.slice(0,3)") &&
		replaceFirstFixed(data, claude260ModelPickerValueTarget, `defaultValue:to,defaultFocusValue:nn,options:Sr.slice(0,3),`)
}

func patchModelPickerSelectionValue_2_1_260(data []byte) bool {
	const replacement = `function vpt(e,n){let r=CDX260(n),o=e.find((d)=>d.value===r||CDX260(d.value)===r);return o?.value??r}`
	return replaceClaude208Function(data, "function vpt(e,n){if(e.some((d)=>d.value===n))return n;", "function ctn(){", replacement)
}

func patchAgentModelValidator_2_1_260(data []byte) bool {
	return replaceFirstFixed(data, `model:Y(["sonnet","opus","haiku","fable"]).optional()`, `model:s().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_260(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Or(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Or(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function cR(){return"Opus 5"}`, `function cR(){return"Codex"}`),
		replaceFirstFixed(data, `function PFe(){return"opus"+(cw()?"[1m]":"")}`, `function PFe(){return"opus"}`),
		replaceFirstFixed(data, `function bEn(e,n){if(!Or())return!1;return!!e&&(It()||Gy()||n)}`, `function bEn(e,n){return Or()&&!!e}`),
		replaceFirstFixed(data, `function LTt(e){if(!Or())return!1;if(!Gy(e))return!1;if(!ef(e))return!1;return wEn(qe())}`, `function LTt(e){return Or()&&(he("flagSettings")?.fastMode===!0||wEn(qe()))}`),
		replaceFirstFixed(data, `function wEn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(he("policySettings")?.fastModePerSessionOptIn===!0)return!1;return he("flagSettings")?.fastMode===!0}`, `function wEn(e){return e.fastMode===!0}`),
		replaceFirstFixed(data, `function ef(e){if(!Or())return!1;let n=e??rh(),r=St(n);if(em(Ue(r),"fast_mode",r))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function ef(e){return Or()}`),
		replaceFirstFixed(data, `function QS(e,n){if(It()){if(e===null)return!!n;return!!n&&ef(e)}if(!ef(e))return!1;return!!n||LTt(e)}`, `function QS(e,n){return Or()&&(n!==void 0?!!n:wEn(qe()))}`),
		replaceFirstFixed(data, `...Or()&&{fastMode:LTt(_t??null)}`, `fastMode:LTt(_t??null)`),
		replaceFirstFixed(data, `...He.gates.fastModeEnabled&&{fastMode:Et.options.fastMode}`, `fastMode:Et.options.fastMode`),
		replaceFirstFixed(data, `d={model:r.model,...Or()&&{fastMode:r.fastMode}}`, `d={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Or()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
	}
	if bytes.Count(data, []byte(`...Or()&&{fastMode:Qa}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...Or()&&{fastMode:Qa}`, `fastMode:Qa`),
		replaceFirstFixed(data, `...Or()?{fastMode:Qa}:!1`, `fastMode:Qa`),
		replaceFirstFixed(data, `if(Or()&&_(()=>Gy())&&!koe()&&_(()=>ef(de))&&!!eo.fastMode)G$="fast";`, `if(eo.fastMode)G$="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_260(data []byte) bool {
	return replaceFirstFixed(data, "function Poe(e){return`${XD(e.inputTokens)}/${XD(e.outputTokens)} per Mtok`}", `function Poe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_260(data []byte) bool {
	return replaceClaude208Function(data, "function R_(w,U,H){", "var A_=", `function R_(w,U,H){return null}`)
}

func patchCompactProgressCurve_2_1_260(data []byte) bool {
	return replaceFirstFixed(data, `function wo(t){let s=Math.max(0,t)/1000,c=1-Math.exp(-s/90);return Math.min(95,Math.round(c*100))}`, `function wo(t){let s=Math.max(0,t)/2000,c=1-Math.exp(-s/90);return Math.min(95,Math.round(c*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_260(data []byte) bool {
	for _, transformation := range claude260RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude260RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function TH(){return}function L3(){return}", "function t(e){", `function TH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function L3(){return}function c_(){return TH()||Vt()?.accessToken}async function cC(e){return c_()}function ype(){return L3()??zt().BASE_API_URL}function hre(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function KS(){if(s())return!0;if(hG())return!1;return!EC()&&dFe()}`, `function KS(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function dTn(){if(s())return!0;return!hG()&&!EC()&&$6()}`, `function dTn(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function pTn(){if(s())return!0;if(hG())return!1;return $6()&&!EC()&&u()&&await Yu("tengu_ccr_bridge")}`, `async function pTn(){return!hG()&&!EC()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function mqt(){", "function T(){", `async function mqt(){if(hG())return"Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).";if(EC())return"Remote Control is not available inside a cloud session.";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.";return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(KS())return!0;try{return $6()&&!EC()&&!hG()&&Ml().source==="none"&&vp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!Ehn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!KS()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude260Transformations(version string) []claude258Transformation {
	return claude260TransformationsForConfig(version, "2.1.260", modelconfig.Default())
}

func claude260TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude260SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude260EmbeddedPatchedModuleBytecode})
}

func claude260SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_260(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_260},
		{"usage", patchUsageFetchFunction_2_1_260},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_260(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_260},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_260},
		{"model-selection", patchModelPickerSelectionValue_2_1_260},
		{"agent-model-validator", patchAgentModelValidator_2_1_260},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_260},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_258},
		{"fast-mode-pricing", patchFastModePricing_2_1_260},
		{"context-warning", patchContextWarningHint_2_1_260},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_260},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_260},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude260UIBrandingReplacements)
		}},
	}
}

func claude260ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX260("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function dQr("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
