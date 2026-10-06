package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude291SHA = "9a1d2ed6bb4421e8fc80c892c0413f293be3ee50ae3d7dda1a7622197a056690"

var claudeUIPatch_2_1_291 = claudeUIPatchSpec{
	Version: "2.1.291",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude291SHA,
	Apply:   applyClaudeUIPatches_2_1_291,
}

var claude291UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude289UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 3
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_291(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude291UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude291EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude291SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude291EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude291EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX291(")) && !bytes.Contains(entrySource, claude291RequiredLogoAnchor()) {
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

func disableClaude291EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude291EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude291RequiredLogoAnchor() []byte {
	return []byte("function hat(){let o=a.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_291(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function hat(){let o=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,i=Spo(),t=Go(se()),c=a.CLAUDE_CODE_HIDE_CWD?"":i?` + "`${t} in ${i.replace(/^https?:\\/\\//,\"\")}`" + `:t,l="Codex Plan",p=ct().agent;return{version:o,cwd:c,billingType:l,agentName:p}}`
	return replaceClaude208Function(data, string(claude291RequiredLogoAnchor()), "function nEr(o,i,t){", replacement)
}

func patchUsageFetchFunction_2_1_291(data []byte) bool {
	const anchor = `var Aae={plain:"/api/oauth/usage",at_wall:"/api/oauth/usage?at_wall=1&skip_spend=1",cedar_ember:"/api/oauth/usage?cedar_ember=1&skip_spend=1"};async function MF(e,n){`
	const startMarker = "async function MF(e,n){"
	const endMarker = "function Tg(e){"
	const replacement = `async function MF(e,n){return Yr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),s=Aae[n],g=await fetch(r+s,{headers:{"Content-Type":"application/json"}});if(!g.ok)throw Error("Auth error: "+g.status);return await g.json()})}`
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

func patchModelPickerOptions_2_1_291(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX291(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function CDXOpts291(e=!1){let n=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna")]}function wwe(e=!1,n=null){return CDXOpts291(e)}`
	return replaceClaude208Function(data, "function wwe(e=!1,n=null){", "function xJ(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_291(data []byte) bool {
	const replacement = `function HSn(e,n){if(n==null)return void 0;let r=CDX291(n),s=e.find((h)=>h.value===r||CDX291(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function HSn(e,n){", "function wM(){", replacement)
}

func patchAgentModelValidator_2_1_291(data []byte) bool {
	return replaceFirstFixed(data, `model:G(["sonnet","opus","haiku","fable"]).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_291(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function No(){if(Me()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function No(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function JB(){`, `function lae(){`, `function JB(){return"Codex"}`),
		replaceFirstFixed(data, `function lae(){return"opus"+(aG()?"[1m]":"")}`, `function lae(){return"opus"}`),
		replaceFirstFixed(data, `function xeo(e,n,r){if(!No())return!1;return zve(e,r)&&(Dt()||GA()||n)}`, `function xeo(e,n,r){return No()&&!!e}`),
		replaceFirstFixed(data, `function fPn(e){if(!No())return!1;if(!GA(e))return!1;if(!Qy(e))return!1;return mPn(ct())}`, `function fPn(e){return No()&&(me("flagSettings")?.fastMode===!0||mPn(ct()))}`),
		replaceFirstFixed(data, `function mPn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(me("policySettings")?.fastModePerSessionOptIn===!0)return!1;return me("flagSettings")?.fastMode===!0}`, `function mPn(e){return e.fastMode===true}`),
		replaceFirstFixed(data, `function Qy(e){if(!No())return!1;let n=e??t_(),r=xt(n),s=Py(Be(r),"fast_mode",r);if(s!==void 0)return s;let g=r.toLowerCase();return g.includes("opus-4-8")||g.includes("opus-5")}`, `function Qy(e){return No()}`),
		replaceFirstFixed(data, `function DR(e,n){if(Dt()){if(e===null)return!!n;return!!n&&Qy(e)}if(!Qy(e))return!1;return!!n||fPn(e)}`, `function DR(e,n){return No()&&(n!==void 0?!!n:mPn(ct()))}`),
		replaceFirstFixed(data, `h={model:r.model,...No()&&{fastMode:r.fastMode}}`, `h={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Pt.gates.fastModeEnabled&&{fastMode:f.options.fastMode}`, `fastMode:f.options.fastMode`),
		replaceFirstFixed(data, `...No()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...No()&&{fastMode:fPn(ut??null)}`, `fastMode:fPn(ut??null)`),
		replaceFirstFixed(data, `...No()?{fastMode:K_}:!1`, `fastMode:K_`),
		replaceFirstFixed(data, `if(No()&&_e(()=>GA())&&!WHe()&&_e(()=>Qy(Nt))&&!!Wn.fastMode&&!F3t(Wn.model))h_="fast";`, `if(Wn.fastMode)h_="fast";`),
	}
	if bytes.Count(data, []byte(`...No()&&{fastMode:K_}`)) != 1 {
		return false
	}
	checks = append(checks, replaceAllFixed(data, `...No()&&{fastMode:K_}`, `fastMode:K_`))
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_291(data []byte) bool {
	return replaceFirstFixed(data, "function Kve(e){return`${w_(e.inputTokens)}/${w_(e.outputTokens)} per Mtok`}", `function Kve(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_291(data []byte) bool {
	return replaceClaude208Function(data, `function Kc(e,o,n){let{source:r,window:s}=iE`, "var Yc=", `function Kc(e,o,n){return null}`)
}

func patchResumeCommandHints_2_1_291(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ", 4},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
		{"Run claude --resume to pick a session, or start a new one.", "Run claudodex --resume to pick a session, or start a new one.", 2},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount {
			return false
		}
	}
	for _, target := range required {
		if !replaceAllFixed(data, target.old, target.replacement) {
			return false
		}
	}
	return true
}

func claude291SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_291(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_283},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_286},
		{"usage", patchUsageFetchFunction_2_1_291},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_291(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_291},
		{"agent-model-validator", patchAgentModelValidator_2_1_291},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_291},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_281},
		{"fast-mode-pricing", patchFastModePricing_2_1_291},
		{"context-warning", patchContextWarningHint_2_1_291},
		{"resume-hints", patchResumeCommandHints_2_1_291},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude291RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude291UIBrandingReplacements)
		}},
	}
}

func claude291RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function qM(){return}function lie(){return}", "function t(e){", `function qM(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function lie(){return}function Cv(){return qM()||gn()?.accessToken}async function yC(e){return Cv()}function q0(){return lie()??pn().BASE_API_URL}function QOe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return t(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function HA(){if(u())return!0;if(vB())return!1;return!V0()&&gZe()}`, `function HA(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function C8r(){if(u())return!0;return!vB()&&!V0()&&RW()}`, `function C8r(){return Bun.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function k8r(){`, `var T=`, `async function k8r(){return!V0()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}async function wrr(){if(V0())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function lbt(){", "async function N(){", `async function lbt(){if(V0())return i("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return i("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(HA())return!0;try{return RW()&&!V0()&&!vB()&&Dc().source==="none"&&yf({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!N3r.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!HA()}`, `get isHidden(){return!1}`)
		}},
	}
}

func claude291Transformations(version string) []claude258Transformation {
	transformations := claude291SourceTransformationsForConfig(version, "2.1.291", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude291EmbeddedPatchedModuleBytecode})
}
