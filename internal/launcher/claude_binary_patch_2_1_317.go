package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude317SHA = "09f153b46b5df9e8b3454716d682a91901de020cbca630f61dec1a015b32b5e9"

var claudeUIPatch_2_1_317 = claudeUIPatchSpec{
	Version: "2.1.317",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude317SHA,
	Apply:   applyClaudeUIPatches_2_1_317,
}

var claude317UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_317(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude317UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude317EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude317SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude317EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude317EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX317(")) && !bytes.Contains(entrySource, claude317RequiredLogoAnchor()) {
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

func disableClaude317EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude317EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude317RequiredLogoAnchor() []byte {
	return []byte("function kQe(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_317(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function kQe(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,a=nRs(),t=i.DEMO_VERSION?"/code/claude":On()&&(un("tengu_violin_wood",!1)||un(ZZo,!1))?Hs(ve()):Is(pe()),l=i.CLAUDE_CODE_HIDE_CWD?"":a?` + "`${t} in ${a.replace(/^https?:\\/\\//,\"\")}`" + `:t,c="Codex Plan",m=pt().agent;return{version:o,cwd:l,billingType:c,agentName:m}}`
	return replaceClaude208Function(data, string(claude317RequiredLogoAnchor()), "function Sko(o,a,t,l){", replacement)
}

func patchWhatsNewFeedFunction_2_1_317(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_317(data []byte) bool {
	const anchor = `function be(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	const startMarker = "async function ye(e,n,r){"
	const endMarker = "function Z(e){"
	const replacement = `async function ye(e,n,r){return Qr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},te)}`
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

func patchModelPickerOptions_2_1_317(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX317(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts317(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function PUe(e=!1,n=null){return CDXOpts317(e)}`
	return replaceClaude208Function(data, "function PUe(e=!1,n=null){", "function ab(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_317(data []byte) bool {
	const replacement = `function Kur(e,n){if(n==null)return void 0;let r=CDX317(n),s=e.find((h)=>h.value===r||CDX317(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function Kur(e,n){", "function Cc(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_317(data []byte) bool {
	// With no explicit preference, native Claude selects a separate default row.
	// Our four-tier picker has no such row. Resolve the native effective default
	// through its existing Ww helper before matching a visible model option.
	const original = `Dt=$t===null?wb:Kur(kt,$t)??$t`
	const replacement = `Dt=Kur(kt,$t??Ww(wb))??$t??wb`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_317(data []byte) bool {
	// The native picker already uses Ect to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function Tve(e,n=Qe(e)){return Pj(n)||rG(e,n)}`
	const replacement = `function Tve(e,n=Qe(e)){return Pj(n)||Ect(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_317(data []byte) bool {
	const original = `model:U(RTi).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_317(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function ws(){if(Ce()!=="firstParty")return!1;return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function ws(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function C3(){`, `function _ve(){`, `function C3(){return"Codex"}`),
		replaceFirstFixed(data, `function _ve(){return P8e()?"opus[1m]":"opus"}`, `function _ve(){return"opus"}`),
		replaceFirstFixed(data, `function _In(e,n,r){if(!ws())return!1;return kBe(e,r)&&(gt()||QI()||n)}`, `function _In(e,n,r){return ws()&&!!e}`),
		replaceFirstFixed(data, `function SIn(e){if(!ws())return!1;if(!QI(e))return!1;if(!xy(e))return!1;return bct(pt())}`, `function SIn(e){return ws()&&(Se("flagSettings")?.fastMode===!0||bct(pt()))}`),
		replaceFirstFixed(data, `function bct(e,n){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(Se("policySettings")?.fastModePerSessionOptIn===!0)return!1;return(n?n.flagFastMode:Se("flagSettings")?.fastMode)===!0}`, `function bct(e,n){return e.fastMode===true}`),
		replaceFirstFixed(data, `function xy(e){if(!ws())return!1;let n=e??nE(),r=Gt(n),s=Ob(Qe(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function xy(e){return ws()}`),
		replaceFirstFixed(data, `function oH(e,n){if(gt()){if(e===null)return!!n;return!!n&&xy(e)}if(!xy(e))return!1;return!!n||SIn(e)}`, `function oH(e,n){return ws()&&(n!==void 0?!!n:bct(pt()))}`),
		replaceFirstFixed(data, `b={model:r.model,...ws()&&{fastMode:r.fastMode}}`, `b={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Lt.gates.fastModeEnabled&&{fastMode:C.options.fastMode}`, `fastMode:C.options.fastMode`),
		replaceFirstFixed(data, `...ws()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...ws()&&{fastMode:SIn(re??null)}`, `fastMode:SIn(re??null)`),
		replaceFirstFixed(data, `...ws()?{fastMode:kP}:!1`, `fastMode:kP`),
		replaceFirstFixed(data, `YR=ws()&&we(()=>QI())&&!S8e()&&we(()=>xy(tn))&&!!Sn.fastMode&&!ji(jMt)&&!wIn(Sn.model);`, `YR=!!Sn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_317(data []byte) bool {
	return replaceClaude208Function(data, "function ABe(e){", "function xIi(e){", `function ABe(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_317(data []byte) bool {
	const original = `function ior(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!Jn(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),a=o===-1?r:r.slice(0,o);if(a==="")return null;try{a=decodeURIComponent(a)}catch{}a=Po(a);let u=yn(a)?a:Tn(pe(),a),h=y2(u);if(h===null)return null;let f=h+s;return ods(f)?null:f}`
	const replacement = `function ior(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||Jn(e):!(e.startsWith("/")&&!Jn(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=Po(a);let h=y2(yn(a)?a:Tn(pe(),a));if(h===null)return null;return ods(h+=s)?null:h}`
	const originalMarkdown = `function Jt(e){let t=ior(e);if(t===null||T4.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=sds(t),r=n.some((o)=>/^file:/i.test(rds(o)));return Aoo(t,n)||r?null:t}`
	const replacementMarkdown = `function Jt(e){let t=ior(e,1);if(t===null||T4.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=sds(t),r=n.some(o=>/^file:/i.test(rds(o)));return Aoo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_317(data []byte) bool {
	return replaceClaude208Function(data, `function u(o,e,n){let{source:d,window:r}=Jb`, "export{PNo", `function u(o,e,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_317(data []byte) bool {
	const start = `function oAe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{oAe};", `function oAe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_317(data []byte) bool {
	const target = `Claude Code";function yWe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function yWe(h){`)
}

func patchActiveFastModeBrand_2_1_317(data []byte) bool {
	const target = `"Opus 5.5",rr("claude",po)("Opus 5.5")`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `"Codex+  ",rr("claude",po)("Codex+  ")`)
}

func patchResumeCommandHints_2_1_317(data []byte) bool {
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
	const startMarker = `function N$i({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:a=""}){`
	const endMarker = "import{sep as sh}"
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

func claude317SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_317(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_317},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_317},
		{"usage", patchUsageFetchFunction_2_1_317},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_317(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_317},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_317},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_317},
		{"agent-model-validator", patchAgentModelValidator_2_1_317},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_317},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_317},
		{"fast-mode-pricing", patchFastModePricing_2_1_317},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_317},
		{"context-warning", patchContextWarningHint_2_1_317},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_317},
		{"resume-hints", patchResumeCommandHints_2_1_317},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude317RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude317UIBrandingReplacements)
		}},
	}
}

func claude317RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function nA(){return}`, `function ZB(){`, `function nA(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Zat(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function keo(e){`, `var K=`, `async function keo(e){if(XB())return"managed_disabled";if(BU())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function B5t(){`, `async function Q(){`, `async function B5t(){if(XB())return s("managed_disabled",N9);if(BU())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(GD()||Ers())return!0;`, `var r=`, `function e(){return!XB()&&!BU()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!GD()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude317Transformations(version string) []claude258Transformation {
	transformations := claude317SourceTransformationsForConfig(version, "2.1.317", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude317EmbeddedPatchedModuleBytecode})
}
