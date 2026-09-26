package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude283SHA = "d8cb1e5c79684cc12a8bfc813e3a2073406921b6245744b3009be3ab5651d21e"

var claudeUIPatch_2_1_283 = claudeUIPatchSpec{
	Version: "2.1.283",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude283SHA,
	Apply:   applyClaudeUIPatches_2_1_283,
}

// Claude Code 2.1.283 retained the Bun module table and fixed-width display
// targets, but changed the minified symbols and removed the old compact
// progress curve. Keep the remaining source transformations version-specific.
var claude283UIBrandingReplacements = claude281UIBrandingReplacements

func applyClaudeUIPatches_2_1_283(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude283UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude283EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude283SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude283EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude283EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX283(")) && !bytes.Contains(entrySource, []byte("function SW(e=!1){")) {
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

func disableClaude283EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude283EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude283RequiredLogoAnchor() []byte {
	return []byte("function q7e(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_283(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function q7e(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=tGr(),t=a.DEMO_VERSION?"/code/claude":xo(oe()),c=a.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",m=Qe().agent;return{version:o,cwd:c,billingType:i,agentName:m}}`
	return replaceClaude208Function(data, string(claude283RequiredLogoAnchor()), "function Wer(o,l,t){", replacement)
}

func patchActiveHeaderBrand_2_1_283(data []byte) bool {
	const guard = "\x1b\x00\x00\x80\x59\x03\xfd\x00deferred_no_consent_surface\x00\x0b\x00\x00\x80\xc9\x2f\x86\x00Claude Code\x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle"
	const target = "Claude Code\x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle"
	if bytes.Count(data, []byte(guard)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x0c\x00\x00\x80\x69\x4d\x7f\x00sessionTitle\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle")
}

func patchWhatsNewFeedFunction_2_1_283(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_283(data []byte) bool {
	const replacement = `function PC(e,n){return Hr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=tY[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function PC(e,n){", "var nY=", replacement)
}

func patchModelPickerOptions_2_1_283(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX283(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts283(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function SW(e=!1){return CDXOpts283(e)}`
	return replaceClaude208Function(data, "function SW(e=!1){", "function Yr(e){", replacement)
}

func patchModelPickerResolver_2_1_283(data []byte) bool {
	return replaceClaude208Function(data, "function OW(e,n){", "function Bkr(e){", `function OW(e,n){return CDXOpts283(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_283(data []byte) bool {
	return replaceClaude208Function(data, "function Pz(e=!1){", "var EW=", `function Pz(e=!1){return CDXOpts283(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_283(data []byte) bool {
	return replaceClaude208Function(data, "function Jme(e=!1,n=null){", "function AW(e,n){", `function Jme(e=!1,n=null){return CDXOpts283(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_283(data []byte) bool {
	const replacement = `function grn(e,n){let r=CDX283(n),s=e.find((g)=>g.value===r||CDX283(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function grn(e,n){if(e.some((g)=>g.value===n))return n;", "function CA(){", replacement)
}

func patchAgentModelValidator_2_1_283(data []byte) bool {
	return replaceFirstFixed(data, `model:G(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_283(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function mo(){if(Ie()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function mo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function a$(){`, `function Nte(){`, `function a$(){return"Codex"}`),
		replaceFirstFixed(data, `function Nte(){return"opus"+(CB()?"[1m]":"")}`, `function Nte(){return"opus"}`),
		replaceFirstFixed(data, `function MLr(e,n){if(!mo())return!1;return!!e&&(Mt()||GC()||n)}`, `function MLr(e,n){return mo()&&!!e}`),
		replaceFirstFixed(data, `function tun(e){if(!mo())return!1;if(!GC(e))return!1;if(!Ky(e))return!1;return Vl(Qe())}`, `function tun(e){return mo()&&(he("flagSettings")?.fastMode===!0||Vl(Qe()))}`),
		replaceFirstFixed(data, `function Vl(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(he("policySettings")?.fastModePerSessionOptIn===!0)return!1;return he("flagSettings")?.fastMode===!0}`, `function Vl(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Ky(e){if(!mo())return!1;let n=e??_y(),r=At(n),s=$h(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Ky(e){return mo()}`),
		replaceFirstFixed(data, `function ak(e,n){if(Mt()){if(e===null)return!!n;return!!n&&Ky(e)}if(!Ky(e))return!1;return!!n||tun(e)}`, `function ak(e,n){return mo()&&(n!==void 0?!!n:Vl(Qe()))}`),
		replaceFirstFixed(data, `g={model:r.model,...mo()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...De.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...mo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...mo()&&{fastMode:tun(Te??null)}`, `fastMode:tun(Te??null)`),
		replaceFirstFixed(data, `...mo()?{fastMode:QR}:!1`, `fastMode:QR`),
		replaceFirstFixed(data, `if(mo()&&b(()=>GC())&&!sTe()&&b(()=>Ky(We))&&!!qn.fastMode)dk="fast";`, `if(qn.fastMode)dk="fast";`),
	}
	if bytes.Count(data, []byte(`...mo()&&{fastMode:QR}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...mo()&&{fastMode:QR}`, `fastMode:QR`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_283(data []byte) bool {
	return replaceFirstFixed(data, "function Jhe(e){return`${Xg(e.inputTokens)}/${Xg(e.outputTokens)} per Mtok`}", `function Jhe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_283(data []byte) bool {
	return replaceClaude208Function(data, `function Zl(e,o,n){let{source:r,window:g}=PA`, "var Ql=", `function Zl(e,o,n){return null}`)
}

func claude283RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function mH(){return}function Qee(){return}", "function t(e){", `function mH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Qee(){return}function mE(){return mH()||un()?.accessToken}async function Tv(e){return mE()}function fI(){return Qee()??ln().BASE_API_URL}function GAe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function LC(){if(u())return!0;if(NF())return!1;return!uI()&&m4e()}`, `function LC(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function QPr(){if(u())return!0;return!NF()&&!uI()&&YU()}`, `function QPr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function ZPr(){`, `var B=`, `async function ZPr(){return!NF()&&!uI()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function K2n(){if(NF())return"managed_disabled";if(uI())return"cloud_session";return await ZPr()?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function Y2n(){", "async function w(){", `async function Y2n(){if(NF())return i("managed_disabled","Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(uI())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(LC())return!0;try{return YU()&&!uI()&&!NF()&&pc().source==="none"&&nf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!wAr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!LC()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude283Transformations(version string) []claude258Transformation {
	transformations := claude283SourceTransformationsForConfig(version, "2.1.283", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude283EmbeddedPatchedModuleBytecode})
}

func claude283SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_283(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_283},
		{"usage", patchUsageFetchFunction_2_1_283},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_283(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_283},
		{"model-list", patchModelListOptions_2_1_283},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_283},
		{"model-selection", patchModelPickerSelectionValue_2_1_283},
		{"agent-model-validator", patchAgentModelValidator_2_1_283},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_283},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_283},
		{"context-warning", patchContextWarningHint_2_1_283},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude283RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude283UIBrandingReplacements)
		}},
	}
}

func claude283ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX283("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function Yr("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
