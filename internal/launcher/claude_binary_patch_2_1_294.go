package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude294SHA = "def0d15e64dd7d89621f88d28214f885b1c38b0ddd69762fb8593e34915d6d53"

var claudeUIPatch_2_1_294 = claudeUIPatchSpec{
	Version: "2.1.294",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude294SHA,
	Apply:   applyClaudeUIPatches_2_1_294,
}

var claude294UIBrandingReplacements = claude292UIBrandingReplacements

func applyClaudeUIPatches_2_1_294(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude294UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude294EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude294SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude294EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude294EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX294(")) && !bytes.Contains(entrySource, claude294RequiredLogoAnchor()) {
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

func disableClaude294EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude294EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude294RequiredLogoAnchor() []byte {
	return []byte("function Pdt(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_294(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Pdt(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=Pbo(),t=Yo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",d=ut().agent;return{version:o,cwd:c,billingType:l,agentName:d}}`
	return replaceClaude208Function(data, string(claude294RequiredLogoAnchor()), "function rxr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_294(data []byte) bool {
	const anchor = `var sle={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function zF(e,n){`
	const startMarker = "async function zF(e,n){"
	const endMarker = "function Kg(e){"
	const replacement = `async function zF(e,n){return to(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=sle[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
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

func patchModelPickerOptions_2_1_294(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX294(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts294(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function Ave(e=!1,n=null){return CDXOpts294(e)}`
	return replaceClaude208Function(data, "function Ave(e=!1,n=null){", "function GJ(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_294(data []byte) bool {
	const replacement = `function wkn(e,n){if(n==null)return void 0;let r=CDX294(n),s=e.find((h)=>h.value===r||CDX294(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function wkn(e,n){", "function FM(){", replacement)
}

func patchAgentModelValidator_2_1_294(data []byte) bool {
	return replaceFirstFixed(data, `model:W(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_294(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function jo(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function jo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function r2(){`, `function zle(){`, `function r2(){return"Codex"}`),
		replaceFirstFixed(data, `function zle(){return"opus"+(m6()?"[1m]":"")}`, `function zle(){return"opus"}`),
		replaceFirstFixed(data, `function Flo(e,n,r){if(!jo())return!1;return JCe(e,r)&&(Mt()||kT()||n)}`, `function Flo(e,n,r){return jo()&&!!e}`),
		replaceFirstFixed(data, `function F8t(e){if(!jo())return!1;if(!kT(e))return!1;if(!v_(e))return!1;return vMn(ut())}`, `function F8t(e){return jo()&&(me("flagSettings")?.fastMode===!0||vMn(ut()))}`),
		replaceFirstFixed(data, `function vMn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(me("policySettings")?.fastModePerSessionOptIn===!0)return!1;return me("flagSettings")?.fastMode===!0}`, `function vMn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function v_(e){if(!jo())return!1;let n=e??k_(),r=Pt(n),s=Zy(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function v_(e){return jo()}`),
		replaceFirstFixed(data, `function tx(e,n){if(Mt()){if(e===null)return!!n;return!!n&&v_(e)}if(!v_(e))return!1;return!!n||F8t(e)}`, `function tx(e,n){return jo()&&(n!==void 0?!!n:vMn(ut()))}`),
		replaceFirstFixed(data, `h={model:r.model,...jo()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...xt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...jo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...jo()&&{fastMode:F8t(j??null)}`, `fastMode:F8t(j??null)`),
		replaceFirstFixed(data, `...jo()?{fastMode:Bu}:!1`, `fastMode:Bu`),
		replaceFirstFixed(data, `if(jo()&&_e(()=>kT())&&!tLe()&&_e(()=>v_(Ht))&&!!Gn.fastMode&&!U8t(Gn.model))cR="fast";`, `if(Gn.fastMode)cR="fast";`),
	}
	if bytes.Count(data, []byte(`...jo()&&{fastMode:Bu}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...jo()&&{fastMode:Bu}`, `fastMode:Bu`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_294(data []byte) bool {
	return replaceClaude208Function(data, "function eAe(e){", "function fps(e){", `function eAe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_294(data []byte) bool {
	return replaceClaude208Function(data, `function zc(e,o,n){let{source:r,window:s}=HE`, "var Vc=", `function zc(e,o,n){return null}`)
}

func claude294SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_294(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_292},
		{"usage", patchUsageFetchFunction_2_1_294},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_294(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_294},
		{"agent-model-validator", patchAgentModelValidator_2_1_294},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_294},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_294},
		{"context-warning", patchContextWarningHint_2_1_294},
		{"resume-hints", patchResumeCommandHints_2_1_291},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude294RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude294UIBrandingReplacements)
		}},
	}
}

func claude294RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function DD(){return}function Wae(){return}", "function t(e){", `function DD(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Wae(){return}function Kv(){return DD()||mn()?.accessToken}async function Uk(e){return Kv()}function xH(){return Wae()??cn().BASE_API_URL}function aMe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function pT(){if(u())return!0;if(Aj())return!1;return!RH()&&vnt()}`, `function pT(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function Tto(){if(u())return!0;return!Aj()&&!RH()&&BG()}`, `function Tto(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function Rto(){`, `var I=`, `async function Rto(){return!RH()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function odr(){if(RH())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Lvt(){", "async function N(){", `async function Lvt(){if(RH())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(pT())return!0;try{return BG()&&!RH()&&!Aj()&&Ul().source==="none"&&Rf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!AJr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!pT()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude294Transformations(version string) []claude258Transformation {
	transformations := claude294SourceTransformationsForConfig(version, "2.1.294", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude294EmbeddedPatchedModuleBytecode})
}
