package launcher

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude330SHA = "f7c5093b9dd6560d08af28bc27b123fb55091deae3180ac39df199522da86815"

var claudeUIPatch_2_1_330 = claudeUIPatchSpec{
	Version: "2.1.330",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude330SHA,
	Apply:   applyClaudeUIPatches_2_1_330,
}

var claude330UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude329UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_330(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude330UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude330SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude330EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX330(")) && !bytes.Contains(entrySource, claude330RequiredLogoAnchor()) {
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

func disableClaude330EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude330EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude330RequiredLogoAnchor() []byte {
	return []byte("function KOt(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_330(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function KOt(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=sNr(),t=s.DEMO_VERSION?"/code/claude":Hn()&&(Lt("tengu_violin_wood",!1)||Lt(wxi,!1))?os(Te()):Po(pe()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=vt().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude330RequiredLogoAnchor()), "function vWs(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_330(data []byte) bool {
	const replacement = `var W=async(r,n)=>{return p("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,r),null};`
	return replaceClaude208Function(data, `var W=async(r,n)=>{try{await qt(A9s(n.storageV5),500,"Timeout")}`, "function L(r){", replacement)
}

func patchUsageFetchFunction_2_1_330(data []byte) bool {
	const anchor = `function ke(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Ue(e,r,o,d){", "function pe(e){return e===null?void 0:Oe.recall(e)}", `async function Ue(e,n,o,u){return qo(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);u?.(Date.parse(d.headers.get("date")));return await d.json()},_e)}`)
}

func patchModelPickerOptions_2_1_330(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX330(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts330(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function dit(e=!1,r=null){return CDXOpts330(e)}`
	return replaceClaude208Function(data, "function dit(e=!1,r=null){", "function qb(e,r){", replacement)
}

func patchModelPickerSelectionValue_2_1_330(data []byte) bool {
	const replacement = `function Pcl(e,r){if(r==null)return void 0;let n=CDX330(r),s=e.find((h)=>h.value===n||CDX330(h.value)===n);return s?.value??n}`
	return replaceClaude208Function(data, "function Pcl(e,r){", "function $f(){return!(zr()||{}).availableModels", replacement)
}

func patchModelPickerDefaultFocus_2_1_330(data []byte) bool {
	const original = `function cB(h,T){return T===null?nA:Pcl(h,T)??T}`
	const replacement = `function cB(h,T){return Pcl(h,T??dt())??T??nA}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_330(data []byte) bool {
	// The native picker already uses dCt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function _Ne(e,r=St(e)){return sz(r)||pW(e,r)}`
	const replacement = `function _Ne(e,r=St(e)){return sz(r)||dCt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_330(data []byte) bool {
	const original = `model:F(Fkl).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:n().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_330(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function na(){return xA()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function na(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function rCt(e){`, `function cNe(){`, `function rCt(e){return"Codex"}`),
		replaceClaude208Function(data, `function cNe(){`, `function MRr(e,r,o){`, `function cNe(){return"opus"}`),
		replaceFirstFixed(data, `function MRr(e,r,o){if(!na())return!1;return idt(e,o)&&(bt()||WU()||r)}`, `function MRr(e,r,o){return na()&&!!e}`),
		replaceClaude208Function(data, `function HRr(e){`, `function C7e(e,r){`, `function HRr(e){return na()&&(_e("flagSettings")?.fastMode===!0||C7e(vt()))}`),
		replaceClaude208Function(data, `function C7e(e,r){`, `function GCo(e,r,o){`, `function C7e(e,r){return e.fastMode===true}`),
		replaceClaude208Function(data, `function dS(e){let r=ve(),o=OA[r];`, `function jU(e,r){`, `function dS(e){return na()}`),
		replaceClaude208Function(data, `function jU(e,r){`, `function KV(e,r){`, `function jU(e,r){return na()&&(r!==void 0?!!r:C7e(vt()))}`),
		replaceFirstFixed(data, `k={model:r.model,...na()&&{fastMode:r.fastMode}}`, `k={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Jt.gates.fastModeEnabled&&{fastMode:A.options.fastMode}`, `fastMode:A.options.fastMode`),
		replaceFirstFixed(data, `toolFreeHelper:o.toolFreeHelper,...na()&&{fastMode:o.fastMode}`, `toolFreeHelper:o.toolFreeHelper,fastMode:o.fastMode`),
		replaceFirstFixed(data, `...na()&&{fastMode:dO}`, `fastMode:dO`),
		replaceFirstFixed(data, `...na()&&{fastMode:HRr(ve()==="firstParty"?De??null:I)}`, `fastMode:HRr(ve()==="firstParty"?De??null:I)`),
		replaceFirstFixed(data, `...na()?{fastMode:dO}:!1`, `fastMode:dO`),
		replaceFirstFixed(data, `GU=na()&&$e(()=>WU())&&!sCt()&&$e(()=>dS(yo))&&(ve()==="firstParty"||$e(()=>dS(Wn.model)))&&!!Wn.fastMode&&Vfn(yo)&&!ld(cAt)&&!LRr(Wn.model);`, `GU=!!Wn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_330(data []byte) bool {
	const label = `io=Pr;co=DA(Mo)??"Opus"`
	const subtitle = "const Jo=`High-speed mode for ${co}. ${c1e()?\"Draws from usage credits\":\"Billed\"} at a higher rate. Separate rate limits apply.`"
	if bytes.Count(data, []byte(label)) != 1 || bytes.Count(data, []byte(subtitle)) != 1 {
		return false
	}
	return replaceClaude208Function(data, "function jRr(e){", "function XCo(e){", `function jRr(e){return"Codex priority"}`) &&
		replaceFirstFixed(data, label, `io=Pr;co=jt(xo)`) &&
		replaceFirstFixed(data, subtitle, "const Jo=`High-speed mode for ${co}. Separate Codex rate limits apply.`")
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_330(data []byte) bool {
	const original = `function oso(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!$r(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!KC(l))return null;let f=n(l)?l:c(pe(),l),i=I6(f);if(i===null)return null;let o=i+u;return Sji(o)?null:o}`
	const replacement = `function oso(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||$r(r):!(r.startsWith("/")&&!$r(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!KC(l))return null;let f=I6(n(l)?l:c(pe(),l));if(f===null)return null;return Sji(f+=s)?null:f}`
	const originalMarkdown = `function je(e){let t=oso(e);if(t===null||Wz.test(t)||Ke.test(t)||t!==t.trimEnd())return null;if(Ye.test(t))return t;let n=kTr(t),r=n.some((i)=>/^file:/i.test(yji(i)));return zws(t,n)||r?null:t}`
	const replacementMarkdown = `function je(e){let t=oso(e,1);if(t===null||Wz.test(t)||Ke.test(t)||t!==t.trimEnd())return null;if(Ye.test(t))return t;let n=kTr(t),r=n.some(i=>/^file:/i.test(yji(i)));return zws(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_330(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=ES`, "export{MZs", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_330(data []byte) bool {
	const start = `function uqe(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{uqe};", `function uqe(e){return null}`)
}

func patchDefaultTierLabel_2_1_330(data []byte) bool {
	const target = "Default (recommended)"
	return bytes.Count(data, []byte(target)) == 4 && replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_330(data []byte) bool {
	const target = `var vYe="Claude Code";function CYe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `var vYe="Claudodex  ";function CYe(h){`)
}

func patchActiveFastModeBrand_2_1_330(data []byte) bool {
	// The native header derives its fast label from the model.
	return replaceClaude208Function(data, `function Ezt(){`, `function rCt(e){`, `function Ezt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_330(data []byte) bool {
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
	const startMarker = `function c$l({sessionId:e,worktree:o,columns:r,envAssignment:a,launchFlags:l="",launcherProgram:u}){`
	const endMarker = `var qn="anthropic-plugin-directory"`
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

func claude330SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_330(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_330},
		{"default-tier-label", patchDefaultTierLabel_2_1_330},
		{"whats-new", patchWhatsNewFeedFunction_2_1_330},
		{"usage", patchUsageFetchFunction_2_1_330},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_330(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_330},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_330},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_330},
		{"agent-model-validator", patchAgentModelValidator_2_1_330},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_330},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_330},
		{"fast-mode-pricing", patchFastModePricing_2_1_330},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_330},
		{"context-warning", patchContextWarningHint_2_1_330},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_330},
		{"resume-hints", patchResumeCommandHints_2_1_330},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude330RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude330UIBrandingReplacements)
		}},
	}
}

func claude330RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function CR(){return}function NUt(e=Fde(),n){`, `async function t1(e){if(!(ne()&&e!==void 0))`, `function CR(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN||void 0}function NUt(e=Fde(),n){return !CR()&&(KAe(e)||n!==void 0&&n!==tr()?.accessToken&&KAe(e,!0))}function V5(){return CR()??(jn()&&Rt()&&!P7()?Poe(tr())?.accessToken:void 0)}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function Yps(e){`, `var z="Remote Control is disabled by your organization's policy.`, `async function Yps(e){if(_je())return"not_signed_in";if(CG())return"managed_disabled";if(AG())return"cloud_session";if(iit()==="denied")return"org_policy_denied";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function SFn(){`, `function D(){if(Wy()||wqn()!==null)`, `async function SFn(){let e=bFn();if(e!==void 0)return l("not_signed_in",e);if(CG())return l("managed_disabled",Jae);if(iit()==="denied")return{reason:j(),code:"org_policy_denied",orgPolicyDenied:!0};if(AG())return l("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(mN()||Lyr()||oIi())return!0;`, `var r=`, `function e(){return!CG()&&!AG()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `isEnabled:e,get isHidden(){return!mN()},immediate:!0`, `isEnabled:e,get isHidden(){return!e()},immediate:!0`)
		}},
	}
}

func claude330Transformations(version string) []claude258Transformation {
	transformations := claude330SourceTransformationsForConfig(version, "2.1.330", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude330EmbeddedPatchedModuleBytecode})
}
