package launcher

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude329SHA = "8c1729568b19cdadedd0a978bf692f9f279ef554c8a333c4e132b94fd8725550"

var claudeUIPatch_2_1_329 = claudeUIPatchSpec{
	Version: "2.1.329",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude329SHA,
	Apply:   applyClaudeUIPatches_2_1_329,
}

var claude329UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude327UIBrandingReplacements...)
	for index := range replacements {
		switch replacements[index].old {
		case "Welcome to Claude Code":
			replacements[index].expectedCount = 6
		case "To add hooks, edit settings.json directly or ask Claude":
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_329(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude329UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude329SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude329EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX329(")) && !bytes.Contains(entrySource, claude329RequiredLogoAnchor()) {
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

func disableClaude329EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude329EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude329RequiredLogoAnchor() []byte {
	return []byte("function YAt(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_329(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function YAt(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=DPr(),t=s.DEMO_VERSION?"/code/claude":On()&&(Ht("tengu_violin_wood",!1)||Ht(Jyi,!1))?ds(Ae()):vo(pe()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=At().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude329RequiredLogoAnchor()), "function pHs(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_329(data []byte) bool {
	const replacement = `var W=async(r,n)=>{return p("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,r),null};`
	return replaceClaude208Function(data, `var W=async(r,n)=>{try{await Jt(yGs(n.storageV5),500,"Timeout")}`, "function w(r){", replacement)
}

func patchUsageFetchFunction_2_1_329(data []byte) bool {
	const anchor = `function ke(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Ue(e,r,o,d){", "function ge(e){return e===null?void 0:Oe.recall(e)}", `async function Ue(e,n,o,u){return No(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);u?.(Date.parse(d.headers.get("date")));return await d.json()},ye)}`)
}

func patchModelPickerOptions_2_1_329(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX329(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts329(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function Jnt(e=!1,r=null){return CDXOpts329(e)}`
	return replaceClaude208Function(data, "function Jnt(e=!1,r=null){", "function PE(e,r){", replacement)
}

func patchModelPickerSelectionValue_2_1_329(data []byte) bool {
	const replacement = `function QQa(e,r){if(r==null)return void 0;let n=CDX329(r),s=e.find((h)=>h.value===n||CDX329(h.value)===n);return s?.value??n}`
	return replaceClaude208Function(data, "function QQa(e,r){", "function Mm(){return!(Ur()||{}).availableModels", replacement)
}

func patchModelPickerDefaultFocus_2_1_329(data []byte) bool {
	const original = `function R0(h,T){return T===null?MP:QQa(h,T)??T}`
	const replacement = `function R0(h,T){return QQa(h,T??ct())??T??MP}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_329(data []byte) bool {
	// The native picker already uses KBt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function eje(e,r=_t(e)){return $2(r)||OK(e,r)}`
	const replacement = `function eje(e,r=_t(e)){return $2(r)||KBt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_329(data []byte) bool {
	const original = `model:U(zcl).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:n().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_329(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Bi(){return yA()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Bi(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function pkt(e){`, `function KHe(){`, `function pkt(e){return"Codex"}`),
		replaceClaude208Function(data, `function KHe(){`, `function uwr(e,r,o){`, `function KHe(){return"opus"}`),
		replaceFirstFixed(data, `function uwr(e,r,o){if(!Bi())return!1;return jst(e,o)&&(bt()||NU()||r)}`, `function uwr(e,r,o){return Bi()&&!!e}`),
		replaceClaude208Function(data, `function fwr(e){`, `function GBt(e,r){`, `function fwr(e){return Bi()&&(ye("flagSettings")?.fastMode===!0||GBt(At()))}`),
		replaceClaude208Function(data, `function GBt(e,r){`, `function Hyo(e,r,o){`, `function GBt(e,r){return e.fastMode===true}`),
		replaceClaude208Function(data, `function My(e){let r=Ee(),o=hA[r];`, `function FU(e,r){`, `function My(e){return Bi()}`),
		replaceClaude208Function(data, `function FU(e,r){`, `function S6(e,r){`, `function FU(e,r){return Bi()&&(r!==void 0?!!r:GBt(At()))}`),
		replaceFirstFixed(data, `k={model:r.model,...Bi()&&{fastMode:r.fastMode}}`, `k={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...oo.gates.fastModeEnabled&&{fastMode:A.options.fastMode}`, `fastMode:A.options.fastMode`),
		replaceFirstFixed(data, `...Bi()&&{fastMode:o.fastMode}`, `fastMode:o.fastMode`),
		replaceFirstFixed(data, `...Bi()&&{fastMode:HM}`, `fastMode:HM`),
		replaceFirstFixed(data, `...Bi()&&{fastMode:fwr(He??null)}`, `fastMode:fwr(He??null)`),
		replaceFirstFixed(data, `...Bi()?{fastMode:HM}:!1`, `fastMode:HM`),
		replaceFirstFixed(data, `SU=Bi()&&Le(()=>NU())&&!hkt()&&Le(()=>My(bo))&&(Ee()==="firstParty"||Le(()=>My(Qn.model)))&&!!Qn.fastMode&&aan(bo)&&!Gl(_Et)&&!mwr(Qn.model);`, `SU=!!Qn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_329(data []byte) bool {
	return replaceClaude208Function(data, "function wwr(e){", "function $yo(e){", `function wwr(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_329(data []byte) bool {
	const original = `function uJr(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!Lr(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!cC(l))return null;let f=n(l)?l:c(pe(),l),i=fz(f);if(i===null)return null;let o=i+u;return G0i(o)?null:o}`
	const replacement = `function uJr(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||Lr(r):!(r.startsWith("/")&&!Lr(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!cC(l))return null;let f=fz(n(l)?l:c(pe(),l));if(f===null)return null;return G0i(f+=s)?null:f}`
	const originalMarkdown = `function je(e){let t=uJr(e);if(t===null||i9.test(t)||Ke.test(t)||t!==t.trimEnd())return null;if(Ye.test(t))return t;let n=z0i(t),r=n.some((i)=>/^file:/i.test(j0i(i)));return Zfs(t,n)||r?null:t}`
	const replacementMarkdown = `function je(e){let t=uJr(e,1);if(t===null||i9.test(t)||Ke.test(t)||t!==t.trimEnd())return null;if(Ye.test(t))return t;let n=z0i(t),r=n.some(i=>/^file:/i.test(j0i(i)));return Zfs(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_329(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=Jy`, "export{c3s", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_329(data []byte) bool {
	const start = `function h6e(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{h6e};", `function h6e(e){return null}`)
}

func patchDefaultTierLabel_2_1_329(data []byte) bool {
	const target = "Default (recommended)"
	return bytes.Count(data, []byte(target)) == 5 && replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_329(data []byte) bool {
	const target = `Claude Code";function GQe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function GQe(h){`)
}

func patchActiveFastModeBrand_2_1_329(data []byte) bool {
	// 2.1.329 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function jBt(){`, `function pkt(e){`, `function jBt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_329(data []byte) bool {
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
	const startMarker = `function oxl({sessionId:e,worktree:o,columns:r,envAssignment:a,launchFlags:l="",launcherProgram:u}){`
	const endMarker = `var Vn="anthropic-plugin-directory"`
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let g=a===void 0?\"claude\":`${a} claude`"
	const newCommand = `let g=(a===void 0?"":a+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude329SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_329(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_329},
		{"default-tier-label", patchDefaultTierLabel_2_1_329},
		{"whats-new", patchWhatsNewFeedFunction_2_1_329},
		{"usage", patchUsageFetchFunction_2_1_329},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_329(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_329},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_329},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_329},
		{"agent-model-validator", patchAgentModelValidator_2_1_329},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_329},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_329},
		{"fast-mode-pricing", patchFastModePricing_2_1_329},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_329},
		{"context-warning", patchContextWarningHint_2_1_329},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_329},
		{"resume-hints", patchResumeCommandHints_2_1_329},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude329RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude329UIBrandingReplacements)
		}},
	}
}

func claude329RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function DA(){return}function sNt(e=Vae(),n){`, `async function JN(e){if(!(re()&&e!==void 0))`, `function DA(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN||void 0}function sNt(e=Vae(),n){return !DA()&&(ITe(e)||n!==void 0&&n!==Zn()?.accessToken&&ITe(e,!0))}function d3(){return DA()??(Bn()&&Ot()&&!dJ()?Fne(Zn())?.accessToken:void 0)}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function rss(e){`, `var z="Remote Control is disabled by your organization's policy.`, `async function rss(e){if(d2())return"managed_disabled";if(TK())return"cloud_session";if(znt()==="denied")return"org_policy_denied";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function iMn(){`, `function D(){if(my()||t2n()!==null)`, `async function iMn(){if(d2())return l("managed_disabled",rie);if(znt()==="denied")return{reason:j(),code:"org_policy_denied",orgPolicyDenied:!0};if(TK())return l("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(iL()||Tur()||Awi())return!0;`, `var r=`, `function e(){return!d2()&&!TK()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!iL()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude329Transformations(version string) []claude258Transformation {
	transformations := claude329SourceTransformationsForConfig(version, "2.1.329", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude329EmbeddedPatchedModuleBytecode})
}
