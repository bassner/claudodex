package launcher

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude331SHA = "4874591dc39bff1188921b82cbeb8a44e3a33b0f92c888b3d43639b9344101f7"

var claudeUIPatch_2_1_331 = claudeUIPatchSpec{
	Version: "2.1.331",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude331SHA,
	Apply:   applyClaudeUIPatches_2_1_331,
}

var claude331UIBrandingReplacements = append([]claude209UIBrandingReplacement(nil), claude329UIBrandingReplacements...)

func applyClaudeUIPatches_2_1_331(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude331UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude331SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude331EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX331(")) && !bytes.Contains(entrySource, claude331RequiredLogoAnchor()) {
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

func disableClaude331EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude331EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude331RequiredLogoAnchor() []byte {
	return []byte("function WLt(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_331(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function WLt(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=s2o(),t=s.DEMO_VERSION?"/code/claude":Ln()&&(Ft("tengu_violin_wood",!1)||Ft(wUi,!1))?ls(ve()):Lr(ue()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=Tt().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude331RequiredLogoAnchor()), "function z5s(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_331(data []byte) bool {
	const replacement = `var W=async(s,n)=>{return p("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,s),null};`
	return replaceClaude208Function(data, `var W=async(s,n)=>{try{await tn(Uni(n.storageV5),500,"Timeout")}`, "function L(s){", replacement)
}

func patchUsageFetchFunction_2_1_331(data []byte) bool {
	const anchor = `function De(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Ne(e,n,r,d){", "function ye(e){return e===null?void 0:Fe.recall(e)}", `async function Ne(e,n,o,u){return os(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);u?.(Date.parse(d.headers.get("date")));return await d.json()},Ae)}`)
}

func patchModelPickerOptions_2_1_331(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX331(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts331(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function Gdt(e=!1,n=null){return CDXOpts331(e)}`
	return replaceClaude208Function(data, "function Gdt(e=!1,n=null){", "function sw(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_331(data []byte) bool {
	const replacement = `function qTl(e,n){if(n==null)return void 0;let r=CDX331(n),h=e.find(y=>y.value===r||CDX331(y.value)===r);return h?.value??r}`
	return replaceClaude208Function(data, "function qTl(e,n){", "function ym(){return!(rr()||{}).availableModels", replacement)
}

func patchModelPickerDefaultFocus_2_1_331(data []byte) bool {
	const original = `function LB(h,E){return E===null?lA:qTl(h,E)??E}`
	const replacement = `function LB(h,E){return qTl(h,E??ct())??E??lA}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_331(data []byte) bool {
	// The native picker already uses XPt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function A_e(e,n=St(e)){return h6(n)||SB(e,n)}`
	const replacement = `function A_e(e,n=St(e)){return h6(n)||XPt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_331(data []byte) bool {
	const original = `model:U(kFl).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_331(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Sa(){return Tb()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Sa(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function GPt(e){`, `function cUe(){`, `function GPt(e){return"Codex"}`),
		replaceClaude208Function(data, `function cUe(){`, `function xLo(e,n,r){`, `function cUe(){return"opus"}`),
		replaceFirstFixed(data, `function xLo(e,n,r){if(!Sa())return!1;return apt(e,r)&&(kt()||BB()||n)}`, `function xLo(e,n,r){return Sa()&&!!e}`),
		replaceClaude208Function(data, `function PLo(e){if(!Sa())`, `function jQe(e,n){`, `function PLo(e){return Sa()&&(he("flagSettings")?.fastMode===!0||jQe(Tt()))}`),
		replaceClaude208Function(data, `function jQe(e,n){`, `function hLr(e,n,r){`, `function jQe(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function WS(e){let n=be(),r=wb[n];`, `function WB(e,n){`, `function WS(e){return Sa()}`),
		replaceClaude208Function(data, `function WB(e,n){`, `function pq(e,n){`, `function WB(e,n){return Sa()&&(n!==void 0?!!n:jQe(Tt()))}`),
		replaceFirstFixed(data, `k={model:r.model,...Sa()&&{fastMode:r.fastMode}}`, `k={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Vn.gates.fastModeEnabled&&{fastMode:b.options.fastMode}`, `fastMode:b.options.fastMode`),
		replaceFirstFixed(data, `toolFreeHelper:n.toolFreeHelper,...Sa()&&{fastMode:n.fastMode}`, `toolFreeHelper:n.toolFreeHelper,fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Sa()&&{fastMode:PM}`, `fastMode:PM`),
		replaceFirstFixed(data, `...Sa()&&{fastMode:PLo(be()==="firstParty"?Ue??null:$e)}`, `fastMode:PLo(be()==="firstParty"?Ue??null:$e)`),
		replaceFirstFixed(data, `...Sa()?{fastMode:PM}:!1`, `fastMode:PM`),
		replaceFirstFixed(data, `PK=Sa()&&je(()=>BB())&&!VPt()&&je(()=>WS(co))&&(be()==="firstParty"||je(()=>WS(Rn.model)))&&!!Rn.fastMode&&wSn(co)&&!Pd(ZOt)&&!ILo(Rn.model);`, `PK=!!Rn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_331(data []byte) bool {
	const label = `vo=Vr;wo=FR(No)??"Opus"`
	const subtitle = "const on=`High-speed mode for ${wo}. ${Z1e()?\"Draws from usage credits\":\"Billed\"} at a higher rate. Separate rate limits apply.`"
	if bytes.Count(data, []byte(label)) != 1 || bytes.Count(data, []byte(subtitle)) != 1 {
		return false
	}
	return replaceClaude208Function(data, "function $Lo(e){", "function bLr(e){", `function $Lo(e){return"Codex priority"}`) &&
		replaceFirstFixed(data, label, `vo=Vr;wo=ct()`) &&
		replaceFirstFixed(data, subtitle, "const on=`High-speed mode for ${wo}. Separate Codex rate limits apply.`")
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_331(data []byte) bool {
	const original = `function Lpr(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!Uo(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!WE(l))return null;let f=n(l)?l:c(ue(),l),i=ZV(f);if(i===null)return null;let o=i+u;return HYi(o)?null:o}`
	const replacement = `function Lpr(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||Uo(r):!(r.startsWith("/")&&!Uo(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!WE(l))return null;let f=ZV(n(l)?l:c(ue(),l));if(f===null)return null;return HYi(f+=s)?null:f}`
	const originalMarkdown = `function je(e){let t=Lpr(e);if(t===null||oV.test(t)||Fe.test(t)||t!==t.trimEnd())return null;if(Je.test(t))return t;let n=MMo(t),r=n.some((i)=>/^file:/i.test(DYi(i)));return $Ms(t,n)||r?null:t}`
	const replacementMarkdown = `function je(e){let t=Lpr(e,1);if(t===null||oV.test(t)||Fe.test(t)||t!==t.trimEnd())return null;if(Je.test(t))return t;let n=MMo(t),r=n.some(i=>/^file:/i.test(DYi(i)));return $Ms(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_331(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=iw`, "export{Jci,Qci,Zci,eui,vKa,ger}", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_331(data []byte) bool {
	const start = `function i5e(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{i5e};", `function i5e(e){return null}`)
}

func patchDefaultTierLabel_2_1_331(data []byte) bool {
	const target = "Default (recommended)"
	return bytes.Count(data, []byte(target)) == 4 && replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_331(data []byte) bool {
	const target = `var c2e="Claude Code";function u2e(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `var c2e="Claudodex  ";function u2e(h){`)
}

func patchActiveFastModeBrand_2_1_331(data []byte) bool {
	// The native header derives its fast label from the model.
	return replaceClaude208Function(data, `function Q4t(){`, `function GPt(e){`, `function Q4t(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_331(data []byte) bool {
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
	const startMarker = `function V5l({sessionId:e,worktree:n,columns:r,envAssignment:a,launchFlags:l="",launcherProgram:c}){`
	const endMarker = `var Qn="anthropic-plugin-directory"`
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let f=a===void 0?\"claude\":`${a} claude`"
	const newCommand = `let f=(a===void 0?"":a+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude331SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_331(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_331},
		{"default-tier-label", patchDefaultTierLabel_2_1_331},
		{"whats-new", patchWhatsNewFeedFunction_2_1_331},
		{"usage", patchUsageFetchFunction_2_1_331},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_331(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_331},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_331},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_331},
		{"agent-model-validator", patchAgentModelValidator_2_1_331},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_331},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_331},
		{"fast-mode-pricing", patchFastModePricing_2_1_331},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_331},
		{"context-warning", patchContextWarningHint_2_1_331},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_331},
		{"resume-hints", patchResumeCommandHints_2_1_331},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude331RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude331UIBrandingReplacements)
		}},
	}
}

func claude331RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function Hx(){return}function XGt(e=lfe(),n){`, `async function sU(e){if(!(ne()&&e!==void 0))`, `function Hx(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN||void 0}function XGt(e=lfe(),n){return !Hx()&&(APe(e)||n!==void 0&&n!==no()?.accessToken&&APe(e,!0))}function S8(){return Hx()??(qn()&&Pt()&&!ZX()?Rie(no())?.accessToken:void 0)}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function hCs(e){`, `var z="Remote Control is disabled by your organization's policy.`, `async function hCs(e){if(Eze())return"not_signed_in";if(Lz())return"managed_disabled";if(Nz())return"cloud_session";if(_i())return"repl_mode";if(fct()==="denied")return"org_policy_denied";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function hGn(){let e=gGn();`, `function O(){if(ay()||E7n()!==null)`, `async function hGn(){let e=gGn();if(e!==void 0)return l("not_signed_in",e);if(Lz())return l("managed_disabled",vce);if(Nz())return l("cloud_session","Remote Control is not available inside a cloud session.");if(_i())return l("repl_mode","Remote Control is unavailable in this mode.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");await Jhe();let r=fct();if(r==="unavailable")return l("policy_unverified",xCs);if(r==="denied")return{reason:j(),code:"org_policy_denied",orgPolicyDenied:!0};return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(_F()||xAo()||M2i())return!0;`, `var r={type:"local-jsx",name:"remote-control"`, `function e(){return!Lz()&&!Nz()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `isEnabled:e,get isHidden(){return!_F()},immediate:!0`, `isEnabled:e,get isHidden(){return!e()},immediate:!0`)
		}},
	}
}

func claude331Transformations(version string) []claude258Transformation {
	transformations := claude331SourceTransformationsForConfig(version, "2.1.331", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude331EmbeddedPatchedModuleBytecode})
}
