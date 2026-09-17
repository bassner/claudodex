package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude274SHA = "3509913f9d1576316c8845b88837f8fd3bbbcf26625833ac82cfb6b8985da94a"

var claudeUIPatch_2_1_274 = claudeUIPatchSpec{
	Version: "2.1.274",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude274SHA,
	Apply:   applyClaudeUIPatches_2_1_274,
}

var claude274UIBrandingReplacements = claude273UIBrandingReplacements

const claude274ActiveHeaderBrandGuard = "\x0b\x00\x00\x80\xc9/\x86\x00Claude Code\x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle"

func applyClaudeUIPatches_2_1_274(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude274UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude274EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude274SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude274ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude274EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude274EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude274EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX274(")) && !bytes.Contains(entrySource, []byte("function CIo(e=!1){")) {
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

func claude274EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude274ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude274EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude274EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude274RequiredLogoAnchor() []byte {
	return []byte("function S2e(){let l=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_274(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function S2e(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,h=nir(),b=a.DEMO_VERSION?"/code/claude":Mo(te()),x=a.CLAUDE_CODE_HIDE_CWD?"":h?` + "`${b} in ${h.replace(/^https?:\\/\\//,\"\")}`" + `:b,k="Codex Plan",I=qe().agent;return{version:l,cwd:x,billingType:k,agentName:I}}`
	return replaceClaude208Function(data, string(claude274RequiredLogoAnchor()), "function BTn(l,h,b){", replacement)
}

func patchWhatsNewFeedFunction_2_1_274(data []byte) bool {
	const replacement = `var ee=async(i,t)=>{return u("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var ee=async(i,t)=>{try{", "function x(X){", replacement)
}

func patchUsageFetchFunction_2_1_274(data []byte) bool {
	const replacement = `function SEn(e,n){return Lr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Q$o[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function SEn(e,n){", "var hoe=", replacement)
}

func patchModelPickerOptions_2_1_274(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX274(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts274(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function CIo(e=!1){return CDXOpts274(e)}`
	return replaceClaude208Function(data, "function CIo(e=!1){", "function Kx(e){", replacement)
}

func patchModelPickerResolver_2_1_274(data []byte) bool {
	return replaceClaude208Function(data, "function LIo(e,n){", "function Jjn(e){", `function LIo(e,n){return CDXOpts274(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_274(data []byte) bool {
	return replaceClaude208Function(data, "function F6(e=!1){", "var AIo=", `function F6(e=!1){return CDXOpts274(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_274(data []byte) bool {
	return replaceClaude208Function(data, "function ope(e=!1,n=null){", "function IIo(e,n){", `function ope(e=!1,n=null){return CDXOpts274(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_274(data []byte) bool {
	const replacement = `function tNt(e,n){let r=CDX274(n),s=e.find((g)=>g.value===r||CDX274(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function tNt(e,n){if(e.some((g)=>g.value===n))return n;", "function emn(){", replacement)
}

func patchAgentModelValidator_2_1_274(data []byte) bool {
	return replaceFirstFixed(data, `model:V(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_274(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Qr(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Qr(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function t0(){return"Opus 5"}`, `function t0(){return"Codex"}`),
		replaceFirstFixed(data, `function v8e(){return"opus"+(_v()?"[1m]":"")}`, `function v8e(){return"opus"}`),
		replaceFirstFixed(data, `function V7n(e,n){if(!Qr())return!1;return!!e&&(Ft()||Zw()||n)}`, `function V7n(e,n){return Qr()&&!!e}`),
		replaceFirstFixed(data, `function HWt(e){if(!Qr())return!1;if(!Zw(e))return!1;if(!Gm(e))return!1;return eH(qe())}`, `function HWt(e){return Qr()&&(me("flagSettings")?.fastMode===!0||eH(qe()))}`),
		replaceFirstFixed(data, `function eH(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(me("policySettings")?.fastModePerSessionOptIn===!0)return!1;return me("flagSettings")?.fastMode===!0}`, `function eH(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Gm(e){if(!Qr())return!1;let n=e??cy(),r=xt(n),s=Ym(Ge(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Gm(e){return Qr()}`),
		replaceFirstFixed(data, `function hv(e,n){if(Ft()){if(e===null)return!!n;return!!n&&Gm(e)}if(!Gm(e))return!1;return!!n||HWt(e)}`, `function hv(e,n){return Qr()&&(n!==void 0?!!n:eH(qe()))}`),
		replaceFirstFixed(data, `g={model:r.model,...Qr()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...$o.gates.fastModeEnabled&&{fastMode:Oe.options.fastMode}`, `fastMode:Oe.options.fastMode`),
		replaceFirstFixed(data, `...Qr()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Qr()&&{fastMode:HWt(Ge??null)}`, `fastMode:HWt(Ge??null)`),
		replaceFirstFixed(data, `...Qr()?{fastMode:dd}:!1`, `fastMode:dd`),
		replaceFirstFixed(data, `if(Qr()&&y(()=>Zw())&&!Sge()&&y(()=>Gm(Me))&&!!Pr.fastMode)pN="fast";`, `if(Pr.fastMode)pN="fast";`),
	}
	if bytes.Count(data, []byte(`...Qr()&&{fastMode:dd}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Qr()&&{fastMode:dd}`, `fastMode:dd`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_274(data []byte) bool {
	return replaceFirstFixed(data, "function Ege(e){return`${oH(e.inputTokens)}/${oH(e.outputTokens)} per Mtok`}", `function Ege(e){return"Codex priority"}`)
}

func patchActiveHeaderBrand_2_1_274(data []byte) bool {
	const target = "Claude Code\x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle"
	if bytes.Count(data, []byte(claude274ActiveHeaderBrandGuard)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle")
}

func patchContextWarningHint_2_1_274(data []byte) bool {
	return replaceClaude208Function(data, "function rL(h,E,O){", "var sL=", `function rL(h,E,O){return null}`)
}

func patchRemoteControlRuntimeFunctions_2_1_274(data []byte) bool {
	for _, transformation := range claude274RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude274RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function XO(){return}function $9(){return}", "function t(e){", `function XO(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function $9(){return}function BS(){return XO()||Zt()?.accessToken}async function zw(e){return BS()}function yk(){return $9()??Xt().BASE_API_URL}function dme(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function pv(){if(l())return!0;if(lD())return!1;return!Rk()&&N9e()}`, `function pv(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function dYn(){if(l())return!0;return!lD()&&!Rk()&&r$()}`, `function dYn(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function pYn(){if(l())return!0;if(lD())return!1;return r$()&&!Rk()&&c()&&await bp("tengu_ccr_bridge")}`, `async function pYn(){return!lD()&&!Rk()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function nhn(){", "async function L(){", `async function nhn(){if(lD())return i("Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(Rk())return i("Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(pv())return!0;try{return r$()&&!Rk()&&!lD()&&Dc().source==="none"&&df({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!ZVn.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!pv()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude274Transformations(version string) []claude258Transformation {
	return claude274TransformationsForConfig(version, "2.1.274", modelconfig.Default())
}

func claude274TransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	transformations := claude274SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg)
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude274EmbeddedPatchedModuleBytecode})
}

func claude274SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_274(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_274},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_274},
		{"usage", patchUsageFetchFunction_2_1_274},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_274(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_274},
		{"model-list", patchModelListOptions_2_1_274},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_274},
		{"model-selection", patchModelPickerSelectionValue_2_1_274},
		{"agent-model-validator", patchAgentModelValidator_2_1_274},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_274},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_273},
		{"fast-mode-pricing", patchFastModePricing_2_1_274},
		{"context-warning", patchContextWarningHint_2_1_274},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_273},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_274},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude274UIBrandingReplacements)
		}},
	}
}

func claude274ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX274("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function Kx("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
