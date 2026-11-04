package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude318SHA = "677a2ea7bf576cedd20e191d973914ed8a41796f53a63b1923381ddc4d5b5e7d"

var claudeUIPatch_2_1_318 = claudeUIPatchSpec{
	Version: "2.1.318",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude318SHA,
	Apply:   applyClaudeUIPatches_2_1_318,
}

var claude318UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_318(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude318UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude318EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude318SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude318EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude318EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX318(")) && !bytes.Contains(entrySource, claude318RequiredLogoAnchor()) {
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

func disableClaude318EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude318EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude318RequiredLogoAnchor() []byte {
	return []byte("function Ent(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_318(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Ent(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,a=nFs(),t=i.DEMO_VERSION?"/code/claude":On()&&(an("tengu_violin_wood",!1)||an(fcs,!1))?hs(ke()):js(fe()),l=i.CLAUDE_CODE_HIDE_CWD?"":a?` + "`${t} in ${a.replace(/^https?:\\/\\//,\"\")}`" + `:t,c="Codex Plan",m=gt().agent;return{version:o,cwd:l,billingType:c,agentName:m}}`
	return replaceClaude208Function(data, string(claude318RequiredLogoAnchor()), "function rMo(o,a,t,l){", replacement)
}

func patchWhatsNewFeedFunction_2_1_318(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_318(data []byte) bool {
	const anchor = `function me(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	const startMarker = "async function be(e,n,r){"
	const endMarker = "function J(e){"
	const replacement = `async function be(e,n,r){return oo(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},ne)}`
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

func patchModelPickerOptions_2_1_318(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX318(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts318(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function _We(e=!1,n=null){return CDXOpts318(e)}`
	return replaceClaude208Function(data, "function _We(e=!1,n=null){", "function Sb(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_318(data []byte) bool {
	const replacement = `function pSr(e,n){if(n==null)return void 0;let r=CDX318(n),s=e.find((h)=>h.value===r||CDX318(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function pSr(e,n){", "function Nc(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_318(data []byte) bool {
	// With no explicit preference, native Claude selects a separate default row.
	// Our four-tier picker has no such row. Resolve the native effective default
	// through its existing $w helper before matching a visible model option.
	const original = `$t=zt===null?vb:pSr(Rt,zt)??zt`
	const replacement = `$t=pSr(Rt,zt??$w(vb))??zt??vb`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_318(data []byte) bool {
	// The native picker already uses Lmt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function STe(e,n=et(e)){return KW(n)||$G(e,n)}`
	const replacement = `function STe(e,n=et(e)){return KW(n)||Lmt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_318(data []byte) bool {
	const original = `model:B(VFi).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_318(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Hs(){if(Te()!=="firstParty")return!1;return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Hs(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function Y4(){`, `function uTe(){`, `function Y4(){return"Codex"}`),
		replaceFirstFixed(data, `function uTe(){return OJe()?"opus[1m]":"opus"}`, `function uTe(){return"opus"}`),
		replaceFirstFixed(data, `function BNn(e,n,r){if(!Hs())return!1;return sGe(e,r)&&(Ct()||l0()||n)}`, `function BNn(e,n,r){return Hs()&&!!e}`),
		replaceFirstFixed(data, `function jNn(e){if(!Hs())return!1;if(!l0(e))return!1;if(!s_(e))return!1;return Mmt(gt())}`, `function jNn(e){return Hs()&&(be("flagSettings")?.fastMode===!0||Mmt(gt()))}`),
		replaceFirstFixed(data, `function Mmt(e,n){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(be("policySettings")?.fastModePerSessionOptIn===!0)return!1;return(n?n.flagFastMode:be("flagSettings")?.fastMode)===!0}`, `function Mmt(e,n){return e.fastMode===true}`),
		replaceFirstFixed(data, `function s_(e){if(!Hs())return!1;let n=e??NE(),r=Gt(n),s=mw(et(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function s_(e){return Hs()}`),
		replaceFirstFixed(data, `function uL(e,n){if(Ct()){if(e===null)return!!n;return!!n&&s_(e)}if(!s_(e))return!1;return!!n||jNn(e)}`, `function uL(e,n){return Hs()&&(n!==void 0?!!n:Mmt(gt()))}`),
		replaceFirstFixed(data, `b={model:r.model,...Hs()&&{fastMode:r.fastMode}}`, `b={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...pt.gates.fastModeEnabled&&{fastMode:C.options.fastMode}`, `fastMode:C.options.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:jNn(re??null)}`, `fastMode:jNn(re??null)`),
		replaceFirstFixed(data, `...Hs()?{fastMode:R0}:!1`, `fastMode:R0`),
		replaceFirstFixed(data, `ny=Hs()&&Ce(()=>l0())&&!bJe()&&Ce(()=>s_(_n))&&!!zn.fastMode&&!$i(gNt)&&!GNn(zn.model);`, `ny=!!zn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_318(data []byte) bool {
	return replaceClaude208Function(data, "function lGe(e){", "function eWi(e){", `function lGe(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_318(data []byte) bool {
	const original = `function iur(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!rr(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),u=o===-1?r:r.slice(0,o);if(u==="")return null;try{u=decodeURIComponent(u)}catch{}u=Go(u);let l=wn(u)?u:En(fe(),u),h=OG(l);if(h===null)return null;let f=h+s;return NSs(f)?null:f}`
	const replacement = `function iur(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||rr(e):!(e.startsWith("/")&&!rr(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=Go(a);let h=OG(wn(a)?a:En(fe(),a));if(h===null)return null;return NSs(h+=s)?null:h}`
	const originalMarkdown = `function Jt(e){let t=iur(e);if(t===null||t9.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=FSs(t),r=n.some((o)=>/^file:/i.test(HSs(o)));return cfo(t,n)||r?null:t}`
	const replacementMarkdown = `function Jt(e){let t=iur(e,1);if(t===null||t9.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=FSs(t),r=n.some(o=>/^file:/i.test(HSs(o)));return cfo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_318(data []byte) bool {
	return replaceClaude208Function(data, `function u(o,e,n){let{source:d,window:r}=OS`, "export{mzo", `function u(o,e,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_318(data []byte) bool {
	const start = `function pPe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{pPe};", `function pPe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_318(data []byte) bool {
	const target = `Claude Code";function wUe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function wUe(h){`)
}

func patchActiveFastModeBrand_2_1_318(data []byte) bool {
	const target = `"Opus 5.5",ar("claude",no)("Opus 5.5")`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `"Codex+  ",ar("claude",no)("Codex+  ")`)
}

func patchResumeCommandHints_2_1_318(data []byte) bool {
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
	const startMarker = `function E3i({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l=""}){`
	const endMarker = "import{sep as dh}"
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

func claude318SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_318(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_318},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_318},
		{"usage", patchUsageFetchFunction_2_1_318},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_318(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_318},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_318},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_318},
		{"agent-model-validator", patchAgentModelValidator_2_1_318},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_318},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_318},
		{"fast-mode-pricing", patchFastModePricing_2_1_318},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_318},
		{"context-warning", patchContextWarningHint_2_1_318},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_318},
		{"resume-hints", patchResumeCommandHints_2_1_318},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude318RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude318UIBrandingReplacements)
		}},
	}
}

func claude318RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function eR(){return}`, `function fW(){`, `function eR(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Vut(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function glo(e){`, `var G=`, `async function glo(e){if(GH())return"managed_disabled";if(Y1())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function L7t(){`, `async function Q(){`, `async function L7t(){if(GH())return s("managed_disabled",vY);if(Y1())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(zH()||QIn())return!0;`, `var r=`, `function e(){return!GH()&&!Y1()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!zH()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude318Transformations(version string) []claude258Transformation {
	transformations := claude318SourceTransformationsForConfig(version, "2.1.318", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude318EmbeddedPatchedModuleBytecode})
}
