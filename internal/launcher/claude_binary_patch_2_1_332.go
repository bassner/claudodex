package launcher

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude332SHA = "5b8214f8f082f3187420b27b4063b9b4f4d9412cd66ea436d6d1966ad582700f"

var claudeUIPatch_2_1_332 = claudeUIPatchSpec{
	Version: "2.1.332",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude332SHA,
	Apply:   applyClaudeUIPatches_2_1_332,
}

var claude332UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude329UIBrandingReplacements...)
	for i := range replacements {
		if replacements[i].old == "Fix with Claude" {
			replacements[i].expectedCount = 4
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_332(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude332UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude281EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude332SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude332EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX332(")) && !bytes.Contains(entrySource, claude332RequiredLogoAnchor()) {
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

func disableClaude332EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude332EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude332RequiredLogoAnchor() []byte {
	return []byte("function c$t(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_332(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function c$t(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=uKo(),t=s.DEMO_VERSION?"/code/claude":Fn()&&(zt("tengu_violin_wood",!1)||zt(n6i,!1))?ys(Ee()):Lr(ue()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=Rt().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude332RequiredLogoAnchor()), "function rZs(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_332(data []byte) bool {
	const replacement = `var W=async(s,n)=>{return p("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",n.applyMessageOp,s),null};`
	return replaceClaude208Function(data, `var W=async(s,n)=>{try{await rn(fci(n.storageV5),500,"Timeout")}`, "function w(s){let a=k(19)", replacement)
}

func patchUsageFetchFunction_2_1_332(data []byte) bool {
	const anchor = `function We(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function ze(e,n,r,d){", "function we(e){return e===null?void 0:Ge.recall(e)}", `async function ze(e,n,o,u){return ss(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);u?.(Date.parse(d.headers.get("date")));return await d.json()},Le)}`)
}

func patchModelPickerOptions_2_1_332(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX332(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts332(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function Mft(e=!1,n=null){return CDXOpts332(e)}`
	return replaceClaude208Function(data, "function Mft(e=!1,n=null){", "function V_(e,n){let r=XI()", replacement)
}

func patchModelPickerSelectionValue_2_1_332(data []byte) bool {
	const replacement = `function eHl(e,n){if(n==null)return void 0;let r=CDX332(n),h=e.find(y=>y.value===r||CDX332(y.value)===r);return h?.value??r}`
	return replaceClaude208Function(data, "function eHl(e,n){", "function Of(){return!(dr()||{}).availableModels", replacement)
}

func patchModelPickerDefaultFocus_2_1_332(data []byte) bool {
	const original = `function DH(h,L){return L===null?CA:eHl(h,L)??L}`
	const replacement = `function DH(h,L){return eHl(h,L??ct())??L??CA}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_332(data []byte) bool {
	// The native picker already uses hHt to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function kSe(e,n=St(e)){return gV(n)||GB(e,n)}`
	const replacement = `function kSe(e,n=St(e)){return gV(n)||hHt(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_332(data []byte) bool {
	const original = `model:U(d6l).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_332(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function Aa(){return Ab()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function Aa(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function dHt(e){`, `function TBe(){`, `function dHt(e){return"Codex"}`),
		replaceClaude208Function(data, `function TBe(){`, `function DUo(e,n,r){`, `function TBe(){return"opus"}`),
		replaceFirstFixed(data, `function DUo(e,n,r){if(!Aa())return!1;return egt(e,r)&&(wt()||WW()||n)}`, `function DUo(e,n,r){return Aa()&&!!e}`),
		replaceClaude208Function(data, `function LUo(e){if(!Aa())`, `function Ott(e,n){`, `function LUo(e){return Aa()&&(he("flagSettings")?.fastMode===!0||Ott(Rt()))}`),
		replaceClaude208Function(data, `function Ott(e,n){`, `function tBr(e,n,r){`, `function Ott(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function dw(e){let n=be(),r=Eb[n];`, `function jW(e,n){`, `function dw(e){return Aa()}`),
		replaceClaude208Function(data, `function jW(e,n){`, `function g4(e,n){`, `function jW(e,n){return Aa()&&(n!==void 0?!!n:Ott(Rt()))}`),
		replaceFirstFixed(data, `k={model:r.model,...Aa()&&{fastMode:r.fastMode}}`, `k={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...lr.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `toolFreeHelper:n.toolFreeHelper,...Aa()&&{fastMode:n.fastMode}`, `toolFreeHelper:n.toolFreeHelper,fastMode:n.fastMode`),
		replaceFirstFixed(data, `...Aa()&&{fastMode:SI}`, `fastMode:SI`),
		replaceFirstFixed(data, `...Aa()&&{fastMode:LUo(be()==="firstParty"?we??null:Ae)}`, `fastMode:LUo(be()==="firstParty"?we??null:Ae)`),
		replaceFirstFixed(data, `...Aa()?{fastMode:SI}:!1`, `fastMode:SI`),
		replaceFirstFixed(data, `PK=Aa()&&Le(()=>WW())&&!uHt()&&Le(()=>dw(To))&&(be()==="firstParty"||Le(()=>dw(jn.model)))&&!!jn.fastMode&&uEn(To)&&!qd($4e)&&!FUo(jn.model);`, `PK=!!jn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_332(data []byte) bool {
	const label = `bo=ns;wo=fA(Io)??"Opus"`
	const subtitle = "const wn=`High-speed mode for ${wo}. ${gje()?\"Draws from usage credits\":\"Billed\"} at a higher rate. Separate rate limits apply.`"
	if bytes.Count(data, []byte(label)) != 1 || bytes.Count(data, []byte(subtitle)) != 1 {
		return false
	}
	return replaceClaude208Function(data, "function zUo(e){", "function aBr(e){", `function zUo(e){return"Codex priority"}`) &&
		replaceFirstFixed(data, label, `bo=ns;wo=ct()`) &&
		replaceFirstFixed(data, subtitle, "const wn=`High-speed mode for ${wo}. Separate Codex rate limits apply.`")
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_332(data []byte) bool {
	const original = `function qSr(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!Fo(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!Ok(l))return null;let f=n(l)?l:c(ue(),l),i=tq(f);if(i===null)return null;let o=i+u;return noa(o)?null:o}`
	const replacement = `function qSr(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||Fo(r):!(r.startsWith("/")&&!Fo(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!Ok(l))return null;let f=tq(n(l)?l:c(ue(),l));if(f===null)return null;return noa(f+=s)?null:f}`
	const originalMarkdown = `function je(e){let t=qSr(e);if(t===null||oK.test(t)||Fe.test(t)||t!==t.trimEnd())return null;if(Je.test(t))return t;let n=MNo(t),r=n.some((i)=>/^file:/i.test(toa(i)));return B$s(t,n)||r?null:t}`
	const replacementMarkdown = `function je(e){let t=qSr(e,1);if(t===null||oK.test(t)||Fe.test(t)||t!==t.trimEnd())return null;if(Je.test(t))return t;let n=MNo(t),r=n.some(i=>/^file:/i.test(toa(i)));return B$s(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_332(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=QS`, "export{zyi,Vyi,Kyi,qyi,nXa,sir}", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_332(data []byte) bool {
	const start = `function F8e(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{F8e};", `function F8e(e){return null}`)
}

func patchDefaultTierLabel_2_1_332(data []byte) bool {
	const target = "Default (recommended)"
	return bytes.Count(data, []byte(target)) == 4 && replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_332(data []byte) bool {
	const target = `var BJe="Claude Code";function HJe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `var BJe="Claudodex  ";function HJe(h){`)
}

func patchActiveFastModeBrand_2_1_332(data []byte) bool {
	// The native header derives its fast label from the model.
	return replaceClaude208Function(data, `function x8t(){`, `function dHt(e){`, `function x8t(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_332(data []byte) bool {
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
	const startMarker = `function jtd({sessionId:e,worktree:n,columns:r,envAssignment:a,launchFlags:l="",launcherProgram:c}){`
	const endMarker = `var ro="anthropic-plugin-directory"`
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

func claude332SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_332(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_332},
		{"default-tier-label", patchDefaultTierLabel_2_1_332},
		{"whats-new", patchWhatsNewFeedFunction_2_1_332},
		{"usage", patchUsageFetchFunction_2_1_332},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_332(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_332},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_332},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_332},
		{"agent-model-validator", patchAgentModelValidator_2_1_332},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_332},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_332},
		{"fast-mode-pricing", patchFastModePricing_2_1_332},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_332},
		{"context-warning", patchContextWarningHint_2_1_332},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_332},
		{"resume-hints", patchResumeCommandHints_2_1_332},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude332RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude332UIBrandingReplacements)
		}},
	}
}

func claude332RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function dO(){return}function fKt(e=Kpe(),n){`, `async function s1(e){if(!(ne()&&e!==void 0))`, `function dO(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN||void 0}function fKt(e=Kpe(),n){return !dO()&&(kIe(e)||n!==void 0&&n!==so()?.accessToken&&kIe(e,!0))}function bY(){return dO()??(Kn()&&It()&&!uG()?ile(so())?.accessToken:void 0)}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function gPs(e){`, `var z="Remote Control is disabled by your organization's policy.`, `async function gPs(e){if(LVe())return"not_signed_in";if(M6())return"managed_disabled";if(H6())return"cloud_session";if(gi())return"repl_mode";if(Aft()==="denied")return"org_policy_denied";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function $Vn(){let e=UVn();`, `function O(){if(Ly()||seo()!==null)`, `async function $Vn(){let e=UVn();if(e!==void 0)return l("not_signed_in",e);if(M6())return l("managed_disabled",Que);if(H6())return l("cloud_session","Remote Control is not available inside a cloud session.");if(gi())return l("repl_mode","Remote Control is unavailable in this mode.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");await R_e();let r=Aft();if(r==="unavailable")return l("policy_unverified",xPs);if(r==="denied")return{reason:j(),code:"org_policy_denied",orgPolicyDenied:!0};return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(u$()||jOo()||rqi())return!0;`, `var r={type:"local-jsx",name:"remote-control"`, `function e(){return!M6()&&!H6()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `isEnabled:e,get isHidden(){return!u$()},immediate:!0`, `isEnabled:e,get isHidden(){return!e()},immediate:!0`)
		}},
	}
}

func claude332Transformations(version string) []claude258Transformation {
	transformations := claude332SourceTransformationsForConfig(version, "2.1.332", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude332EmbeddedPatchedModuleBytecode})
}
