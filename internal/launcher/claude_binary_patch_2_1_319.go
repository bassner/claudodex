package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude319SHA = "1f53587e92dee8e089fa349188a4cda64fa34ace6e7b0ca2ee278ddaa3cc2ebd"

var claudeUIPatch_2_1_319 = claudeUIPatchSpec{
	Version: "2.1.319",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude319SHA,
	Apply:   applyClaudeUIPatches_2_1_319,
}

var claude319UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_319(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude319UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude319EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude319SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude319EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude319EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX319(")) && !bytes.Contains(entrySource, claude319RequiredLogoAnchor()) {
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

func disableClaude319EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude319EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude319RequiredLogoAnchor() []byte {
	return []byte("function Bot(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_319(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Bot(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,a=y2s(),t=i.DEMO_VERSION?"/code/claude":xn()&&(rn("tengu_violin_wood",!1)||rn(ays,!1))?Es(ve()):Gs(pe()),l=i.CLAUDE_CODE_HIDE_CWD?"":a?` + "`${t} in ${a.replace(/^https?:\\/\\//,\"\")}`" + `:t,c="Codex Plan",m=yt().agent;return{version:o,cwd:l,billingType:c,agentName:m}}`
	return replaceClaude208Function(data, string(claude319RequiredLogoAnchor()), "function uFo(o,a,t,l){", replacement)
}

func patchWhatsNewFeedFunction_2_1_319(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function J(i){", replacement)
}

func patchUsageFetchFunction_2_1_319(data []byte) bool {
	const anchor = `function ne(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	const startMarker = "async function re(e,i,n){"
	const endMarker = "function B(e){"
	const replacement = `async function re(e,i,n){return ao(i==="at_wall"?"api_usage_fetch_at_wall":i==="cedar_ember"?"api_usage_fetch_cedar_ember":i==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+n,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},Z)}`
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

func patchModelPickerOptions_2_1_319(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX319(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts319(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function cGe(e=!1,n=null){return CDXOpts319(e)}`
	return replaceClaude208Function(data, "function cGe(e=!1,n=null){", "function ow(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_319(data []byte) bool {
	const replacement = `function sen(e,n){if(n==null)return void 0;let r=CDX319(n),s=e.find((h)=>h.value===r||CDX319(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function sen(e,n){", "function Cp(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_319(data []byte) bool {
	// With no explicit preference, native Claude selects a separate default row.
	// Our four-tier picker has no such row. Resolve the native effective default
	// through its existing nk helper before matching a visible model option.
	const original = `$t=Ot===null?Fw:sen(xt,Ot)??Ot`
	const replacement = `$t=sen(xt,Ot??nk(Fw))??Ot??Fw`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_319(data []byte) bool {
	// The native picker already uses nyt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function DAe(e,n=rt(e)){return P2(n)||jG(e,n)}`
	const replacement = `function DAe(e,n=rt(e)){return P2(n)||nyt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_319(data []byte) bool {
	const original = `model:B(hVi).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_319(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function $s(){if(Ce()!=="firstParty")return!1;return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function $s(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function W5(){`, `function AAe(){`, `function W5(){return"Codex"}`),
		replaceFirstFixed(data, `function AAe(){return VZe()?"opus[1m]":"opus"}`, `function AAe(){return"opus"}`),
		replaceFirstFixed(data, `function A1n(e,n,r){if(!$s())return!1;return lVe(e,r)&&(Tt()||L0()||n)}`, `function A1n(e,n,r){return $s()&&!!e}`),
		replaceFirstFixed(data, `function R1n(e){if(!$s())return!1;if(!L0(e))return!1;if(!b_(e))return!1;return eyt(yt())}`, `function R1n(e){return $s()&&(Se("flagSettings")?.fastMode===!0||eyt(yt()))}`),
		replaceFirstFixed(data, `function eyt(e,n){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(Se("policySettings")?.fastModePerSessionOptIn===!0)return!1;return(n?n.flagFastMode:Se("flagSettings")?.fastMode)===!0}`, `function eyt(e,n){return e.fastMode===true}`),
		replaceFirstFixed(data, `function b_(e){if(!$s())return!1;let n=e??YE(),r=Vt(n),s=Rw(rt(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function b_(e){return $s()}`),
		replaceFirstFixed(data, `function qL(e,n){if(Tt()){if(e===null)return!!n;return!!n&&b_(e)}if(!b_(e))return!1;return!!n||R1n(e)}`, `function qL(e,n){return $s()&&(n!==void 0?!!n:eyt(yt()))}`),
		replaceFirstFixed(data, `w={model:r.model,...$s()&&{fastMode:r.fastMode}}`, `w={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...lt.gates.fastModeEnabled&&{fastMode:C.options.fastMode}`, `fastMode:C.options.fastMode`),
		replaceFirstFixed(data, `...$s()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...$s()&&{fastMode:R1n(re??null)}`, `fastMode:R1n(re??null)`),
		replaceFirstFixed(data, `...$s()?{fastMode:YH}:!1`, `fastMode:YH`),
		replaceFirstFixed(data, `_x=$s()&&Ee(()=>L0())&&!LZe()&&Ee(()=>b_(so))&&!!Pn.fastMode&&!sa(oUt)&&!P1n(Pn.model);`, `_x=!!Pn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_319(data []byte) bool {
	return replaceClaude208Function(data, "function uVe(e){", "function o5i(e){", `function uVe(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_319(data []byte) bool {
	const original = `function ghr(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!sr(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),u=o===-1?r:r.slice(0,o);if(u==="")return null;try{u=decodeURIComponent(u)}catch{}u=No(u);let l=Rn(u)?u:An(pe(),u),h=wz(l);if(h===null)return null;let p=h+s;return UTs(p)?null:p}`
	const replacement = `function ghr(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||sr(e):!(e.startsWith("/")&&!sr(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=No(a);let h=wz(Rn(a)?a:An(pe(),a));if(h===null)return null;return UTs(h+=s)?null:h}`
	const originalMarkdown = `function Jt(e){let t=ghr(e);if(t===null||G9.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=BTs(t),r=n.some((o)=>/^file:/i.test($Ts(o)));return O_o(t,n)||r?null:t}`
	const replacementMarkdown = `function Jt(e){let t=ghr(e,1);if(t===null||G9.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=BTs(t),r=n.some(o=>/^file:/i.test($Ts(o)));return O_o(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_319(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:s}=KS`, "export{h3o", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_319(data []byte) bool {
	const start = `function AOe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{AOe};", `function AOe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_319(data []byte) bool {
	const target = `Claude Code";function dUe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function dUe(h){`)
}

func patchActiveFastModeBrand_2_1_319(data []byte) bool {
	const target = `"Opus 5.5",dr("claude",io)("Opus 5.5")`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `"Codex+  ",dr("claude",io)("Codex+  ")`)
}

func patchResumeCommandHints_2_1_319(data []byte) bool {
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
	const startMarker = `function JZi({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l=""}){`
	const endMarker = "import{sep as _g}"
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let c=s===void 0?\"claude\":`${s} claude`"
	const newCommand = `let c=(s===void 0?"":s+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude319SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_319(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_319},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_319},
		{"usage", patchUsageFetchFunction_2_1_319},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_319(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_319},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_319},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_319},
		{"agent-model-validator", patchAgentModelValidator_2_1_319},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_319},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_319},
		{"fast-mode-pricing", patchFastModePricing_2_1_319},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_319},
		{"context-warning", patchContextWarningHint_2_1_319},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_319},
		{"resume-hints", patchResumeCommandHints_2_1_319},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude319RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude319UIBrandingReplacements)
		}},
	}
}

func claude319RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function wR(){return}`, `function JW(){`, `function wR(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Emt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function apo(e){`, `var G=`, `async function apo(e){if(IN())return"managed_disabled";if(EB())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function UZt(){`, `async function Q(){`, `async function UZt(){if(IN())return s("managed_disabled",mX);if(EB())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(MN()||fLn())return!0;`, `var r=`, `function e(){return!IN()&&!EB()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!MN()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude319Transformations(version string) []claude258Transformation {
	transformations := claude319SourceTransformationsForConfig(version, "2.1.319", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude319EmbeddedPatchedModuleBytecode})
}
