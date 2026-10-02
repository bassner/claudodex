package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude287SHA = "6eab8333fe2121553100d8f40bfada384a3e989b94f947e18ba6677a6fcb41ea"

var claudeUIPatch_2_1_287 = claudeUIPatchSpec{
	Version: "2.1.287",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude287SHA,
	Apply:   applyClaudeUIPatches_2_1_287,
}

var claude287UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_287(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude287UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude287EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude287SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude287EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude287EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX287(")) && !bytes.Contains(entrySource, claude287RequiredLogoAnchor()) {
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

func disableClaude287EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude287EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude287RequiredLogoAnchor() []byte {
	return []byte("function $nt(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_287(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function $nt(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=KQr(),t=jo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",m=tt().agent;return{version:o,cwd:c,billingType:l,agentName:m}}`
	return replaceClaude208Function(data, string(claude287RequiredLogoAnchor()), "function hpr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_287(data []byte) bool {
	const replacement = `async function IF(e,n){return Br(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Cie[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "async function IF(e,n){", "function Ug(e){", replacement)
}

func patchModelPickerOptions_2_1_287(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX287(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts287(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function s_e(e=!1,n=null){return CDXOpts287(e)}`
	return replaceClaude208Function(data, "function s_e(e=!1,n=null){", "function s8(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_287(data []byte) bool {
	const replacement = `function sdn(e,n){let r=CDX287(n),s=e.find((h)=>h.value===r||CDX287(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function sdn(e,n){", "function gM(){", replacement)
}

func patchAgentModelValidator_2_1_287(data []byte) bool {
	return replaceFirstFixed(data, `model:B(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_287(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function To(){if(He()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function To(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function pU(){`, `function Roe(){`, `function pU(){return"Codex"}`),
		replaceFirstFixed(data, `function Roe(){return"opus"+(bj()?"[1m]":"")}`, `function Roe(){return"opus"}`),
		replaceFirstFixed(data, `function _qr(e,n,r){if(!To())return!1;return kbe(e,r)&&(Ft()||tT()||n)}`, `function _qr(e,n,r){return To()&&!!e}`),
		replaceFirstFixed(data, `function Cbn(e){if(!To())return!1;if(!tT(e))return!1;if(!Ry(e))return!1;return Abn(tt())}`, `function Cbn(e){return To()&&(ge("flagSettings")?.fastMode===!0||Abn(tt()))}`),
		replaceFirstFixed(data, `function Abn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function Abn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Ry(e){if(!To())return!1;let n=e??Py(),r=Rt(n),s=ny(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Ry(e){return To()}`),
		replaceFirstFixed(data, `function Lk(e,n){if(Ft()){if(e===null)return!!n;return!!n&&Ry(e)}if(!Ry(e))return!1;return!!n||Cbn(e)}`, `function Lk(e,n){return To()&&(n!==void 0?!!n:Abn(tt()))}`),
		replaceFirstFixed(data, `h={model:r.model,...To()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...bt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...To()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...To()&&{fastMode:Cbn(at??null)}`, `fastMode:Cbn(at??null)`),
		replaceFirstFixed(data, `...To()?{fastMode:HP}:!1`, `fastMode:HP`),
		replaceFirstFixed(data, `if(To()&&he(()=>tT())&&!VPe()&&he(()=>Ry($t))&&!!Gn.fastMode&&!sWt(Gn.model))np="fast";`, `if(Gn.fastMode)np="fast";`),
	}
	if bytes.Count(data, []byte(`...To()&&{fastMode:HP}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...To()&&{fastMode:HP}`, `fastMode:HP`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_287(data []byte) bool {
	return replaceFirstFixed(data, "function Pbe(e){return`${s_(e.inputTokens)}/${s_(e.outputTokens)} per Mtok`}", `function Pbe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_287(data []byte) bool {
	return replaceClaude208Function(data, `function gc(e,o,n){let{source:r,window:s}=wE`, "var hc=", `function gc(e,o,n){return null}`)
}

func claude287RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function TH(){return}function Dre(){return}", "function t(e){", `function TH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Dre(){return}function OE(){return TH()||fn()?.accessToken}async function Iv(e){return OE()}function I0(){return Dre()??dn().BASE_API_URL}function cxe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function WA(){if(u())return!0;if(N1())return!1;return!R0()&&lYe()}`, `function WA(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function aBr(){if(u())return!0;return!N1()&&!R0()&&j2()}`, `function aBr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function lBr(){`, `var T=`, `async function lBr(){return!R0()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function y5n(){if(R0())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Kft(){", "async function N(){", `async function Kft(){if(R0())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(WA())return!0;try{return j2()&&!R0()&&!N1()&&Rc().source==="none"&&Gp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!z$r.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!WA()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude287SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_287(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_286},
		{"usage", patchUsageFetchFunction_2_1_287},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_287(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_287},
		{"agent-model-validator", patchAgentModelValidator_2_1_287},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_287},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_287},
		{"context-warning", patchContextWarningHint_2_1_287},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude287RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude287UIBrandingReplacements)
		}},
	}
}

func claude287Transformations(version string) []claude258Transformation {
	transformations := claude287SourceTransformationsForConfig(version, "2.1.287", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude287EmbeddedPatchedModuleBytecode})
}
