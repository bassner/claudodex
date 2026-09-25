package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude282SHA = "fcfd837103965c64de34a6b9b94370d77a347ea71819715a27d5f0ef01775ea4"

var claudeUIPatch_2_1_282 = claudeUIPatchSpec{
	Version: "2.1.282",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude282SHA,
	Apply:   applyClaudeUIPatches_2_1_282,
}

// Claude Code 2.1.282 retained the Bun module table and several fixed targets,
// but changed minified symbols in the source transformations below. Keep a
// version-specific gate so a republished binary cannot reuse these byte regions.
var claude282UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_282(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude282UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude282EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude282SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude282EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude282EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX282(")) && !bytes.Contains(entrySource, []byte("function Rj(e=!1){")) {
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

func disableClaude282EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude282EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude282RequiredLogoAnchor() []byte {
	return []byte("function OYe(){let o=a.DEMO_VERSION??")
}

func claude282RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function RN(){return}function hZ(){return}", "function t(e){", `function RN(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function hZ(){return}function jw(){return RN()||pn()?.accessToken}async function cv(e){return jw()}function RP(){return hZ()??dn().BASE_API_URL}function $ve(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function uC(){if(u())return!0;if(AN())return!1;return!kP()&&UVe()}`, `function uC(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function Bvr(){if(u())return!0;return!AN()&&!kP()&&tj()}`, `function Bvr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function jvr(){`, `var O=`, `async function jvr(){return!AN()&&!kP()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function LFn(){if(AN())return"managed_disabled";if(kP())return"cloud_session";return await jvr()?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function MFn(){", "async function v(){", `async function MFn(){if(AN())return i("managed_disabled","Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(kP())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(uC())return!0;try{return tj()&&!kP()&&!AN()&&jc().source==="none"&&Zf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!P_r.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!uC()}`, `get isHidden(){return!1}`)
		}},
	}
}

func patchLogoDisplayDataFunction_2_1_282(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function OYe(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=H$r(),t=a.DEMO_VERSION?"/code/claude":So(ne()),c=a.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",m=Ye().agent;return{version:o,cwd:c,billingType:i,agentName:m}}`
	return replaceClaude208Function(data, string(claude282RequiredLogoAnchor()), "function _Xn(o,l,t){", replacement)
}

func patchUsageFetchFunction_2_1_282(data []byte) bool {
	const replacement = `function tC(e,n){return Ir(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=YV[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function tC(e,n){", `var qV=`, replacement)
}

func patchModelPickerOptions_2_1_282(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX282(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts282(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function Rj(e=!1){return CDXOpts282(e)}`
	return replaceClaude208Function(data, "function Rj(e=!1){", "function zr(e){", replacement)
}

func patchModelPickerResolver_2_1_282(data []byte) bool {
	return replaceClaude208Function(data, "function Ij(e,n){", "function jSr(e){", `function Ij(e,n){return CDXOpts282(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_282(data []byte) bool {
	return replaceClaude208Function(data, "function tG(e=!1){", "var Aj=", `function tG(e=!1){return CDXOpts282(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_282(data []byte) bool {
	return replaceClaude208Function(data, "function Jpe(e=!1,n=null){", "function Cj(e,n){", `function Jpe(e=!1,n=null){return CDXOpts282(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_282(data []byte) bool {
	const replacement = `function JQt(e,n){let r=CDX282(n),s=e.find((g)=>g.value===r||CDX282(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function JQt(e,n){if(e.some((g)=>g.value===n))return n;", "function rA(){", replacement)
}

func patchAgentModelValidator_2_1_282(data []byte) bool {
	return replaceFirstFixed(data, `model:G(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_282(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function oo(){if(Le()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function oo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function KN(){`, `function see(){`, `function KN(){return"Codex"}`),
		replaceFirstFixed(data, `function see(){return"opus"+(BA()?"[1m]":"")}`, `function see(){return"opus"}`),
		replaceFirstFixed(data, `function kPr(e,n){if(!oo())return!1;return!!e&&(Mt()||bC()||n)}`, `function kPr(e,n){return oo()&&!!e}`),
		replaceFirstFixed(data, `function Rin(e){if(!oo())return!1;if(!bC(e))return!1;if(!Iy(e))return!1;return Ll(Ye())}`, `function Rin(e){return oo()&&(ye("flagSettings")?.fastMode===!0||Ll(Ye()))}`),
		replaceFirstFixed(data, `function Ll(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function Ll(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Iy(e){if(!oo())return!1;let n=e??s_(),r=At(n),s=Ch(Ue(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Iy(e){return oo()}`),
		replaceFirstFixed(data, `function $A(e,n){if(Mt()){if(e===null)return!!n;return!!n&&Iy(e)}if(!Iy(e))return!1;return!!n||Rin(e)}`, `function $A(e,n){return oo()&&(n!==void 0?!!n:Ll(Ye()))}`),
		replaceFirstFixed(data, `g={model:r.model,...oo()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Xe.gates.fastModeEnabled&&{fastMode:h.options.fastMode}`, `fastMode:h.options.fastMode`),
		replaceFirstFixed(data, `...oo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...oo()&&{fastMode:Rin(tt??null)}`, `fastMode:Rin(tt??null)`),
		replaceFirstFixed(data, `...oo()?{fastMode:VC}:!1`, `fastMode:VC`),
		replaceFirstFixed(data, `if(oo()&&S(()=>bC())&&!QCe()&&S(()=>Iy(We))&&!!zn.fastMode)qS="fast";`, `if(zn.fastMode)qS="fast";`),
	}
	if bytes.Count(data, []byte(`...oo()&&{fastMode:VC}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...oo()&&{fastMode:VC}`, `fastMode:VC`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_282(data []byte) bool {
	return replaceFirstFixed(data, "function Kme(e){return`${Eg(e.inputTokens)}/${Eg(e.outputTokens)} per Mtok`}", `function Kme(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_282(data []byte) bool {
	return replaceClaude208Function(data, `function pm(e,n,r){let{source:s,window:g}=cA`, "var fm=", `function pm(e,n,r){return null}`)
}

func claude282Transformations(version string) []claude258Transformation {
	transformations := claude282SourceTransformationsForConfig(version, "2.1.282", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude282EmbeddedPatchedModuleBytecode})
}

func claude282SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_282(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_281},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_281},
		{"usage", patchUsageFetchFunction_2_1_282},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_282(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_282},
		{"model-list", patchModelListOptions_2_1_282},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_282},
		{"model-selection", patchModelPickerSelectionValue_2_1_282},
		{"agent-model-validator", patchAgentModelValidator_2_1_282},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_282},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_282},
		{"context-warning", patchContextWarningHint_2_1_282},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_281},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude282RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude282UIBrandingReplacements)
		}},
	}
}

func claude282ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX282("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function zr("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
