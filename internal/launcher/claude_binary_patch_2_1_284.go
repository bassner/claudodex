package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude284SHA = "50a14c2f50f56668380fdda490167f1d3630d5cc18fb8aed3073c2c7ea7314fe"

var claudeUIPatch_2_1_284 = claudeUIPatchSpec{
	Version: "2.1.284",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude284SHA,
	Apply:   applyClaudeUIPatches_2_1_284,
}

var claude284UIBrandingReplacements = claude283UIBrandingReplacements

func applyClaudeUIPatches_2_1_284(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude284UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude284EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude284SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude284EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude284EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX284(")) && !bytes.Contains(entrySource, claude284RequiredLogoAnchor()) {
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

func disableClaude284EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude284EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude284RequiredLogoAnchor() []byte {
	return []byte("function vQe(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_284(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function vQe(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=fqr(),t=Ro(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",f=Qe().agent;return{version:o,cwd:c,billingType:i,agentName:f}}`
	return replaceClaude208Function(data, string(claude284RequiredLogoAnchor()), "function _or(o,l,t){", replacement)
}

func patchUsageFetchFunction_2_1_284(data []byte) bool {
	const replacement = `async function xP(e,n){return Mr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=g6[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "async function xP(e,n){", "function y6(e){", replacement)
}

func patchModelPickerOptions_2_1_284(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX284(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts284(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function Kge(e=!1,n=null){return CDXOpts284(e)}`
	return replaceClaude208Function(data, "function Kge(e=!1,n=null){", "function d2(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_284(data []byte) bool {
	const replacement = `function Gsn(e,n){let r=CDX284(n),s=e.find((g)=>g.value===r||CDX284(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function Gsn(e,n){", "function BT(){", replacement)
}

func patchAgentModelValidator_2_1_284(data []byte) bool {
	return replaceFirstFixed(data, `model:z(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_284(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function So(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function So(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function L$(){`, `function Tne(){`, `function L$(){return"Codex"}`),
		replaceFirstFixed(data, `function Tne(){return"opus"+(zB()?"[1m]":"")}`, `function Tne(){return"opus"}`),
		replaceFirstFixed(data, `function W1r(e,n){if(!So())return!1;return!!e&&(Lt()||tC()||n)}`, `function W1r(e,n){return So()&&!!e}`),
		replaceFirstFixed(data, `function Yfn(e){if(!So())return!1;if(!tC(e))return!1;if(!c_(e))return!1;return au(Qe())}`, `function Yfn(e){return So()&&(he("flagSettings")?.fastMode===!0||au(Qe()))}`),
		replaceFirstFixed(data, `function au(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(he("policySettings")?.fastModePerSessionOptIn===!0)return!1;return he("flagSettings")?.fastMode===!0}`, `function au(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function c_(e){if(!So())return!1;let n=e??Dy(),r=xt(n),s=Ch(Ue(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function c_(e){return So()}`),
		replaceFirstFixed(data, `function vA(e,n){if(Lt()){if(e===null)return!!n;return!!n&&c_(e)}if(!c_(e))return!1;return!!n||Yfn(e)}`, `function vA(e,n){return So()&&(n!==void 0?!!n:au(Qe()))}`),
		replaceFirstFixed(data, `h={model:r.model,...So()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...De.gates.fastModeEnabled&&{fastMode:u.options.fastMode}`, `fastMode:u.options.fastMode`),
		replaceFirstFixed(data, `...So()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...So()&&{fastMode:Yfn(tt??null)}`, `fastMode:Yfn(tt??null)`),
		replaceFirstFixed(data, `...So()?{fastMode:nS}:!1`, `fastMode:nS`),
		replaceFirstFixed(data, `if(So()&&w(()=>tC())&&!gRe()&&w(()=>c_(Je))&&!!Un.fastMode)Kk="fast";`, `if(Un.fastMode)Kk="fast";`),
	}
	if bytes.Count(data, []byte(`...So()&&{fastMode:nS}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...So()&&{fastMode:nS}`, `fastMode:nS`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_284(data []byte) bool {
	return replaceFirstFixed(data, "function Jye(e){return`${um(e.inputTokens)}/${um(e.outputTokens)} per Mtok`}", `function Jye(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_284(data []byte) bool {
	return replaceClaude208Function(data, `function nd(e,o,n){let{source:r,window:s}=zC`, "var rd=", `function nd(e,o,n){return null}`)
}

func claude284RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function xH(){return}function Hte(){return}", "function t(e){", `function xH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Hte(){return}function IE(){return xH()||pn()?.accessToken}async function zv(e){return IE()}function AI(){return Hte()??ln().BASE_API_URL}function YAe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function Gk(){if(u())return!0;if(m$())return!1;return!vI()&&O4e()}`, `function Gk(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function sOr(){if(u())return!0;return!m$()&&!vI()&&uB()}`, `function sOr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function iOr(){`, `var B=`, `async function iOr(){return!m$()&&!vI()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function zGn(){if(m$())return"managed_disabled";if(vI())return"cloud_session";return await iOr()?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Act(){", "async function w(){", `async function Act(){if(m$())return i("managed_disabled","Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(vI())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(Gk())return!0;try{return uB()&&!vI()&&!m$()&&cc().source==="none"&&Du({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!WRr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!Gk()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude284SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_284(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_283},
		{"usage", patchUsageFetchFunction_2_1_284},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_284(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_284},
		{"agent-model-validator", patchAgentModelValidator_2_1_284},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_284},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_284},
		{"context-warning", patchContextWarningHint_2_1_284},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude284RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude284UIBrandingReplacements)
		}},
	}
}

func claude284Transformations(version string) []claude258Transformation {
	transformations := claude284SourceTransformationsForConfig(version, "2.1.284", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude284EmbeddedPatchedModuleBytecode})
}
