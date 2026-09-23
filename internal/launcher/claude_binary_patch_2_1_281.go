package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude281SHA = "a922981f6f3b55a251ef9f9dbaa0621a5f99cbcb5ca67f8a797476ccfc83f626"

var claudeUIPatch_2_1_281 = claudeUIPatchSpec{
	Version: "2.1.281",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude281SHA,
	Apply:   applyClaudeUIPatches_2_1_281,
}

var claude281UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude278UIBrandingReplacements))
	for _, replacement := range claude278UIBrandingReplacements {
		switch replacement.old {
		case "To modify or remove this hook, edit settings.json directly or ask Claude to help.":
			continue
		case "Claude needs your input":
			replacement.expectedCount = 3
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_281(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude281UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude281SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude281EmbeddedBunModuleRecords(data []byte) ([]claude259BunModuleRecord, bool) {
	return claude260EmbeddedBunModuleRecords(data)
}

func claude281EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX281(")) && !bytes.Contains(entrySource, []byte("function RW(e=!1){")) {
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

func claude281EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude260EmbeddedBunModuleHashes(data)
}

func disableClaude281ChangedEmbeddedModuleBytecode(data []byte, records []claude259BunModuleRecord, hashes [][sha256.Size]byte) bool {
	return disableClaude260ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func disableClaude281EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude281EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude281RequiredLogoAnchor() []byte {
	return []byte("function f8e(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_281(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function f8e(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=iLr(),t=a.DEMO_VERSION?"/code/claude":Co(ne()),c=a.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,i="Codex Plan",m=Ye().agent;return{version:o,cwd:c,billingType:i,agentName:m}}`
	return replaceClaude208Function(data, string(claude281RequiredLogoAnchor()), "function k9n(o,l,t){", replacement)
}

func patchActiveHeaderBrand_2_1_281(data []byte) bool {
	const guard = "\x1b\x00\x00\x80\x59\x03\xfd\x00deferred_no_consent_surface\x00\x0b\x00\x00\x80\xc9\x2f\x86\x00Claude Code\x00\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle"
	const target = "Claude Code\x00\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle"
	if bytes.Count(data, []byte(guard)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x0e\x00\x00\x80\x49\x2a\x8e\x00aiSessionTitle")
}

func patchWhatsNewFeedFunction_2_1_281(data []byte) bool {
	const replacement = `var te=async(i,o)=>{return m("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",o.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,o)=>{try{", "function x(i){", replacement)
}

func patchUsageFetchFunction_2_1_281(data []byte) bool {
	const replacement = `function iA(e,n){return Pr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=uV[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
	return replaceClaude208Function(data, "function iA(e,n){", `import{isAbsolute as _V`, replacement)
}

func patchModelPickerOptions_2_1_281(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX281(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts281(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function RW(e=!1){return CDXOpts281(e)}`
	return replaceClaude208Function(data, "function RW(e=!1){", "function or(e){", replacement)
}

func patchModelPickerResolver_2_1_281(data []byte) bool {
	return replaceClaude208Function(data, "function DW(e,n){", "function egr(e){", `function DW(e,n){return CDXOpts281(e).slice(0,3)}`)
}

func patchModelListOptions_2_1_281(data []byte) bool {
	return replaceClaude208Function(data, "function P6(e=!1){", "var TW=", `function P6(e=!1){return CDXOpts281(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_281(data []byte) bool {
	return replaceClaude208Function(data, "function Que(e=!1,n=null){", "function xW(e,n){", `function Que(e=!1,n=null){return CDXOpts281(e).slice(0,3)}`)
}

func patchModelPickerSelectionValue_2_1_281(data []byte) bool {
	const replacement = `function AXt(e,n){let r=CDX281(n),s=e.find((g)=>g.value===r||CDX281(g.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function AXt(e,n){if(e.some((g)=>g.value===n))return n;", "function yT(){", replacement)
}

func patchAgentModelValidator_2_1_281(data []byte) bool {
	return replaceFirstFixed(data, `model:z(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_281(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function ro(){if(He()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ro(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function IN(){`, `function aZ(){`, `function IN(){return"Codex"}`),
		replaceFirstFixed(data, `function aZ(){return"opus"+(kA()?"[1m]":"")}`, `function aZ(){return"opus"}`),
		replaceFirstFixed(data, `function fkr(e,n){if(!ro())return!1;return!!e&&(Nt()||tC()||n)}`, `function fkr(e,n){return ro()&&!!e}`),
		replaceFirstFixed(data, `function Rrn(e){if(!ro())return!1;if(!tC(e))return!1;if(!yy(e))return!1;return Pl(Ye())}`, `function Rrn(e){return ro()&&(ye("flagSettings")?.fastMode===!0||Pl(Ye()))}`),
		replaceFirstFixed(data, `function Pl(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(ye("policySettings")?.fastModePerSessionOptIn===!0)return!1;return ye("flagSettings")?.fastMode===!0}`, `function Pl(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function yy(e){if(!ro())return!1;let n=e??Vy(),r=Ct(n),s=$m(je(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function yy(e){return ro()}`),
		replaceFirstFixed(data, `function CA(e,n){if(Nt()){if(e===null)return!!n;return!!n&&yy(e)}if(!yy(e))return!1;return!!n||Rrn(e)}`, `function CA(e,n){return ro()&&(n!==void 0?!!n:Pl(Ye()))}`),
		replaceFirstFixed(data, `g={model:r.model,...ro()&&{fastMode:r.fastMode}}`, `g={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Ce.gates.fastModeEnabled&&{fastMode:h.options.fastMode}`, `fastMode:h.options.fastMode`),
		replaceFirstFixed(data, `...ro()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...ro()&&{fastMode:Rrn(Je??null)}`, `fastMode:Rrn(Je??null)`),
		replaceFirstFixed(data, `...ro()?{fastMode:HC}:!1`, `fastMode:HC`),
		replaceFirstFixed(data, `if(ro()&&_(()=>tC())&&!Hve()&&_(()=>yy(Ie))&&!!po.fastMode)$k="fast";`, `if(po.fastMode)$k="fast";`),
	}
	if bytes.Count(data, []byte(`...ro()&&{fastMode:HC}`)) != 2 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...ro()&&{fastMode:HC}`, `fastMode:HC`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchActiveFastModeBrand_2_1_281(data []byte) bool {
	const target = "Opus 5.5\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex+  \x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok")
}

func patchFastModePricing_2_1_281(data []byte) bool {
	return replaceFirstFixed(data, "function Wfe(e){return`${lg(e.inputTokens)}/${lg(e.outputTokens)} per Mtok`}", `function Wfe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_281(data []byte) bool {
	return replaceClaude208Function(data, `function dm(e,n,r){let{source:s,window:g}=GC`, "var um=", `function dm(e,n,r){return null}`)
}

func patchCompactProgressCurve_2_1_281(data []byte) bool {
	return replaceFirstFixed(data, `function hn(l){let t=Math.max(0,l)/1000,m=1-Math.exp(-t/90);return Math.min(95,Math.round(m*100))}`, `function hn(l){let t=Math.max(0,l)/2000,m=1-Math.exp(-t/90);return Math.min(95,Math.round(m*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_281(data []byte) bool {
	for _, transformation := range claude281RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude281RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function sN(){return}function lQ(){return}", "function t(e){", `function sN(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function lQ(){return}function Ow(){return sN()||un()?.accessToken}async function YE(e){return Ow()}function uP(){return lQ()??dn().BASE_API_URL}function oEe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function zv(){if(u())return!0;if(ZM())return!1;return!rP()&&IGe()}`, `function zv(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function fyr(){if(u())return!0;return!ZM()&&!rP()&&_2()}`, `function fyr(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function myr(){`, `var O=`, `async function myr(){return!ZM()&&!rP()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function $On(){if(ZM())return"managed_disabled";if(rP())return"cloud_session";return await myr()?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function UOn(){", "async function v(){", `async function UOn(){if(ZM())return i("managed_disabled","Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).");if(rP())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(zv())return!0;try{return _2()&&!rP()&&!ZM()&&Lc().source==="none"&&Xf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!wfr.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!zv()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude281Transformations(version string) []claude258Transformation {
	transformations := claude281SourceTransformationsForConfig(version, "2.1.281", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude281EmbeddedPatchedModuleBytecode})
}

func claude281SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_281(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_281},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_281},
		{"usage", patchUsageFetchFunction_2_1_281},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_281(data, modelCfg) }},
		{"model-resolver", patchModelPickerResolver_2_1_281},
		{"model-list", patchModelListOptions_2_1_281},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_281},
		{"model-selection", patchModelPickerSelectionValue_2_1_281},
		{"agent-model-validator", patchAgentModelValidator_2_1_281},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_281},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_281},
		{"context-warning", patchContextWarningHint_2_1_281},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_281},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_281},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude281UIBrandingReplacements)
		}},
	}
}

func claude281ModelPickerTierCount(data []byte) int {
	start := bytes.Index(data, []byte("function CDX281("))
	if start < 0 {
		return 0
	}
	end := bytes.Index(data[start:], []byte("function or("))
	if end < 0 {
		return 0
	}
	return strings.Count(string(data[start:start+end]), `r("`)
}
