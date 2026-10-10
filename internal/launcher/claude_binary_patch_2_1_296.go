package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude296SHA = "c9b5341637becbd423ddffc5b254afb645682a3868cb708bbc6cc0e7bb419937"

var claudeUIPatch_2_1_296 = claudeUIPatchSpec{
	Version: "2.1.296",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude296SHA,
	Apply:   applyClaudeUIPatches_2_1_296,
}

var claude296UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude295UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_296(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude296UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude296EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude296SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude296EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude296EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX296(")) && !bytes.Contains(entrySource, claude296RequiredLogoAnchor()) {
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

func disableClaude296EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude296EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude296RequiredLogoAnchor() []byte {
	return []byte("function wmt(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_296(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function wmt(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=$Ro(),t=a.DEMO_VERSION?"/code/claude":jn()&&(zr("tengu_violin_wood",!1)||zr(reo,!1))?Rs(Ee()):os(se()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",d=ft().agent;return{version:o,cwd:c,billingType:l,agentName:d}}`
	return replaceClaude208Function(data, string(claude296RequiredLogoAnchor()), "function YMr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_296(data []byte) bool {
	const anchor = `var Ww={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function km(e,r){`
	const startMarker = "async function km(e,r){"
	const endMarker = "function Ns(e){"
	const replacement = `async function km(e,r){return Zr(r==="at_wall"?"api_usage_fetch_at_wall":r==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let n=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Ww[r],h=await fetch(n+s,{headers:{"Content-Type":"application/json"}});if(!h.ok)throw Error("Auth error: "+h.status);return await h.json()})}`
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

func patchModelPickerOptions_2_1_296(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX296(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts296(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function GTe(e=!1,r=null){return CDXOpts296(e)}`
	return replaceClaude208Function(data, "function GTe(e=!1,r=null){", "function mb(e,r){", replacement)
}

func patchModelPickerSelectionValue_2_1_296(data []byte) bool {
	const replacement = `function EFn(e,r){if(r==null)return void 0;let n=CDX296(r),s=e.find((h)=>h.value===n||CDX296(h.value)===n);return s?.value??n}`
	return replaceClaude208Function(data, "function EFn(e,r){", "function xc(){", replacement)
}

func patchAgentModelValidator_2_1_296(data []byte) bool {
	return replaceFirstFixed(data, `model:U(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_296(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Zo(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Zo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function mW(){`, `function eue(){`, `function mW(){return"Codex"}`),
		replaceFirstFixed(data, `function eue(){return"opus"+(G6()?"[1m]":"")}`, `function eue(){return"opus"}`),
		replaceFirstFixed(data, `function Nho(e,n,r){if(!Zo())return!1;return SRe(e,r)&&(Ht()||yR()||n)}`, `function Nho(e,n,r){return Zo()&&!!e}`),
		replaceFirstFixed(data, `function BQt(e){if(!Zo())return!1;if(!yR(e))return!1;if(!V_(e))return!1;return VUn(ft())}`, `function BQt(e){return Zo()&&(fe("flagSettings")?.fastMode===!0||VUn(ft()))}`),
		replaceFirstFixed(data, `function VUn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(fe("policySettings")?.fastModePerSessionOptIn===!0)return!1;return fe("flagSettings")?.fastMode===!0}`, `function VUn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function V_(e){if(!Zo())return!1;let n=e??q_(),r=Dt(n),s=E_(We(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function V_(e){return Zo()}`),
		replaceFirstFixed(data, `function lP(e,n){if(Ht()){if(e===null)return!!n;return!!n&&V_(e)}if(!V_(e))return!1;return!!n||BQt(e)}`, `function lP(e,n){return Zo()&&(n!==void 0?!!n:VUn(ft()))}`),
		replaceFirstFixed(data, `y={model:r.model,...Zo()&&{fastMode:r.fastMode}}`, `y={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...bt.gates.fastModeEnabled&&{fastMode:p.options.fastMode}`, `fastMode:p.options.fastMode`),
		replaceFirstFixed(data, `...Zo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Zo()&&{fastMode:BQt(re??null)}`, `fastMode:BQt(re??null)`),
		replaceFirstFixed(data, `...Zo()?{fastMode:xD}:!1`, `fastMode:xD`),
		replaceFirstFixed(data, `if(Zo()&&_e(()=>yR())&&!qFe()&&_e(()=>V_(Wt))&&!!Jn.fastMode&&!WQt(Jn.model))SC="fast";`, `if(Jn.fastMode)SC="fast";`),
	}
	if bytes.Count(data, []byte(`...Zo()&&{fastMode:xD}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Zo()&&{fastMode:xD}`, `fastMode:xD`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_296(data []byte) bool {
	return replaceClaude208Function(data, "function ERe(e){", "function cvs(e){", `function ERe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_296(data []byte) bool {
	return replaceClaude208Function(data, `function nu(e,o,n){let{source:r,window:s}=aE`, "var ru=", `function nu(e,o,n){return null}`)
}

func claude296SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_296(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_295},
		{"usage", patchUsageFetchFunction_2_1_296},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_296(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_296},
		{"agent-model-validator", patchAgentModelValidator_2_1_296},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_296},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_296},
		{"context-warning", patchContextWarningHint_2_1_296},
		{"resume-hints", patchResumeCommandHints_2_1_291},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude296RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude296UIBrandingReplacements)
		}},
	}
}

func claude296RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function KL(){return}function hde(){return}", "function t(e){", `function KL(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function hde(){return}function Fk(){return KL()||Sn()?.accessToken}async function HA(e){return Fk()}function XM(){return hde()??dn().BASE_API_URL}function GNe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function iR(){if(u())return!0;if(N2())return!1;return!WM()&&Ist()}`, `function iR(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function dlo(){if(u())return!0;return!N2()&&!WM()&&g6()}`, `function dlo(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function ulo(){`, `var T=`, `async function ulo(){return!WM()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function Fgr(){if(WM())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function ITt(){", "async function N(){", `async function ITt(){if(WM())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(iR())return!0;try{return g6()&&!WM()&&!N2()&&vl().source==="none"&&Wp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!soo.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!iR()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude296Transformations(version string) []claude258Transformation {
	transformations := claude296SourceTransformationsForConfig(version, "2.1.296", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude296EmbeddedPatchedModuleBytecode})
}
