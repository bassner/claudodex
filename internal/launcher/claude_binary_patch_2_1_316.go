package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude316SHA = "64c327e92b2c5e0f78c0a980c5a518121e1f0d140ca771ca5dd4488d1a2dcb3f"

var claudeUIPatch_2_1_316 = claudeUIPatchSpec{
	Version: "2.1.316",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude316SHA,
	Apply:   applyClaudeUIPatches_2_1_316,
}

var claude316UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_316(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude316UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude316EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude316SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude316EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude316EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX316(")) && !bytes.Contains(entrySource, claude316RequiredLogoAnchor()) {
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

func disableClaude316EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude316EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude316RequiredLogoAnchor() []byte {
	return []byte("function h7e(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_316(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function h7e(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,a=Sbs(),t=i.DEMO_VERSION?"/code/claude":In()&&(Sn("tengu_violin_wood",!1)||Sn(Y9o,!1))?Ps(ve()):Rs(pe()),l=i.CLAUDE_CODE_HIDE_CWD?"":a?` + "`${t} in ${a.replace(/^https?:\\/\\//,\"\")}`" + `:t,c="Codex Plan",m=pt().agent;return{version:o,cwd:l,billingType:c,agentName:m}}`
	return replaceClaude208Function(data, string(claude316RequiredLogoAnchor()), "function n_o(o,a,t,l){", replacement)
}

func patchWhatsNewFeedFunction_2_1_316(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_316(data []byte) bool {
	const anchor = `function fe(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	const startMarker = "async function pe(e,n,r){"
	const endMarker = "function X(e){"
	const replacement = `async function pe(e,n,r){return Yr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},ee)}`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	anchorStart := bytes.Index(data, []byte(anchor))
	startRel := bytes.Index(data[anchorStart:], []byte(startMarker))
	if startRel < 0 {
		return false
	}
	start := anchorStart + startRel
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

func patchModelPickerOptions_2_1_316(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX316(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts316(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function l$e(e=!1,n=null){return CDXOpts316(e)}`
	return replaceClaude208Function(data, "function l$e(e=!1,n=null){", "function J_(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_316(data []byte) bool {
	const replacement = `function Nir(e,n){if(n==null)return void 0;let r=CDX316(n),s=e.find((h)=>h.value===r||CDX316(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function Nir(e,n){", "function Ac(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_316(data []byte) bool {
	// With no explicit preference, native Claude selects a separate default row.
	// Our four-tier picker has no such row. Resolve the native effective default
	// through its existing Iw helper before matching a visible model option.
	const original = `Ht=Dt===null?gb:Nir(Tt,Dt)??Dt`
	const replacement = `Ht=Nir(Tt,Dt??Iw(gb))??Dt??gb`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_316(data []byte) bool {
	// The native picker already uses tlt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function PEe(e,n=Qe(e)){return ej(n)||uG(e,n)}`
	const replacement = `function PEe(e,n=Qe(e)){return ej(n)||tlt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_316(data []byte) bool {
	return replaceFirstFixed(data, `model:U(jhi).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_316(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function _s(){if(Ce()!=="firstParty")return!1;return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function _s(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function Qq(){`, `function kEe(){`, `function Qq(){return"Codex"}`),
		replaceFirstFixed(data, `function kEe(){return M5e()?"opus[1m]":"opus"}`, `function kEe(){return"opus"}`),
		replaceFirstFixed(data, `function tRn(e,n,r){if(!_s())return!1;return XUe(e,r)&&(wt()||AI()||n)}`, `function tRn(e,n,r){return _s()&&!!e}`),
		replaceFirstFixed(data, `function nRn(e){if(!_s())return!1;if(!AI(e))return!1;if(!_y(e))return!1;return Zat(pt())}`, `function nRn(e){return _s()&&(Se("flagSettings")?.fastMode===!0||Zat(pt()))}`),
		replaceFirstFixed(data, `function Zat(e,n){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(Se("policySettings")?.fastModePerSessionOptIn===!0)return!1;return(n?n.flagFastMode:Se("flagSettings")?.fastMode)===!0}`, `function Zat(e,n){return e.fastMode===true}`),
		replaceFirstFixed(data, `function _y(e){if(!_s())return!1;let n=e??jw(),r=Ut(n),s=_b(Qe(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function _y(e){return _s()}`),
		replaceFirstFixed(data, `function R0(e,n){if(wt()){if(e===null)return!!n;return!!n&&_y(e)}if(!_y(e))return!1;return!!n||nRn(e)}`, `function R0(e,n){return _s()&&(n!==void 0?!!n:Zat(pt()))}`),
		replaceFirstFixed(data, `b={model:r.model,..._s()&&{fastMode:r.fastMode}}`, `b={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...wt.gates.fastModeEnabled&&{fastMode:C.options.fastMode}`, `fastMode:C.options.fastMode`),
		replaceFirstFixed(data, `..._s()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `..._s()&&{fastMode:nRn(de??null)}`, `fastMode:nRn(de??null)`),
		replaceFirstFixed(data, `..._s()?{fastMode:Zv}:!1`, `fastMode:Zv`),
		replaceFirstFixed(data, `Uw=_s()&&be(()=>AI())&&!v5e()&&be(()=>_y(Zt))&&!!Wn.fastMode&&!Jr(QPt)&&!oRn(Wn.model);`, `Uw=!!Wn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_316(data []byte) bool {
	return replaceClaude208Function(data, "function ZUe(e){", "function KEi(e){", `function ZUe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_316(data []byte) bool {
	return replaceClaude208Function(data, `function u(o,e,n){let{source:d,window:r}=jb`, "export{pIo", `function u(o,e,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_316(data []byte) bool {
	const start = `function tTe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{tTe};", `function tTe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_316(data []byte) bool {
	const target = `Claude Code";function rHe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function rHe(h){`)
}

func patchActiveFastModeBrand_2_1_316(data []byte) bool {
	const target = `"Opus 5.5",rr("claude",uo)("Opus 5.5")`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `"Codex+  ",rr("claude",uo)("Codex+  ")`)
}

func patchResumeCommandHints_2_1_316(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"Resume this session with:", "Resume with Claudodex:   ", 3},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
		{"Run claude --resume to pick a session, or start a new one.", "Run claudodex -r to pick a session, or start a new one.", 2},
		{"resume it with: claude --resume ", "resume it with: claudodex -r ", 1},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount || len(target.replacement) > len(target.old) {
			return false
		}
	}
	for _, target := range required {
		if !replaceAllFixed(data, target.old, target.replacement) {
			return false
		}
	}
	// The resume command is now built separately from the hint text. Keep the
	// original argument quoting and width handling, and change only the binary.
	const startMarker = "function nOi({sessionId:e,worktree:n,columns:r,envAssignment:s}){"
	const endMarker = "import{sep as kh}"
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let a=s===void 0?\"claude\":`${s} claude`"
	const newCommand = `let a=(s===void 0?"":s+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude316SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_316(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_316},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_316},
		{"usage", patchUsageFetchFunction_2_1_316},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_316(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_316},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_316},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_316},
		{"agent-model-validator", patchAgentModelValidator_2_1_316},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_316},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_316},
		{"fast-mode-pricing", patchFastModePricing_2_1_316},
		{"context-warning", patchContextWarningHint_2_1_316},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_316},
		{"resume-hints", patchResumeCommandHints_2_1_316},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude316RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude316UIBrandingReplacements)
		}},
	}
}

func claude316RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function jT(){return}`, `function _B(){`, `function jT(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function yst(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function O8r(e){`, `var K=`, `async function O8r(e){if(gq())return"managed_disabled";if(rU())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function hqt(){`, `async function Q(){`, `async function hqt(){if(gq())return s("managed_disabled",Wie);if(rU())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(kD()||rXo())return!0;`, `var r=`, `function e(){return!gq()&&!rU()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!kD()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude316Transformations(version string) []claude258Transformation {
	transformations := claude316SourceTransformationsForConfig(version, "2.1.316", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude316EmbeddedPatchedModuleBytecode})
}
