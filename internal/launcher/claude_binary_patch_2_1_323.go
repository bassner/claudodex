package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude323SHA = "5432918d904a9b6afef7bb79d20df24d76052eabc1f41bceb84091d21f8889a7"

var claudeUIPatch_2_1_323 = claudeUIPatchSpec{
	Version: "2.1.323",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude323SHA,
	Apply:   applyClaudeUIPatches_2_1_323,
}

var claude323UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_323(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude323UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude323EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude323SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude323EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude323EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX323(")) && !bytes.Contains(entrySource, claude323RequiredLogoAnchor()) {
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

func disableClaude323EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude323EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude323RequiredLogoAnchor() []byte {
	return []byte("function jdt(){let e=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_323(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function jdt(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=zni(),t=i.DEMO_VERSION?"/code/claude":An()&&(tn("tengu_violin_wood",!1)||tn(mIs,!1))?$s(Te()):ss(pe()),a=i.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=St().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude323RequiredLogoAnchor()), "function eqo(e,r,t,s){", replacement)
}

func patchWhatsNewFeedFunction_2_1_323(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function J(i){", replacement)
}

func patchUsageFetchFunction_2_1_323(data []byte) bool {
	const anchor = `function Ae(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Re(e,n,r){", "function oe(e){", `async function Re(e,n,r){return Po(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},le)}`)
}

func patchModelPickerOptions_2_1_323(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX323(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts323(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function cVe(e=!1,n=null){return CDXOpts323(e)}`
	return replaceClaude208Function(data, "function cVe(e=!1,n=null){", "function rw(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_323(data []byte) bool {
	const replacement = `function ona(e,n){if(n==null)return void 0;let r=CDX323(n),s=e.find((h)=>h.value===r||CDX323(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function ona(e,n){", "function xp(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_323(data []byte) bool {
	const original = `function tL(h,P){return P===null?LT:ona(h,P)??P}`
	const replacement = `function tL(h,P){return ona(h,P??kd())??P??LT}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_323(data []byte) bool {
	// The native picker already uses pvt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function tIe(e,n=at(e)){return G1(n)||pz(e,n)}`
	const replacement = `function tIe(e,n=at(e)){return G1(n)||pvt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_323(data []byte) bool {
	const original = `model:B(Zua).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_323(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Hs(){return _A()&&!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Hs(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function vot(e){`, `function yve(){`, `function vot(e){return"Codex"}`),
		replaceClaude208Function(data, `function yve(){`, `function B3n(e,n,r){`, `function yve(){return"opus"}`),
		replaceFirstFixed(data, `function B3n(e,n,r){if(!Hs())return!1;return Nqe(e,r)&&(Et()||yL()||n)}`, `function B3n(e,n,r){return Hs()&&!!e}`),
		replaceClaude208Function(data, `function j3n(e){`, `function cvt(e,n){`, `function j3n(e){return Hs()&&(ye("flagSettings")?.fastMode===!0||cvt(St()))}`),
		replaceClaude208Function(data, `function cvt(e,n){`, `function y2r(e,n,r){`, `function cvt(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function kg(e){let n=ve(),r=gA[n];`, `function xM(e,n){`, `function kg(e){return Hs()}`),
		replaceClaude208Function(data, `function xM(e,n){`, `function PN(e,n){`, `function xM(e,n){return Hs()&&(n!==void 0?!!n:cvt(St()))}`),
		replaceFirstFixed(data, `E={model:r.model,...Hs()&&{fastMode:r.fastMode}}`, `E={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...ro.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:YP}`, `fastMode:YP`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:j3n(Re??null)}`, `fastMode:j3n(Re??null)`),
		replaceFirstFixed(data, `...Hs()?{fastMode:YP}:!1`, `fastMode:YP`),
		replaceFirstFixed(data, `dC=Hs()&&xe(()=>yL())&&!Tot()&&xe(()=>kg(io))&&(ve()==="firstParty"||xe(()=>kg(eo.model)))&&!!eo.fastMode&&P6t(io)&&!Ra(BGt)&&!G3n(eo.model);`, `dC=!!eo.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_323(data []byte) bool {
	return replaceClaude208Function(data, "function Ipn(e){", "function tya(e){", `function Ipn(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_323(data []byte) bool {
	const original = `function ATr(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!fr(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),u=o===-1?r:r.slice(0,o);if(u==="")return null;try{u=decodeURIComponent(u)}catch{}if(u=Lo(u),!DC(u))return null;let l=Tn(u)?u:Rn(pe(),u),h=w4(l);if(h===null)return null;let p=h+s;return nWs(p)?null:p}`
	const replacement = `function ATr(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||fr(e):!(e.startsWith("/")&&!fr(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=Lo(a);if(!DC(a))return null;let h=w4(Tn(a)?a:Rn(pe(),a));if(h===null)return null;return nWs(h+=s)?null:h}`
	const originalMarkdown = `function Jt(e){let t=ATr(e);if(t===null||mY.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=rWs(t),r=n.some((o)=>/^file:/i.test(tWs(o)));return IIo(t,n)||r?null:t}`
	const replacementMarkdown = `function Jt(e){let t=ATr(e,1);if(t===null||mY.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=rWs(t),r=n.some(o=>/^file:/i.test(tWs(o)));return IIo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_323(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:s}=PS`, "export{dis", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_323(data []byte) bool {
	const start = `function FDe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{FDe};", `function FDe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_323(data []byte) bool {
	const target = `Claude Code";function qWe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function qWe(h){`)
}

func patchActiveFastModeBrand_2_1_323(data []byte) bool {
	// 2.1.323 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function dvt(){`, `function vot(e){`, `function dvt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_323(data []byte) bool {
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
	const startMarker = `function aRa({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l="",launcherProgram:c}){`
	const endMarker = "import{sep as Vp}"
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let g=s===void 0?\"claude\":`${s} claude`"
	const newCommand = `let g=(s===void 0?"":s+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude323SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_323(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_323},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_323},
		{"usage", patchUsageFetchFunction_2_1_323},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_323(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_323},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_323},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_323},
		{"agent-model-validator", patchAgentModelValidator_2_1_323},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_323},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_323},
		{"fast-mode-pricing", patchFastModePricing_2_1_323},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_323},
		{"context-warning", patchContextWarningHint_2_1_323},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_323},
		{"resume-hints", patchResumeCommandHints_2_1_323},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude323RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude323UIBrandingReplacements)
		}},
	}
}

func claude323RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function Ck(){return}`, `function sG(){`, `function Ck(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Kyt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function uCo(e){`, `var G=`, `async function uCo(e){if(_1())return"managed_disabled";if(fj())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function gan(){`, `function AFa(){`, `async function gan(){if(_1())return s("managed_disabled",NJ);if(fj())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(gN()||sjn())return!0;`, `var r=`, `function e(){return!_1()&&!fj()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!gN()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude323Transformations(version string) []claude258Transformation {
	transformations := claude323SourceTransformationsForConfig(version, "2.1.323", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude323EmbeddedPatchedModuleBytecode})
}
