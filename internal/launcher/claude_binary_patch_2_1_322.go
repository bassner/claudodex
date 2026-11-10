package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude322SHA = "a9d237949db816a0b7afd88a9a8e1df36b3a597a30f5a41d1c0539eb180a3450"

var claudeUIPatch_2_1_322 = claudeUIPatchSpec{
	Version: "2.1.322",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude322SHA,
	Apply:   applyClaudeUIPatches_2_1_322,
}

var claude322UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_322(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude322UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude322EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude322SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude322EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude322EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX322(")) && !bytes.Contains(entrySource, claude322RequiredLogoAnchor()) {
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

func disableClaude322EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude322EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude322RequiredLogoAnchor() []byte {
	return []byte("function Nit(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_322(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Nit(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=O4s(),t=i.DEMO_VERSION?"/code/claude":Rn()&&(rn("tengu_violin_wood",!1)||rn(Bks,!1))?Rs(Ce()):es(pe()),a=i.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=St().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude322RequiredLogoAnchor()), "function eWo(o,l,t,a){", replacement)
}

func patchWhatsNewFeedFunction_2_1_322(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function J(i){", replacement)
}

func patchUsageFetchFunction_2_1_322(data []byte) bool {
	const anchor = `function re(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function ae(e,n,r){", "function q(e){", `async function ae(e,n,r){return _o(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},X)}`)
}

func patchModelPickerOptions_2_1_322(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX322(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts322(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function zze(e=!1,n=null){return CDXOpts322(e)}`
	return replaceClaude208Function(data, "function zze(e=!1,n=null){", "function hE(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_322(data []byte) bool {
	const replacement = `function n6i(e,n){if(n==null)return void 0;let r=CDX322(n),s=e.find((h)=>h.value===r||CDX322(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function n6i(e,n){", "function hm(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_322(data []byte) bool {
	const original = `function cN(h,M){return M===null?kR:n6i(h,M)??M}`
	const replacement = `function cN(h,M){return n6i(h,M??bd())??M??kR}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_322(data []byte) bool {
	// The native picker already uses fSt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function cxe(e,n=it(e)){return NG(n)||$$(e,n)}`
	const replacement = `function cxe(e,n=it(e)){return NG(n)||fSt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_322(data []byte) bool {
	const original = `model:B(C8i).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_322(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Hs(){return Kk()&&!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Hs(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function ctt(e){`, `function ewe(){`, `function ctt(e){return"Codex"}`),
		replaceClaude208Function(data, `function ewe(){`, `function S2n(e,n,r){`, `function ewe(){return"opus"}`),
		replaceFirstFixed(data, `function S2n(e,n,r){if(!Hs())return!1;return Y6e(e,r)&&(vt()||hD()||n)}`, `function S2n(e,n,r){return Hs()&&!!e}`),
		replaceClaude208Function(data, `function b2n(e){`, `function cSt(e,n){`, `function b2n(e){return Hs()&&(be("flagSettings")?.fastMode===!0||cSt(St()))}`),
		replaceClaude208Function(data, `function cSt(e,n){`, `function TLr(e,n,r){`, `function cSt(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function ng(e){let n=ve(),r=Gk[n];`, `function TM(e,n){`, `function ng(e){return Hs()}`),
		replaceClaude208Function(data, `function TM(e,n){`, `function MH(e,n){`, `function TM(e,n){return Hs()&&(n!==void 0?!!n:cSt(St()))}`),
		replaceFirstFixed(data, `w={model:r.model,...Hs()&&{fastMode:r.fastMode}}`, `w={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Rt.gates.fastModeEnabled&&{fastMode:T.options.fastMode}`, `fastMode:T.options.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:zx}`, `fastMode:zx`),
		replaceFirstFixed(data, `...Hs()&&{fastMode:b2n(Q??null)}`, `fastMode:b2n(Q??null)`),
		replaceFirstFixed(data, `...Hs()?{fastMode:zx}:!1`, `fastMode:zx`),
		replaceFirstFixed(data, `oA=Hs()&&Ee(()=>hD())&&!ftt()&&Ee(()=>ng(so))&&(ve()==="firstParty"||Ee(()=>ng(zn.model)))&&!!zn.fastMode&&B1t(so)&&!ia($Bt)&&!E2n(zn.model);`, `oA=!!zn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_322(data []byte) bool {
	return replaceClaude208Function(data, "function kln(e){", "function wZi(e){", `function kln(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_322(data []byte) bool {
	const original = `function hbr(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!dr(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),u=o===-1?r:r.slice(0,o);if(u==="")return null;try{u=decodeURIComponent(u)}catch{}if(u=Do(u),!YC(u))return null;let l=wn(u)?u:En(pe(),u),h=_V(l);if(h===null)return null;let p=h+s;return HMs(p)?null:p}`
	const replacement = `function hbr(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||dr(e):!(e.startsWith("/")&&!dr(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=Do(a);if(!YC(a))return null;let h=_V(wn(a)?a:En(pe(),a));if(h===null)return null;return HMs(h+=s)?null:h}`
	const originalMarkdown = `function Jt(e){let t=hbr(e);if(t===null||D8.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=NMs(t),r=n.some((o)=>/^file:/i.test(LMs(o)));return tvo(t,n)||r?null:t}`
	const replacementMarkdown = `function Jt(e){let t=hbr(e,1);if(t===null||D8.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=NMs(t),r=n.some(o=>/^file:/i.test(LMs(o)));return tvo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_322(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:s}=ub`, "export{_7o", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_322(data []byte) bool {
	const start = `function ZIe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{ZIe};", `function ZIe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_322(data []byte) bool {
	const target = `Claude Code";function VUe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function VUe(h){`)
}

func patchActiveFastModeBrand_2_1_322(data []byte) bool {
	// 2.1.322 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function dSt(){`, `function ctt(e){`, `function dSt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_322(data []byte) bool {
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
	const startMarker = `function bla({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l=""}){`
	const endMarker = "import{sep as Rg}"
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

func claude322SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_322(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_322},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_322},
		{"usage", patchUsageFetchFunction_2_1_322},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_322(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_322},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_322},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_322},
		{"agent-model-validator", patchAgentModelValidator_2_1_322},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_322},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_322},
		{"fast-mode-pricing", patchFastModePricing_2_1_322},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_322},
		{"context-warning", patchContextWarningHint_2_1_322},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_322},
		{"resume-hints", patchResumeCommandHints_2_1_322},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude322RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude322UIBrandingReplacements)
		}},
	}
}

func claude322RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function Fv(){return}`, `function nG(){`, `function Fv(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Cgt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function M_o(e){`, `var G=`, `async function M_o(e){if(gF())return"managed_disabled";if(dj())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function Knn(){`, `async function Q(){`, `async function Knn(){if(gF())return s("managed_disabled",o7);if(dj())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(yF()||zFn())return!0;`, `var r=`, `function e(){return!gF()&&!dj()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!yF()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude322Transformations(version string) []claude258Transformation {
	transformations := claude322SourceTransformationsForConfig(version, "2.1.322", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude322EmbeddedPatchedModuleBytecode})
}
