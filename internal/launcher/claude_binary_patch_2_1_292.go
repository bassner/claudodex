package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude292SHA = "97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f"

var claudeUIPatch_2_1_292 = claudeUIPatchSpec{
	Version: "2.1.292",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude292SHA,
	Apply:   applyClaudeUIPatches_2_1_292,
}

var claude292UIBrandingReplacements = claude291UIBrandingReplacements

func applyClaudeUIPatches_2_1_292(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude292UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude292EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude292SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude292EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude292EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX292(")) && !bytes.Contains(entrySource, claude292RequiredLogoAnchor()) {
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

func disableClaude292EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude292EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude292RequiredLogoAnchor() []byte {
	return []byte("function Sct(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_292(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Sct(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=Eyo(),t=qo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",d=dt().agent;return{version:o,cwd:c,billingType:l,agentName:d}}`
	return replaceClaude208Function(data, string(claude292RequiredLogoAnchor()), "function xAr(o,i,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_292(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function k(i){", replacement)
}

func patchUsageFetchFunction_2_1_292(data []byte) bool {
	const anchor = `var Fse={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function MN(e,n){`
	const startMarker = "async function MN(e,n){"
	const endMarker = "function gg(e){"
	const replacement = `async function MN(e,n){return Qr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Fse[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	anchorStart := bytes.Index(data, []byte(anchor))
	start := anchorStart + strings.Index(anchor, startMarker)
	endRel := bytes.Index(data[start:], []byte(endMarker))
	if endRel < 0 {
		return false
	}
	old := data[start : start+endRel]
	newBytes, ok := fitReplacement(old, replacement)
	if !ok {
		return false
	}
	copy(old, newBytes)
	return true
}

func patchModelPickerOptions_2_1_292(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX292(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts292(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function JEe(e=!1,n=null){return CDXOpts292(e)}`
	return replaceClaude208Function(data, "function JEe(e=!1,n=null){", "function K3(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_292(data []byte) bool {
	const replacement = `function NEn(e,n){if(n==null)return void 0;let r=CDX292(n),s=e.find((h)=>h.value===r||CDX292(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function NEn(e,n){", "function TO(){", replacement)
}

func patchAgentModelValidator_2_1_292(data []byte) bool {
	return replaceFirstFixed(data, `model:W(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_292(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Uo(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Uo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function zj(){`, `function wle(){`, `function zj(){return"Codex"}`),
		replaceFirstFixed(data, `function wle(){return"opus"+(e6()?"[1m]":"")}`, `function wle(){return"opus"}`),
		replaceFirstFixed(data, `function Dso(e,n,r){if(!Uo())return!1;return _Ce(e,r)&&(Ht()||gT()||n)}`, `function Dso(e,n,r){return Uo()&&!!e}`),
		replaceFirstFixed(data, `function m9t(e){if(!Uo())return!1;if(!gT(e))return!1;if(!g_(e))return!1;return P0n(dt())}`, `function m9t(e){return Uo()&&(me("flagSettings")?.fastMode===!0||P0n(dt()))}`),
		replaceFirstFixed(data, `function P0n(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(me("policySettings")?.fastModePerSessionOptIn===!0)return!1;return me("flagSettings")?.fastMode===!0}`, `function P0n(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function g_(e){if(!Uo())return!1;let n=e??y_(),r=Pt(n),s=qy(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function g_(e){return Uo()}`),
		replaceFirstFixed(data, `function QR(e,n){if(Ht()){if(e===null)return!!n;return!!n&&g_(e)}if(!g_(e))return!1;return!!n||m9t(e)}`, `function QR(e,n){return Uo()&&(n!==void 0?!!n:P0n(dt()))}`),
		replaceFirstFixed(data, `h={model:r.model,...Uo()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...wt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...Uo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Uo()&&{fastMode:m9t(j??null)}`, `fastMode:m9t(j??null)`),
		replaceFirstFixed(data, `...Uo()?{fastMode:Hm}:!1`, `fastMode:Hm`),
		replaceFirstFixed(data, `if(Uo()&&_e(()=>gT())&&!yDe()&&_e(()=>g_(Nt))&&!!Wn.fastMode&&!h9t(Wn.model))zE="fast";`, `if(Wn.fastMode)zE="fast";`),
	}
	if bytes.Count(data, []byte(`...Uo()&&{fastMode:Hm}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Uo()&&{fastMode:Hm}`, `fastMode:Hm`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_292(data []byte) bool {
	return replaceFirstFixed(data, "function wCe(e){return`${C_(e.inputTokens)}/${C_(e.outputTokens)} per Mtok`}", `function wCe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_292(data []byte) bool {
	return replaceClaude208Function(data, `function Yc(e,o,n){let{source:r,window:s}=wE`, "var zc=", `function Yc(e,o,n){return null}`)
}

func claude292SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_292(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_292},
		{"usage", patchUsageFetchFunction_2_1_292},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_292(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_292},
		{"agent-model-validator", patchAgentModelValidator_2_1_292},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_292},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_292},
		{"context-warning", patchContextWarningHint_2_1_292},
		{"resume-hints", patchResumeCommandHints_2_1_291},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude292RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude292UIBrandingReplacements)
		}},
	}
}

func claude292RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function _D(){return}function Sae(){return}", "function t(e){", `function _D(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Sae(){return}function Fv(){return _D()||mn()?.accessToken}async function Ok(e){return Fv()}function hH(){return Sae()??pn().BASE_API_URL}function wHe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function nT(){if(u())return!0;if(mj())return!1;return!gH()&&gtt()}`, `function nT(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function RQr(){if(u())return!0;return!mj()&&!gH()&&AG()}`, `function RQr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function xQr(){`, `var I=`, `async function xQr(){return!gH()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function Lar(){if(gH())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function bEt(){", "async function N(){", `async function bEt(){if(gH())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(nT())return!0;try{return AG()&&!gH()&&!mj()&&mc().source==="none"&&Tf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!FYr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!nT()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude292Transformations(version string) []claude258Transformation {
	transformations := claude292SourceTransformationsForConfig(version, "2.1.292", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude292EmbeddedPatchedModuleBytecode})
}
