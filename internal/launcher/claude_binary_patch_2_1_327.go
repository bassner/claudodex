package launcher

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude327SHA = "76d48205c926c8a6978cfc2171b631bcd01e20bbadd085638bb2ec57bea91c5b"

var claudeUIPatch_2_1_327 = claudeUIPatchSpec{
	Version: "2.1.327",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude327SHA,
	Apply:   applyClaudeUIPatches_2_1_327,
}

var claude327UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude326UIBrandingReplacements))
	for _, replacement := range claude326UIBrandingReplacements {
		switch replacement.old {
		case "Set up Claude Code's status line UI":
			// The native command now has an unbranded description.
			continue
		case "Set the AI model for Claude Code":
			replacement.expectedCount = 2
		case "Switch between Claude models. Your pick becomes the default for new sessions. For other or older models, use --model.":
			replacement.old = "Your pick is the default for new sessions. For other models, use --model."
			replacement.replacement = "Pick the Codex model for new sessions. For other models, use --model."
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_327(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude327UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude327SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude327EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX327(")) && !bytes.Contains(entrySource, claude327RequiredLogoAnchor()) {
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

func disableClaude327EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude327EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude327RequiredLogoAnchor() []byte {
	return []byte("function Zbt(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_327(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function Zbt(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=I_r(),t=s.DEMO_VERSION?"/code/claude":xn()&&(Qt("tengu_violin_wood",!1)||Qt(bQs,!1))?ss(Ce()):ko(ce()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=wt().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude327RequiredLogoAnchor()), "function m_s(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_327(data []byte) bool {
	const replacement = `var W=async(r,n)=>{return p("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,r),null};`
	return replaceClaude208Function(data, "var W=async(r,n)=>{try{", "function L(r){", replacement)
}

func patchUsageFetchFunction_2_1_327(data []byte) bool {
	const anchor = `function Se(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function xe(e,n,o,u){", "function fe(e){return e===null?void 0:ke.recall(e)}", `async function xe(e,n,o,u){return Do(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);u?.(Date.parse(d.headers.get("date")));return await d.json()},ge)}`)
}

func patchModelPickerOptions_2_1_327(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX327(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts327(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function $Xe(e=!1,n=null){return CDXOpts327(e)}`
	return replaceClaude208Function(data, "function $Xe(e=!1,n=null){", "function vE(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_327(data []byte) bool {
	const replacement = `function HHa(e,n){if(n==null)return void 0;let r=CDX327(n),s=e.find((h)=>h.value===r||CDX327(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function HHa(e,n){", "function Am(){return!(Fr()||{}).availableModels", replacement)
}

func patchModelPickerDefaultFocus_2_1_327(data []byte) bool {
	const original = `function EF(h,R){return R===null?SP:HHa(h,R)??R}`
	const replacement = `function EF(h,R){return HHa(h,R??it())??R??SP}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_327(data []byte) bool {
	// The native picker already uses GDt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function wFe(e,n=ut(e)){return W2(n)||RV(e,n)}`
	const replacement = `function wFe(e,n=ut(e)){return W2(n)||GDt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_327(data []byte) bool {
	const original = `model:B(Z6a).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:r().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_327(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function _i(){return nC()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function _i(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function wht(e){`, `function xPe(){`, `function wht(e){return"Codex"}`),
		replaceClaude208Function(data, `function xPe(){`, `function vlr(e,n,o){`, `function xPe(){return"opus"}`),
		replaceFirstFixed(data, `function vlr(e,n,o){if(!_i())return!1;return IZe(e,o)&&(Et()||UF()||n)}`, `function vlr(e,n,o){return _i()&&!!e}`),
		replaceClaude208Function(data, `function Tlr(e){`, `function BDt(e,n){`, `function Tlr(e){return _i()&&(ye("flagSettings")?.fastMode===!0||BDt(wt()))}`),
		replaceClaude208Function(data, `function BDt(e,n){`, `function hoo(e,n,o){`, `function BDt(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function G_(e){let n=Te(),o=tC[n];`, `function $F(e,n){`, `function G_(e){return _i()}`),
		replaceClaude208Function(data, `function $F(e,n){`, `function C6(e,n){`, `function $F(e,n){return _i()&&(n!==void 0?!!n:BDt(wt()))}`),
		replaceFirstFixed(data, `E={model:o.model,..._i()&&{fastMode:o.fastMode}}`, `E={model:o.model,fastMode:o.fastMode}`),
		replaceFirstFixed(data, `...bt.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `..._i()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `..._i()&&{fastMode:DP}`, `fastMode:DP`),
		replaceFirstFixed(data, `..._i()&&{fastMode:Tlr(Me??null)}`, `fastMode:Tlr(Me??null)`),
		replaceFirstFixed(data, `..._i()?{fastMode:DP}:!1`, `fastMode:DP`),
		replaceFirstFixed(data, `BH=_i()&&Oe(()=>UF())&&!kht()&&Oe(()=>G_(Wn))&&(Te()==="firstParty"||Oe(()=>G_(Cn.model)))&&!!Cn.fastMode&&UJt(Wn)&&!$a(Egt)&&!Alr(Cn.model);`, `BH=!!Cn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_327(data []byte) bool {
	return replaceClaude208Function(data, "function Mlr(e){", "function woo(e){", `function Mlr(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_327(data []byte) bool {
	const original = `function x6r(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!Or(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!cT(l))return null;let f=n(l)?l:c(ce(),l),i=UK(f);if(i===null)return null;let o=i+u;return mmi(o)?null:o}`
	const replacement = `function x6r(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||Or(r):!(r.startsWith("/")&&!Or(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!cT(l))return null;let f=UK(n(l)?l:c(ce(),l));if(f===null)return null;return mmi(f+=s)?null:f}`
	const originalMarkdown = `function Ke(e){let t=x6r(e);if(t===null||Bq.test(t)||qe.test(t)||t!==t.trimEnd())return null;if(rt.test(t))return t;let n=hmi(t),r=n.some((i)=>/^file:/i.test(pmi(i)));return vXo(t,n)||r?null:t}`
	const replacementMarkdown = `function Ke(e){let t=x6r(e,1);if(t===null||Bq.test(t)||qe.test(t)||t!==t.trimEnd())return null;if(rt.test(t))return t;let n=hmi(t),r=n.some(i=>/^file:/i.test(pmi(i)));return vXo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_327(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=iy`, "export{CDs", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_327(data []byte) bool {
	const start = `function ABe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{ABe};", `function ABe(e){return null}`)
}

func patchDefaultTierLabel_2_1_327(data []byte) bool {
	const target = "Default (recommended)"
	return bytes.Count(data, []byte(target)) == 5 && replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_327(data []byte) bool {
	const target = `Claude Code";function HGe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function HGe(h){`)
}

func patchActiveFastModeBrand_2_1_327(data []byte) bool {
	// 2.1.327 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function $Dt(){`, `function wht(e){`, `function $Dt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_327(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"Resume this session with:", "Resume with Claudodex:   ", 3},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a session", "Run claudodex --resume to resume a session", 2},
		{"Run claude --resume to pick a session, or start a new one.", "Run claudodex -r to pick a session, or start a new one.", 2},
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
	const startMarker = `function Qel({sessionId:e,worktree:n,columns:o,envAssignment:a,launchFlags:l="",launcherProgram:u}){`
	const endMarker = `var zn="anthropic-plugin-directory"`
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let p=a===void 0?\"claude\":`${a} claude`"
	const newCommand = `let p=(a===void 0?"":a+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude327SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_327(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_327},
		{"default-tier-label", patchDefaultTierLabel_2_1_327},
		{"whats-new", patchWhatsNewFeedFunction_2_1_327},
		{"usage", patchUsageFetchFunction_2_1_327},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_327(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_327},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_327},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_327},
		{"agent-model-validator", patchAgentModelValidator_2_1_327},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_327},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_327},
		{"fast-mode-pricing", patchFastModePricing_2_1_327},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_327},
		{"context-warning", patchContextWarningHint_2_1_327},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_327},
		{"resume-hints", patchResumeCommandHints_2_1_327},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude327RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude327UIBrandingReplacements)
		}},
	}
}

func claude327RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function BC(){return}`, `function s3(){let e=BC();`, `function BC(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function WPt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function h3o(e){`, `var G="Remote Control is disabled by your organization's policy.`, `async function h3o(e){if(m2())return"managed_disabled";if(wz())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function DEn(){`, `function Z(){return}function Wut(){`, `async function DEn(){if(m2())return l("managed_disabled",Lne);if(wz())return l("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(JU()||nZn()||yti())return!0;`, `var r=`, `function e(){return!m2()&&!wz()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!JU()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude327Transformations(version string) []claude258Transformation {
	transformations := claude327SourceTransformationsForConfig(version, "2.1.327", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude327EmbeddedPatchedModuleBytecode})
}
