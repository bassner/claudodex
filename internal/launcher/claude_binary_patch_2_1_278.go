package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude278SHA = "bd245662fb8a0e321b3bf133e930371d6563c387527885f30b2613aef3ba14d6"

var claudeUIPatch_2_1_278 = claudeUIPatchSpec{
	Version: "2.1.278",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude278SHA,
	Apply:   applyClaudeUIPatches_2_1_278,
}

var claude278UIBrandingReplacements = claude274UIBrandingReplacements

func applyClaudeUIPatches_2_1_278(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude278UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude278EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude278SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude278ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude278EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude278EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude278EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX278(")) && !bytes.Contains(entrySource, []byte("function zEr(e=!1){")) {
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

func claude278EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude278ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude278EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude278EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude278RequiredLogoAnchor() []byte {
	return []byte("function K6e(){let l=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_278(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function K6e(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,h=Ggr(),b=a.DEMO_VERSION?"/code/claude":Io(ne()),x=a.CLAUDE_CODE_HIDE_CWD?"":h?` + "`${b} in ${h.replace(/^https?:\\/\\//,\"\")}`" + `:b,k="Codex Plan",O=Ke().agent;return{version:l,cwd:x,billingType:k,agentName:O}}`
	return replaceClaude208Function(data, string(claude278RequiredLogoAnchor()), "function DDn(l,h,b){", replacement)
}

func patchWhatsNewFeedFunction_2_1_278(data []byte) bool {
	const replacement = `var ee=async(i,t)=>{return u("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var ee=async(i,t)=>{try{", "function x(X){", replacement)
}

func patchUsageFetchFunction_2_1_278(data []byte) bool {
	const replacement = `function t0t(e,n){return Mr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=TPr[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function t0t(e,n){", "var yft=", replacement)
}

func patchModelPickerOptions_2_1_278(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX278(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts278(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function zEr(e=!1){return CDXOpts278(e)}`
	return replaceClaude208Function(data, "function zEr(e=!1){", "function lS(e){", replacement)
}

func patchModelPickerResolver_2_1_278(data []byte) bool {
	return replaceClaude208Function(data, "function JEr(e,n){", "function QKn(e){", `function JEr(e,n){return CDXOpts278(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_278(data []byte) bool {
	return replaceClaude208Function(data, "function Q6(e=!1){", "var qEr=", `function Q6(e=!1){return CDXOpts278(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_278(data []byte) bool {
	return replaceClaude208Function(data, "function Pme(e=!1,n=null){", "function VEr(e,n){", `function Pme(e=!1,n=null){return CDXOpts278(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_278(data []byte) bool {
	const replacement = `function kBt(e,n){let r=CDX278(n),s=e.find((g)=>g.value===r||CDX278(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function kBt(e,n){if(e.some((g)=>g.value===n))return n;", "function QIt(){", replacement)
}

func patchAgentModelValidator_2_1_278(data []byte) bool {
	return replaceFirstFixed(data, `model:z(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_278(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Zr(){if(Oe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Zr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function J0(){return"Opus 5"}`, `function J0(){return"Codex"}`),
		replaceFirstFixed(data, `function DJe(){return"opus"+(uA()?"[1m]":"")}`, `function DJe(){return"opus"}`),
		replaceFirstFixed(data, `function Hir(e,n){if(!Zr())return!1;return!!e&&(Nt()||GE()||n)}`, `function Hir(e,n){return Zr()&&!!e}`),
		replaceFirstFixed(data, `function OKt(e){if(!Zr())return!1;if(!GE(e))return!1;if(!Sg(e))return!1;return NV(Ke())}`, `function OKt(e){return Zr()&&(ge("flagSettings")?.fastMode===!0||NV(Ke()))}`),
		replaceFirstFixed(data, `function NV(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function NV(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Sg(e){if(!Zr())return!1;let n=e??Ly(),r=Rt(n),s=Um(Ge(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Sg(e){return Zr()}`),
		replaceFirstFixed(data, `function lA(e,n){if(Nt()){if(e===null)return!!n;return!!n&&Sg(e)}if(!Sg(e))return!1;return!!n||OKt(e)}`, `function lA(e,n){return Zr()&&(n!==void 0?!!n:NV(Ke()))}`),
		replaceFirstFixed(data, `g={model:r.model,...Zr()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...qt.gates.fastModeEnabled&&{fastMode:M.options.fastMode}`, `fastMode:M.options.fastMode`),
		replaceFirstFixed(data, `...Zr()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Zr()&&{fastMode:OKt(ee??null)}`, `fastMode:OKt(ee??null)`),
		replaceFirstFixed(data, `...Zr()?{fastMode:SM}:!1`, `fastMode:SM`),
		replaceFirstFixed(data, `if(Zr()&&y(()=>GE())&&!a_e()&&y(()=>Sg(Le))&&!!Bn.fastMode)kb="fast";`, `if(Bn.fastMode)kb="fast";`),
	}
	if bytes.Count(data, []byte(`...Zr()&&{fastMode:SM}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Zr()&&{fastMode:SM}`, `fastMode:SM`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_278(data []byte) bool {
	return replaceFirstFixed(data, "function ice(e){return`${FV(e.inputTokens)}/${FV(e.outputTokens)} per Mtok`}", `function ice(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_278(data []byte) bool {
	return replaceClaude208Function(data, "function gc(o,u,f){", "var Sc=", `function gc(o,u,f){return null}`)
}

func patchRemoteControlRuntimeFunctions_2_1_278(data []byte) bool {
	for _, transformation := range claude278RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude278RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function XD(){return}function HY(){return}", "function t(e){", `function XD(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function HY(){return}function hb(){return XD()||Zt()?.accessToken}async function LE(e){return hb()}function dR(){return HY()??Xt().BASE_API_URL}function Fhe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function nA(){if(l())return!0;if(dL())return!1;return!ER()&&O7e()}`, `function nA(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function Vrr(){if(l())return!0;return!dL()&&!ER()&&kU()}`, `function Vrr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function qrr(){if(l())return!0;if(dL())return!1;return kU()&&!ER()&&c()&&await xd("tengu_ccr_bridge")}`, `async function qrr(){return!dL()&&!ER()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Qvn(){", "async function L(){", `async function Qvn(){if(dL())return i("Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(ER())return i("Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(nA())return!0;try{return kU()&&!ER()&&!dL()&&Xc().source==="none"&&Df({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!sJn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!nA()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude278Transformations(version string) []claude258Transformation {
	return claude278TransformationsForConfig(version, "2.1.278", modelconfig.Default())
}

func claude278TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude278SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude278EmbeddedPatchedModuleBytecode})
}

func claude278SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_278(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_274},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_278},
		{"usage", patchUsageFetchFunction_2_1_278},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_278(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_278},
		{"model-list", patchModelListOptions_2_1_278},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_278},
		{"model-selection", patchModelPickerSelectionValue_2_1_278},
		{"agent-model-validator", patchAgentModelValidator_2_1_278},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_278},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_273},
		{"fast-mode-pricing", patchFastModePricing_2_1_278},
		{"context-warning", patchContextWarningHint_2_1_278},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", func(data []byte) bool {
			return replaceFirstFixed(data, `function Hn(t){let o=Math.max(0,t)/1000,l=1-Math.exp(-o/90);return Math.min(95,Math.round(l*100))}`, `function Hn(t){let o=Math.max(0,t)/2000,l=1-Math.exp(-o/90);return Math.min(95,Math.round(l*100))}`)
		}},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_278},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude278UIBrandingReplacements)
		}},
	}
}

func claude278ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX278("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function lS("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
