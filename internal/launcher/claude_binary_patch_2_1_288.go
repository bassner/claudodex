package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude288SHA = "bbe93063f7a0879a1021b2891e5c9354e5b3b98433e32efe6750f7710afed750"

var claudeUIPatch_2_1_288 = claudeUIPatchSpec{
	Version: "2.1.288",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude288SHA,
	Apply:   applyClaudeUIPatches_2_1_288,
}

var claude288UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_288(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude288UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude288EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude288SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude288EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude288EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX288(")) && !bytes.Contains(entrySource, claude288RequiredLogoAnchor()) {
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

func disableClaude288EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude288EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude288RequiredLogoAnchor() []byte {
	return []byte("function sot(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_288(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function sot(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=xno(),t=Vo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",m=st().agent;return{version:o,cwd:c,billingType:l,agentName:m}}`
	return replaceClaude208Function(data, string(claude288RequiredLogoAnchor()), "function Dmr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_288(data []byte) bool {
	const replacement = `async function CF(e,n){return Gr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Hie[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "async function CF(e,n){", "function Ug(e){", replacement)
}

func patchModelPickerOptions_2_1_288(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX288(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts288(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function uSe(e=!1,n=null){return CDXOpts288(e)}`
	return replaceClaude208Function(data, "function uSe(e=!1,n=null){", "function X8(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_288(data []byte) bool {
	const replacement = `function qpn(e,n){if(n==null)return void 0;let r=CDX288(n),s=e.find((h)=>h.value===r||CDX288(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function qpn(e,n){", "function nI(){", replacement)
}

func patchAgentModelValidator_2_1_288(data []byte) bool {
	return replaceFirstFixed(data, `model:j(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_288(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Io(){if(He()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Io(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function $U(){`, `function Sse(){`, `function $U(){return"Codex"}`),
		replaceFirstFixed(data, `function Sse(){return"opus"+(Y2()?"[1m]":"")}`, `function Sse(){return"opus"}`),
		replaceFirstFixed(data, `function Q4r(e,n,r){if(!Io())return!1;return Owe(e,r)&&(Ut()||gT()||n)}`, `function Q4r(e,n,r){return Io()&&!!e}`),
		replaceFirstFixed(data, `function zEn(e){if(!Io())return!1;if(!gT(e))return!1;if(!Ly(e))return!1;return VEn(st())}`, `function zEn(e){return Io()&&(ge("flagSettings")?.fastMode===!0||VEn(st()))}`),
		replaceFirstFixed(data, `function VEn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function VEn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Ly(e){if(!Io())return!1;let n=e??Fy(),r=Rt(n),s=uy(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Ly(e){return Io()}`),
		replaceFirstFixed(data, `function Yk(e,n){if(Ut()){if(e===null)return!!n;return!!n&&Ly(e)}if(!Ly(e))return!1;return!!n||zEn(e)}`, `function Yk(e,n){return Io()&&(n!==void 0?!!n:VEn(st()))}`),
		replaceFirstFixed(data, `h={model:r.model,...Io()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...wt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...Io()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Io()&&{fastMode:zEn(dt??null)}`, `fastMode:zEn(dt??null)`),
		replaceFirstFixed(data, `...Io()?{fastMode:LM}:!1`, `fastMode:LM`),
		replaceFirstFixed(data, `if(Io()&&ye(()=>gT())&&!a0e()&&ye(()=>Ly(Pt))&&!!qn.fastMode&&!Y6t(qn.model))Fh="fast";`, `if(qn.fastMode)Fh="fast";`),
	}
	if bytes.Count(data, []byte(`...Io()&&{fastMode:LM}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Io()&&{fastMode:LM}`, `fastMode:LM`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_288(data []byte) bool {
	return replaceFirstFixed(data, "function Dwe(e){return`${f_(e.inputTokens)}/${f_(e.outputTokens)} per Mtok`}", `function Dwe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_288(data []byte) bool {
	return replaceClaude208Function(data, `function _c(e,o,n){let{source:r,window:s}=Lw`, "var yc=", `function _c(e,o,n){return null}`)
}

func claude288RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function WH(){return}function Coe(){return}", "function t(e){", `function WH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Coe(){return}function zE(){return WH()||mn()?.accessToken}async function jv(e){return zE()}function U0(){return Coe()??pn().BASE_API_URL}function SPe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function oT(){if(u())return!0;if(lU())return!1;return!F0()&&MXe()}`, `function oT(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function kWr(){if(u())return!0;return!lU()&&!F0()&&f2()}`, `function kWr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function RWr(){`, `var I=`, `async function RWr(){return!F0()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function gYn(){if(F0())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Igt(){", "async function N(){", `async function Igt(){if(F0())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(oT())return!0;try{return f2()&&!F0()&&!lU()&&Hc().source==="none"&&Jp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!ljr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!oT()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude288SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_288(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_286},
		{"usage", patchUsageFetchFunction_2_1_288},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_288(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_288},
		{"agent-model-validator", patchAgentModelValidator_2_1_288},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_288},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_288},
		{"context-warning", patchContextWarningHint_2_1_288},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude288RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude288UIBrandingReplacements)
		}},
	}
}

func claude288Transformations(version string) []claude258Transformation {
	transformations := claude288SourceTransformationsForConfig(version, "2.1.288", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude288EmbeddedPatchedModuleBytecode})
}
