package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude289SHA = "03d66745e3bb69ec727d66023696f3820bc0a00a8a5ba725eb6706d0c67cbe69"

var claudeUIPatch_2_1_289 = claudeUIPatchSpec{
	Version: "2.1.289",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude289SHA,
	Apply:   applyClaudeUIPatches_2_1_289,
}

var claude289UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_289(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude289UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude289EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude289SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude289EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude289EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX289(")) && !bytes.Contains(entrySource, claude289RequiredLogoAnchor()) {
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

func disableClaude289EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude289EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude289RequiredLogoAnchor() []byte {
	return []byte("function xot(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_289(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function xot(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=Oro(),t=Uo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",m=lt().agent;return{version:o,cwd:c,billingType:l,agentName:m}}`
	return replaceClaude208Function(data, string(claude289RequiredLogoAnchor()), "function Hgr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_289(data []byte) bool {
	const anchor = `var Tle={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function dH(e,n){`
	const startMarker = "async function dH(e,n){"
	const endMarker = "function Zg(e){"
	const replacement = `async function dH(e,n){return Gr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Tle[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
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

func patchModelPickerOptions_2_1_289(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX289(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts289(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function wSe(e=!1,n=null){return CDXOpts289(e)}`
	return replaceClaude208Function(data, "function wSe(e=!1,n=null){", "function UJ(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_289(data []byte) bool {
	const replacement = `function bfn(e,n){if(n==null)return void 0;let r=CDX289(n),s=e.find((h)=>h.value===r||CDX289(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function bfn(e,n){", "function II(){", replacement)
}

func patchAgentModelValidator_2_1_289(data []byte) bool {
	return replaceFirstFixed(data, `model:j(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_289(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Io(){if(He()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Io(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function GU(){`, `function Cse(){`, `function GU(){return"Codex"}`),
		replaceFirstFixed(data, `function Cse(){return"opus"+(eW()?"[1m]":"")}`, `function Cse(){return"opus"}`),
		replaceFirstFixed(data, `function Q5r(e,n,r){if(!Io())return!1;return Bwe(e,r)&&(Ut()||gk()||n)}`, `function Q5r(e,n,r){return Io()&&!!e}`),
		replaceFirstFixed(data, `function Hvn(e){if(!Io())return!1;if(!gk(e))return!1;if(!Fy(e))return!1;return Mvn(lt())}`, `function Hvn(e){return Io()&&(ge("flagSettings")?.fastMode===!0||Mvn(lt()))}`),
		replaceFirstFixed(data, `function Mvn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ge("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ge("flagSettings")?.fastMode===!0}`, `function Mvn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Fy(e){if(!Io())return!1;let n=e??Uy(),r=At(n),s=fy(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Fy(e){return Io()}`),
		replaceFirstFixed(data, `function XT(e,n){if(Ut()){if(e===null)return!!n;return!!n&&Fy(e)}if(!Fy(e))return!1;return!!n||Hvn(e)}`, `function XT(e,n){return Io()&&(n!==void 0?!!n:Mvn(lt()))}`),
		replaceFirstFixed(data, `h={model:r.model,...Io()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...wt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...Io()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Io()&&{fastMode:Hvn(dt??null)}`, `fastMode:Hvn(dt??null)`),
		replaceFirstFixed(data, `...Io()?{fastMode:IM}:!1`, `fastMode:IM`),
		replaceFirstFixed(data, `if(Io()&&ye(()=>gk())&&!_0e()&&ye(()=>Fy(Pt))&&!!Wn.fastMode&&!vGt(Wn.model))Uh="fast";`, `if(Wn.fastMode)Uh="fast";`),
	}
	if bytes.Count(data, []byte(`...Io()&&{fastMode:IM}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Io()&&{fastMode:IM}`, `fastMode:IM`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_289(data []byte) bool {
	return replaceFirstFixed(data, "function Gwe(e){return`${f_(e.inputTokens)}/${f_(e.outputTokens)} per Mtok`}", `function Gwe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_289(data []byte) bool {
	return replaceClaude208Function(data, `function gc(e,o,n){let{source:r,window:s}=Fw`, "var hc=", `function gc(e,o,n){return null}`)
}

func claude289RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function VH(){return}function Poe(){return}", "function t(e){", `function VH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Poe(){return}function qE(){return VH()||mn()?.accessToken}async function Wv(e){return qE()}function W0(){return Poe()??pn().BASE_API_URL}function xPe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function ok(){if(u())return!0;if(fU())return!1;return!B0()&&t7e()}`, `function ok(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function k6r(){if(u())return!0;return!fU()&&!B0()&&S2()}`, `function k6r(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function T6r(){`, `var T=`, `async function T6r(){return!B0()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function fXn(){if(B0())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function tht(){", "async function N(){", `async function tht(){if(B0())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(ok())return!0;try{return S2()&&!B0()&&!fU()&&Dc().source==="none"&&Qp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!a2r.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!ok()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude289SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_289(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_286},
		{"usage", patchUsageFetchFunction_2_1_289},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_289(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_289},
		{"agent-model-validator", patchAgentModelValidator_2_1_289},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_289},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_289},
		{"context-warning", patchContextWarningHint_2_1_289},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude289RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude289UIBrandingReplacements)
		}},
	}
}

func claude289Transformations(version string) []claude258Transformation {
	transformations := claude289SourceTransformationsForConfig(version, "2.1.289", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude289EmbeddedPatchedModuleBytecode})
}
