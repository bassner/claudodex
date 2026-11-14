package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude325SHA = "84590fefe6338be9087f5814b61f9a54c5f21524163a4c127cb5a808e31009f2"

var claudeUIPatch_2_1_325 = claudeUIPatchSpec{
	Version: "2.1.325",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude325SHA,
	Apply:   applyClaudeUIPatches_2_1_325,
}

var claude325UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_325(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude325UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude325EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude325SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude325EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude325EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX325(")) && !bytes.Contains(entrySource, claude325RequiredLogoAnchor()) {
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

func disableClaude325EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude325EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude325RequiredLogoAnchor() []byte {
	return []byte("function bft(){let e=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_325(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function bft(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=kui(),t=i.DEMO_VERSION?"/code/claude":Rn()&&(Jt("tengu_violin_wood",!1)||Jt(M1s,!1))?Ps(Te()):is(ue()),a=i.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=St().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude325RequiredLogoAnchor()), "function VYo(e,r,t,s){", replacement)
}

func patchWhatsNewFeedFunction_2_1_325(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function J(i){", replacement)
}

func patchUsageFetchFunction_2_1_325(data []byte) bool {
	const anchor = `function ve(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Ae(e,n,r,s){", "function oe(e){", `async function Ae(e,n,r,s){return Eo(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);s?.(Date.parse(d.headers.get("date")));return await d.json()},ue)}`)
}

func patchModelPickerOptions_2_1_325(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX325(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts325(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function Dqe(e=!1,n=null){return CDXOpts325(e)}`
	return replaceClaude208Function(data, "function Dqe(e=!1,n=null){", "function lw(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_325(data []byte) bool {
	const replacement = `function cua(e,n){if(n==null)return void 0;let r=CDX325(n),s=e.find((h)=>h.value===r||CDX325(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function cua(e,n){", "function Op(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_325(data []byte) bool {
	const original = `function _N(h,P){return P===null?dT:cua(h,P)??P}`
	const replacement = `function _N(h,P){return cua(h,P??at())??P??dT}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_325(data []byte) bool {
	// The native picker already uses kCt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function K0e(e,n=lt(e)){return $$(n)||qj(e,n)}`
	const replacement = `function K0e(e,n=lt(e)){return $$(n)||kCt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_325(data []byte) bool {
	const original = `model:B(Xba).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_325(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Vs(){return lA()&&!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Vs(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function iat(e){`, `function sTe(){`, `function iat(e){return"Codex"}`),
		replaceClaude208Function(data, `function sTe(){`, `function i7n(e,n,r){`, `function sTe(){return"opus"}`),
		replaceFirstFixed(data, `function i7n(e,n,r){if(!Vs())return!1;return d9e(e,r)&&(wt()||tH()||n)}`, `function i7n(e,n,r){return Vs()&&!!e}`),
		replaceClaude208Function(data, `function a7n(e){`, `function wCt(e,n){`, `function a7n(e){return Vs()&&(Se("flagSettings")?.fastMode===!0||wCt(St()))}`),
		replaceClaude208Function(data, `function wCt(e,n){`, `function VVr(e,n,r){`, `function wCt(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function $g(e){let n=ke(),r=uA[n];`, `function rH(e,n){`, `function $g(e){return Vs()}`),
		replaceClaude208Function(data, `function rH(e,n){`, `function h2(e,n){`, `function rH(e,n){return Vs()&&(n!==void 0?!!n:wCt(St()))}`),
		replaceFirstFixed(data, `E={model:r.model,...Vs()&&{fastMode:r.fastMode}}`, `E={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...ro.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `...Vs()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Vs()&&{fastMode:wx}`, `fastMode:wx`),
		replaceFirstFixed(data, `...Vs()&&{fastMode:a7n(Ce??null)}`, `fastMode:a7n(Ce??null)`),
		replaceFirstFixed(data, `...Vs()?{fastMode:wx}:!1`, `fastMode:wx`),
		replaceFirstFixed(data, `zC=Vs()&&xe(()=>tH())&&!lat()&&xe(()=>$g(Pn))&&(ke()==="firstParty"||xe(()=>$g(eo.model)))&&!!eo.fastMode&&d4t(Pn)&&!_a(S3t)&&!d7n(eo.model);`, `zC=!!eo.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_325(data []byte) bool {
	return replaceClaude208Function(data, "function H_n(e){", "function aAa(e){", `function H_n(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_325(data []byte) bool {
	const original = `function U0r(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!hr(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),u=o===-1?r:r.slice(0,o);if(u==="")return null;try{u=decodeURIComponent(u)}catch{}if(u=Fo(u),!lA(u))return null;let l=Tn(u)?u:Rn(ue(),u),h=p4(l);if(h===null)return null;let p=h+s;return IKs(p)?null:p}`
	const replacement = `function U0r(e,t){let n=/^file:/i.test(e);if(!n&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(e)||hr(e):!(e.startsWith("/")&&!hr(e))))return e;let r=n?e.slice(5):e;if(r.startsWith("//"))r=r.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let o=r.search(/[#?]/),s=o<0?"":r.slice(o),a=o<0?r:r.slice(0,o);if(!a)return null;try{a=decodeURIComponent(a)}catch{}a=Fo(a);if(!lA(a))return null;let h=p4(Tn(a)?a:Rn(ue(),a));if(h===null)return null;return IKs(h+=s)?null:h}`
	const originalMarkdown = `function Vt(e){let t=U0r(e);if(t===null||IX.test(t)||je.test(t)||t!==t.trimEnd())return null;if(Xe.test(t))return t;let n=MKs(t),r=n.some((o)=>/^file:/i.test(OKs(o)));return MFo(t,n)||r?null:t}`
	const replacementMarkdown = `function Vt(e){let t=U0r(e,1);if(t===null||IX.test(t)||je.test(t)||t!==t.trimEnd())return null;if(Xe.test(t))return t;let n=MKs(t),r=n.some(o=>/^file:/i.test(OKs(o)));return MFo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_325(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:s}=YS`, "export{qps", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_325(data []byte) bool {
	const start = `function FHe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{FHe};", `function FHe(e){return null}`)
}

func patchActiveHeaderBrand_2_1_325(data []byte) bool {
	const target = `Claude Code";function vWe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function vWe(h){`)
}

func patchActiveFastModeBrand_2_1_325(data []byte) bool {
	// 2.1.325 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function bCt(){`, `function iat(e){`, `function bCt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_325(data []byte) bool {
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
	const startMarker = `function wHa({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l="",launcherProgram:c}){`
	const endMarker = `var On="anthropic-plugin-directory"`
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

func claude325SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_325(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_325},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_325},
		{"usage", patchUsageFetchFunction_2_1_325},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_325(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_325},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_325},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_325},
		{"agent-model-validator", patchAgentModelValidator_2_1_325},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_325},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_325},
		{"fast-mode-pricing", patchFastModePricing_2_1_325},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_325},
		{"context-warning", patchContextWarningHint_2_1_325},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_325},
		{"resume-hints", patchResumeCommandHints_2_1_325},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude325RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude325UIBrandingReplacements)
		}},
	}
}

func claude325RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function Xk(){return}`, `function KG(){`, `function Xk(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Xwt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function c0o(e){`, `var G=`, `async function c0o(e){if(l$())return"managed_disabled";if(aW())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function wun(){`, `function ZGa(){`, `async function wun(){if(l$())return s("managed_disabled",nZ);if(aW())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(dF()||pVn())return!0;`, `var r=`, `function e(){return!l$()&&!aW()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!dF()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude325Transformations(version string) []claude258Transformation {
	transformations := claude325SourceTransformationsForConfig(version, "2.1.325", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude325EmbeddedPatchedModuleBytecode})
}
