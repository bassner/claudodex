package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude295SHA = "0116ee2e0a513900b633d9951367f18747686478e2b462805b8c31609f047f70"

var claudeUIPatch_2_1_295 = claudeUIPatchSpec{
	Version: "2.1.295",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude295SHA,
	Apply:   applyClaudeUIPatches_2_1_295,
}

var claude295UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude294UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "Claude wants to use your browser" {
			replacements[index].expectedCount = 5
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_295(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude295UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude295EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude295SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude295EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude295EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX295(")) && !bytes.Contains(entrySource, claude295RequiredLogoAnchor()) {
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

func disableClaude295EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude295EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude295RequiredLogoAnchor() []byte {
	return []byte("function Pft(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_295(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Pft(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=KRo(),t=a.DEMO_VERSION?"/code/claude":jn()&&(Vr("tengu_violin_wood",!1)||Vr(Heo,!1))?Cs(we()):rs(se()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",d=ft().agent;return{version:o,cwd:c,billingType:l,agentName:d}}`
	return replaceClaude208Function(data, string(claude295RequiredLogoAnchor()), "function nDr(o,i,t){", replacement)
}

func patchWhatsNewFeedFunction_2_1_295(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function J(i){", replacement)
}

func patchUsageFetchFunction_2_1_295(data []byte) bool {
	const anchor = `var G_={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function hf(e,r){`
	const startMarker = "async function hf(e,r){"
	const endMarker = "function Cs(e){"
	const replacement = `async function hf(e,r){return Zr(r==="at_wall"?"api_usage_fetch_at_wall":r==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let n=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=G_[r],h=await fetch(n+s,{headers:{"Content-Type":"application/json"}});if(!h.ok)throw Error("Auth error: "+h.status);return await h.json()})}`
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

func patchModelPickerOptions_2_1_295(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX295(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts295(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function gTe(e=!1,r=null){return CDXOpts295(e)}`
	return replaceClaude208Function(data, "function gTe(e=!1,r=null){", "function ig(e,r){", replacement)
}

func patchModelPickerSelectionValue_2_1_295(data []byte) bool {
	const replacement = `function RNn(e,r){if(r==null)return void 0;let n=CDX295(r),s=e.find((h)=>h.value===n||CDX295(h.value)===n);return s?.value??n}`
	return replaceClaude208Function(data, "function RNn(e,r){", "function lc(){", replacement)
}

func patchAgentModelValidator_2_1_295(data []byte) bool {
	return replaceFirstFixed(data, `model:U(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_295(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Yo(){if(Pe()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Yo(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function dW(){`, `function Pde(){`, `function dW(){return"Codex"}`),
		replaceFirstFixed(data, `function Pde(){return"opus"+(I6()?"[1m]":"")}`, `function Pde(){return"opus"}`),
		replaceFirstFixed(data, `function tyo(e,n,r){if(!Yo())return!1;return GTe(e,r)&&(Mt()||pR()||n)}`, `function tyo(e,n,r){return Yo()&&!!e}`),
		replaceFirstFixed(data, `function MJt(e){if(!Yo())return!1;if(!pR(e))return!1;if(!B_(e))return!1;return J$n(ft())}`, `function MJt(e){return Yo()&&(me("flagSettings")?.fastMode===!0||J$n(ft()))}`),
		replaceFirstFixed(data, `function J$n(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(me("policySettings")?.fastModePerSessionOptIn===!0)return!1;return me("flagSettings")?.fastMode===!0}`, `function J$n(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function B_(e){if(!Yo())return!1;let n=e??j_(),r=Dt(n),s=h_(We(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function B_(e){return Yo()}`),
		replaceFirstFixed(data, `function Jx(e,n){if(Mt()){if(e===null)return!!n;return!!n&&B_(e)}if(!B_(e))return!1;return!!n||MJt(e)}`, `function Jx(e,n){return Yo()&&(n!==void 0?!!n:J$n(ft()))}`),
		replaceFirstFixed(data, `y={model:r.model,...Yo()&&{fastMode:r.fastMode}}`, `y={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...St.gates.fastModeEnabled&&{fastMode:p.options.fastMode}`, `fastMode:p.options.fastMode`),
		replaceFirstFixed(data, `...Yo()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Yo()&&{fastMode:MJt(J??null)}`, `fastMode:MJt(J??null)`),
		replaceFirstFixed(data, `...Yo()?{fastMode:_D}:!1`, `fastMode:_D`),
		replaceFirstFixed(data, `if(Yo()&&_e(()=>pR())&&!EFe()&&_e(()=>B_(jt))&&!!Kn.fastMode&&!LJt(Kn.model))oC="fast";`, `if(Kn.fastMode)oC="fast";`),
	}
	if bytes.Count(data, []byte(`...Yo()&&{fastMode:_D}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...Yo()&&{fastMode:_D}`, `fastMode:_D`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_295(data []byte) bool {
	return replaceClaude208Function(data, "function qTe(e){", "function PEs(e){", `function qTe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_295(data []byte) bool {
	return replaceClaude208Function(data, `function Kc(e,o,n){let{source:r,window:s}=dv`, "var Yc=", `function Kc(e,o,n){return null}`)
}

func claude295SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_295(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_295},
		{"usage", patchUsageFetchFunction_2_1_295},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_295(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_295},
		{"agent-model-validator", patchAgentModelValidator_2_1_295},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_295},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_295},
		{"context-warning", patchContextWarningHint_2_1_295},
		{"resume-hints", patchResumeCommandHints_2_1_291},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude295RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude295UIBrandingReplacements)
		}},
	}
}

func claude295RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function DL(){return}function Xce(){return}", "function t(e){", `function DL(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Xce(){return}function Hk(){return DL()||_n()?.accessToken}async function CC(e){return Hk()}function FM(){return Xce()??cn().BASE_API_URL}function hNe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function tR(){if(u())return!0;if(O2())return!1;return!IM()&&rst()}`, `function tR(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function Hlo(){if(u())return!0;return!O2()&&!IM()&&s6()}`, `function Hlo(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function Mlo(){`, `var T=`, `async function Mlo(){return!IM()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function Ugr(){if(IM())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function DAt(){", "async function N(){", `async function DAt(){if(IM())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(tR())return!0;try{return s6()&&!IM()&&!O2()&&Gl().source==="none"&&Uf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!Ioo.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!tR()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude295Transformations(version string) []claude258Transformation {
	transformations := claude295SourceTransformationsForConfig(version, "2.1.295", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude295EmbeddedPatchedModuleBytecode})
}
