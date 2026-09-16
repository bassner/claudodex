package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_273 = claudeUIPatchSpec{
	Version: "2.1.273",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "953e9880dbcb0b70f31c1f508de6a3fd389753d131688557fd992da9184693fb",
	Apply:   applyClaudeUIPatches_2_1_273,
}

const claude273ActiveFastModeBrandTarget = "\x06\x00\x00\x80\x6c\x06\x46\x00Opus 5\x00\x00\x04\x00\x00\x80\x9e\x43\xd2\x00Opus"

var claude273UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude261UIBrandingReplacements))
	for _, replacement := range claude261UIBrandingReplacements {
		switch replacement.old {
		case "Claude has written up a plan. Would you like to review it as an artifact first?":
			continue
		case "Claude is waiting for your input":
			replacement.expectedCount = 2
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_273(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude273UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude273EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude273SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude273ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude273EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude273EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude273EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX273(")) && !bytes.Contains(entrySource, []byte("function DVo(e=!1){")) {
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

func claude273EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude273ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude273EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude273EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func patchLogoDisplayDataFunction_2_1_273(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Z$e(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,h=FZn(),x=a.DEMO_VERSION?"/code/claude":Mo(te()),O=a.CLAUDE_CODE_HIDE_CWD?"":h?` + "`${x} in ${h.replace(/^https?:\\/\\//,\"\")}`" + `:x,k="Codex Plan",I=Ge().agent;return{version:l,cwd:O,billingType:k,agentName:I}}`
	return replaceClaude208Function(data, "function Z$e(){let l=a.DEMO_VERSION??", "function Jwn(l,h,x){", replacement)
}

func patchWhatsNewFeedFunction_2_1_273(data []byte) bool {
	const replacement = `var ee=async(i,t)=>{return u("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var ee=async(i,t)=>{try{", "function x(L){", replacement)
}

func patchUsageFetchFunction_2_1_273(data []byte) bool {
	const replacement = `function q4n(e,n){return Ir(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=xcs[n],d=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()})}`
	return replaceClaude208Function(data, "function q4n(e,n){", "var Ene=", replacement)
}

func patchModelPickerOptions_2_1_273(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX273(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts273(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function DVo(e=!1){return CDXOpts273(e)}`
	return replaceClaude208Function(data, "function DVo(e=!1){", "function yR(e){", replacement)
}

func patchModelPickerResolver_2_1_273(data []byte) bool {
	return replaceClaude208Function(data, "function jVo(e,n){", "function b1n(e){", `function jVo(e,n){return CDXOpts273(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_273(data []byte) bool {
	return replaceClaude208Function(data, "function tue(e=!1,n=null){", "function $Vo(e,n){", `function tue(e=!1,n=null){return CDXOpts273(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_273(data []byte) bool {
	const replacement = `function SIt(e,n){let r=CDX273(n),s=e.find((d)=>d.value===r||CDX273(d.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function SIt(e,n){if(e.some((d)=>d.value===n))return n;", "function FCn(){", replacement)
}

func patchAgentModelValidator_2_1_273(data []byte) bool {
	return replaceFirstFixed(data, `model:V(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_273(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Kr(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Kr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function a0(){return"Opus 5"}`, `function a0(){return"Codex"}`),
		replaceFirstFixed(data, `function SKe(){return"opus"+(oA()?"[1m]":"")}`, `function SKe(){return"opus"}`),
		replaceFirstFixed(data, `function Bqn(e,n){if(!Kr())return!1;return!!e&&(Nt()||yw()||n)}`, `function Bqn(e,n){return Kr()&&!!e}`),
		replaceFirstFixed(data, `function aBt(e){if(!Kr())return!1;if(!yw(e))return!1;if(!Dm(e))return!1;return jqn(Ge())}`, `function aBt(e){return Kr()&&(ye("flagSettings")?.fastMode===!0||jqn(Ge()))}`),
		replaceFirstFixed(data, `function jqn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function jqn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Dm(e){if(!Kr())return!1;let n=e??Gh(),r=Rt(n),s=Fm(ze(r),"fast_mode",r);if(s!==void 0)return s;let d=r.toLowerCase();return d.includes("opus-4-8")||d.includes("opus-5")}`, `function Dm(e){return Kr()}`),
		replaceFirstFixed(data, `function jE(e,n){if(Nt()){if(e===null)return!!n;return!!n&&Dm(e)}if(!Dm(e))return!1;return!!n||aBt(e)}`, `function jE(e,n){return Kr()&&(n!==void 0?!!n:jqn(Ge()))}`),
		replaceFirstFixed(data, `...Kr()&&{fastMode:aBt(We??null)}`, `fastMode:aBt(We??null)`),
		replaceFirstFixed(data, `...Bo.gates.fastModeEnabled&&{fastMode:Ee.options.fastMode}`, `fastMode:Ee.options.fastMode`),
		replaceFirstFixed(data, `d={model:r.model,...Kr()&&{fastMode:r.fastMode}}`, `d={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Kr()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
	}
	if bytes.Count(data, []byte(`...Kr()&&{fastMode:hc}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...Kr()&&{fastMode:hc}`, `fastMode:hc`),
		replaceFirstFixed(data, `...Kr()?{fastMode:hc}:!1`, `fastMode:hc`),
		replaceFirstFixed(data, `if(Kr()&&y(()=>yw())&&!bfe()&&y(()=>Dm(De))&&!!Pr.fastMode)E$="fast";`, `if(Pr.fastMode)E$="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchActiveFastModeBrand_2_1_273(data []byte) bool {
	const target = "Opus 5\x00\x00\x04\x00\x00\x80\x9e\x43\xd2\x00Opus"
	if bytes.Count(data, []byte(claude273ActiveFastModeBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex \x00\x00\x04\x00\x00\x80\x9e\x43\xd2\x00Opus")
}

func patchFastModePricing_2_1_273(data []byte) bool {
	return replaceFirstFixed(data, "function Afe(e){return`${fH(e.inputTokens)}/${fH(e.outputTokens)} per Mtok`}", `function Afe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_273(data []byte) bool {
	return replaceClaude208Function(data, "function sE(h,E,O){", "var aE=", `function sE(h,E,O){return null}`)
}

func patchCompactProgressCurve_2_1_273(data []byte) bool {
	return replaceFirstFixed(data, `function Gn(t){let o=Math.max(0,t)/1000,l=1-Math.exp(-o/90);return Math.min(95,Math.round(l*100))}`, `function Gn(t){let o=Math.max(0,t)/2000,l=1-Math.exp(-o/90);return Math.min(95,Math.round(l*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_273(data []byte) bool {
	for _, transformation := range claude273RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude273RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function aO(){return}function x9(){return}", "function t(e){", `function aO(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function x9(){return}function pS(){return aO()||en()?.accessToken}async function lw(e){return pS()}function FT(){return x9()??Kt().BASE_API_URL}function ppe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function FE(){if(l())return!0;if(wO())return!1;return!QT()&&Nqe()}`, `function FE(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function u4n(){if(l())return!0;return!wO()&&!QT()&&rF()}`, `function u4n(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function d4n(){if(l())return!0;if(wO())return!1;return rF()&&!QT()&&c()&&await Qd("tengu_ccr_bridge")}`, `async function d4n(){return!wO()&&!QT()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function zun(){", "async function L(){", `async function zun(){if(wO())return i("Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(QT())return i("Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(FE())return!0;try{return rF()&&!QT()&&!wO()&&wc().source==="none"&&Gp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!xjn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!FE()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude273Transformations(version string) []claude258Transformation {
	return claude273TransformationsForConfig(version, "2.1.273", modelconfig.Default())
}

func claude273TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude273SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude273EmbeddedPatchedModuleBytecode})
}

func claude273SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_273(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_273},
		{"usage", patchUsageFetchFunction_2_1_273},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_273(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_273},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_273},
		{"model-selection", patchModelPickerSelectionValue_2_1_273},
		{"agent-model-validator", patchAgentModelValidator_2_1_273},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_273},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_273},
		{"fast-mode-pricing", patchFastModePricing_2_1_273},
		{"context-warning", patchContextWarningHint_2_1_273},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_273},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_273},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude273UIBrandingReplacements)
		}},
	}
}

func claude273ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX273("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function yR("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
