package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude276SHA = "9de364db11a410d53cbbb0f6b1f18c66c90053efc9a63370072856d10db66329"

var claudeUIPatch_2_1_276 = claudeUIPatchSpec{
	Version: "2.1.276",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude276SHA,
	Apply:   applyClaudeUIPatches_2_1_276,
}

var claude276UIBrandingReplacements = claude274UIBrandingReplacements

func applyClaudeUIPatches_2_1_276(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude276UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude276EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude276SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude276ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude276EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude276EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude276EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX276(")) && !bytes.Contains(entrySource, []byte("function XCo(e=!1){")) {
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

func claude276EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude276ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude276EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude276EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude276RequiredLogoAnchor() []byte {
	return []byte("function hWe(){let l=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_276(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function hWe(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,h=Qur(),b=a.DEMO_VERSION?"/code/claude":No(ne()),x=a.CLAUDE_CODE_HIDE_CWD?"":h?` + "`${b} in ${h.replace(/^https?:\\/\\//,\"\")}`" + `:b,k="Codex Plan",O=Ke().agent;return{version:l,cwd:x,billingType:k,agentName:O}}`
	return replaceClaude208Function(data, string(claude276RequiredLogoAnchor()), "function f0n(l,h,b){", replacement)
}

func patchWhatsNewFeedFunction_2_1_276(data []byte) bool {
	const replacement = `var ee=async(i,t)=>{return u("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var ee=async(i,t)=>{try{", "function x(X){", replacement)
}

func patchUsageFetchFunction_2_1_276(data []byte) bool {
	const replacement = `function Kbn(e,n){return Or(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=HOo[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function Kbn(e,n){", "var sdt=", replacement)
}

func patchModelPickerOptions_2_1_276(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX276(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts276(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function XCo(e=!1){return CDXOpts276(e)}`
	return replaceClaude208Function(data, "function XCo(e=!1){", "function BC(e){", replacement)
}

func patchModelPickerResolver_2_1_276(data []byte) bool {
	return replaceClaude208Function(data, "function oRo(e,n){", "function I3n(e){", `function oRo(e,n){return CDXOpts276(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_276(data []byte) bool {
	return replaceClaude208Function(data, "function w6(e=!1){", "var ZCo=", `function w6(e=!1){return CDXOpts276(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_276(data []byte) bool {
	return replaceClaude208Function(data, "function gfe(e=!1,n=null){", "function eRo(e,n){", `function gfe(e=!1,n=null){return CDXOpts276(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_276(data []byte) bool {
	const replacement = `function BFt(e,n){let r=CDX276(n),s=e.find((g)=>g.value===r||CDX276(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function BFt(e,n){if(e.some((g)=>g.value===n))return n;", "function Usn(){", replacement)
}

func patchAgentModelValidator_2_1_276(data []byte) bool {
	return replaceFirstFixed(data, `model:V(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_276(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Zr(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Zr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function x0(){return"Opus 5"}`, `function x0(){return"Codex"}`),
		replaceFirstFixed(data, `function HXe(){return"opus"+(Jv()?"[1m]":"")}`, `function HXe(){return"opus"}`),
		replaceFirstFixed(data, `function Ftr(e,n){if(!Zr())return!1;return!!e&&(jt()||SE()||n)}`, `function Ftr(e,n){return Zr()&&!!e}`),
		replaceFirstFixed(data, `function SVt(e){if(!Zr())return!1;if(!SE(e))return!1;if(!rg(e))return!1;return gH(Ke())}`, `function SVt(e){return Zr()&&(ge("flagSettings")?.fastMode===!0||gH(Ke()))}`),
		replaceFirstFixed(data, `function gH(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function gH(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function rg(e){if(!Zr())return!1;let n=e??Sy(),r=Rt(n),s=Tm(ze(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function rg(e){return Zr()}`),
		replaceFirstFixed(data, `function Yv(e,n){if(jt()){if(e===null)return!!n;return!!n&&rg(e)}if(!rg(e))return!1;return!!n||SVt(e)}`, `function Yv(e,n){return Zr()&&(n!==void 0?!!n:gH(Ke()))}`),
		replaceFirstFixed(data, `g={model:r.model,...Zr()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Qt.gates.fastModeEnabled&&{fastMode:M.options.fastMode}`, `fastMode:M.options.fastMode`),
		replaceFirstFixed(data, `...Zr()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Zr()&&{fastMode:SVt(ee??null)}`, `fastMode:SVt(ee??null)`),
		replaceFirstFixed(data, `...Zr()?{fastMode:uE}:!1`, `fastMode:uE`),
		replaceFirstFixed(data, `if(Zr()&&y(()=>SE())&&!zhe()&&y(()=>rg(De))&&!!hr.fastMode)BR="fast";`, `if(hr.fastMode)BR="fast";`),
	}
	if bytes.Count(data, []byte(`...Zr()&&{fastMode:uE}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Zr()&&{fastMode:uE}`, `fastMode:uE`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_276(data []byte) bool {
	return replaceFirstFixed(data, "function qhe(e){return`${bH(e.inputTokens)}/${bH(e.outputTokens)} per Mtok`}", `function qhe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_276(data []byte) bool {
	return replaceClaude208Function(data, "function mf(h,k,E){", "var ff=", `function mf(h,k,E){return null}`)
}

func patchRemoteControlRuntimeFunctions_2_1_276(data []byte) bool {
	for _, transformation := range claude276RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude276RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function wD(){return}function I8(){return}", "function t(e){", `function wD(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function I8(){return}function QS(){return wD()||Qt()?.accessToken}async function dE(e){return QS()}function Vk(){return I8()??Yt().BASE_API_URL}function Tge(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function jv(){if(l())return!0;if(OD())return!1;return!nR()&&PYe()}`, `function jv(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function dZn(){if(l())return!0;return!OD()&&!nR()&&z$()}`, `function dZn(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function pZn(){if(l())return!0;if(OD())return!1;return z$()&&!nR()&&c()&&await Rp("tengu_ccr_bridge")}`, `async function pZn(){return!OD()&&!nR()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function GSn(){", "async function L(){", `async function GSn(){if(OD())return i("Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(nR())return i("Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(jv())return!0;try{return z$()&&!nR()&&!OD()&&Uc().source==="none"&&_f({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!L5n.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!jv()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude276Transformations(version string) []claude258Transformation {
	return claude276TransformationsForConfig(version, "2.1.276", modelconfig.Default())
}

func claude276TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude276SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude276EmbeddedPatchedModuleBytecode})
}

func claude276SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_276(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_274},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_276},
		{"usage", patchUsageFetchFunction_2_1_276},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_276(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_276},
		{"model-list", patchModelListOptions_2_1_276},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_276},
		{"model-selection", patchModelPickerSelectionValue_2_1_276},
		{"agent-model-validator", patchAgentModelValidator_2_1_276},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_276},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_273},
		{"fast-mode-pricing", patchFastModePricing_2_1_276},
		{"context-warning", patchContextWarningHint_2_1_276},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_273},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_276},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude276UIBrandingReplacements)
		}},
	}
}

func claude276ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX276("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function BC("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
