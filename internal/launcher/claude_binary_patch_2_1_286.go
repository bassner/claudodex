package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude286SHA = "75e3016e9d2570767b08e43a7467d4817a4f149232c169ca295f2c95fef21433"

var claudeUIPatch_2_1_286 = claudeUIPatchSpec{
	Version: "2.1.286",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude286SHA,
	Apply:   applyClaudeUIPatches_2_1_286,
}

var claude286UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_286(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude286UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude286EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude286SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude286EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude286EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX286(")) && !bytes.Contains(entrySource, claude286RequiredLogoAnchor()) {
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

func disableClaude286EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude286EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude286RequiredLogoAnchor() []byte {
	return []byte("function GZe(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_286(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function GZe(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=M9r(),t=No(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",p=Je().agent;return{version:o,cwd:c,billingType:l,agentName:p}}`
	return replaceClaude208Function(data, string(claude286RequiredLogoAnchor()), "function Rir(o,i,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_286(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_286(data []byte) bool {
	const replacement = `async function pF(e,n){return Mr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Ise[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "async function pF(e,n){", "function Kg(e){", replacement)
}

func patchModelPickerOptions_2_1_286(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX286(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts286(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function The(e=!1,n=null){return CDXOpts286(e)}`
	return replaceClaude208Function(data, "function The(e=!1,n=null){", "function t3(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_286(data []byte) bool {
	const replacement = `function zsn(e,n){let r=CDX286(n),s=e.find((h)=>h.value===r||CDX286(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function zsn(e,n){", "function fM(){", replacement)
}

func patchAgentModelValidator_2_1_286(data []byte) bool {
	return replaceFirstFixed(data, `model:U(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_286(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function go(){if(He()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function go(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function y1(){`, `function are(){`, `function y1(){return"Codex"}`),
		replaceFirstFixed(data, `function are(){return"opus"+(b2()?"[1m]":"")}`, `function are(){return"opus"}`),
		replaceFirstFixed(data, `function gjr(e,n,r){if(!go())return!1;return K_e(e,r)&&(Lt()||wA()||n)}`, `function gjr(e,n,r){return go()&&!!e}`),
		replaceFirstFixed(data, `function nhn(e){if(!go())return!1;if(!wA(e))return!1;if(!dy(e))return!1;return rhn(Je())}`, `function nhn(e){return go()&&(ge("flagSettings")?.fastMode===!0||rhn(Je()))}`),
		replaceFirstFixed(data, `function rhn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function rhn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function dy(e){if(!go())return!1;let n=e??py(),r=It(n),s=Uh(Ue(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function dy(e){return go()}`),
		replaceFirstFixed(data, `function zT(e,n){if(Lt()){if(e===null)return!!n;return!!n&&dy(e)}if(!dy(e))return!1;return!!n||nhn(e)}`, `function zT(e,n){return go()&&(n!==void 0?!!n:rhn(Je()))}`),
		replaceFirstFixed(data, `h={model:r.model,...go()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...kt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...go()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...go()&&{fastMode:nhn(ze??null)}`, `fastMode:nhn(ze??null)`),
		replaceFirstFixed(data, `...go()?{fastMode:xS}:!1`, `fastMode:xS`),
		replaceFirstFixed(data, `if(go()&&W(()=>wA())&&!VRe()&&W(()=>dy(gt))&&!!Wn.fastMode&&!_Ut(Wn.model))YS="fast";`, `if(Wn.fastMode)YS="fast";`),
	}
	if bytes.Count(data, []byte(`...go()&&{fastMode:xS}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...go()&&{fastMode:xS}`, `fastMode:xS`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_286(data []byte) bool {
	return replaceFirstFixed(data, "function J_e(e){return`${f_(e.inputTokens)}/${f_(e.outputTokens)} per Mtok`}", `function J_e(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_286(data []byte) bool {
	return replaceClaude208Function(data, `function oc(e,o,n){let{source:r,window:s}=Ww`, "var nc=", `function oc(e,o,n){return null}`)
}

func claude286RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function $H(){return}function pne(){return}", "function t(e){", `function $H(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function pne(){return}function QE(){return $H()||pn()?.accessToken}async function sC(e){return QE()}function NI(){return pne()??cn().BASE_API_URL}function fke(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function sA(){if(u())return!0;if(U$())return!1;return!MI()&&N5e()}`, `function sA(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function uLr(){if(u())return!0;return!U$()&&!MI()&&WB()}`, `function uLr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function pLr(){`, `var I=`, `async function pLr(){return!MI()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function aVn(){if(MI())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function jdt(){", "async function x(){", `async function jdt(){if(MI())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(sA())return!0;try{return WB()&&!MI()&&!U$()&&dc().source==="none"&&wp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!ZOr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!sA()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude286SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_286(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_286},
		{"usage", patchUsageFetchFunction_2_1_286},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_286(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_286},
		{"agent-model-validator", patchAgentModelValidator_2_1_286},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_286},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_286},
		{"context-warning", patchContextWarningHint_2_1_286},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude286RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude286UIBrandingReplacements)
		}},
	}
}

func claude286Transformations(version string) []claude258Transformation {
	transformations := claude286SourceTransformationsForConfig(version, "2.1.286", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude286EmbeddedPatchedModuleBytecode})
}
