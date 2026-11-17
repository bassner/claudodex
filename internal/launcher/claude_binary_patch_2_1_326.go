package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude326SHA = "e214ad76dcc0e846b03be28ed1c2734aacd4e27acb527cb7f8a9e2bed644e837"

var claudeUIPatch_2_1_326 = claudeUIPatchSpec{
	Version: "2.1.326",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude326SHA,
	Apply:   applyClaudeUIPatches_2_1_326,
}

var claude326UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := make([]claude209UIBrandingReplacement, 0, len(claude325UIBrandingReplacements))
	for _, replacement := range claude325UIBrandingReplacements {
		switch replacement.old {
		// 2.1.326 removed these messages rather than renaming their symbols.
		case `[Image data detected and sent to Claude]`,
			`get pinged when Claude finishes \xB7 enable push notifications in`,
			`Sorry, Claude encountered an error`,
			`Working on a plan or design doc? Ask Claude to publish it as an artifact \u2014 a polished web page you can open in your browser.`,
			`Start with small features or bug fixes, tell Claude to propose a plan, and verify its suggested edits`,
			`Show Claude Code status including version, model, account, API connectivity, and tool statuses`,
			`Claude is now exploring and designing an implementation approach.`,
			`Hit Enter to queue up additional messages while Claude is working.`,
			`Send messages to Claude while it works to steer Claude in real-time`,
			`Claude can spawn copies of itself to work in parallel.`:
			continue
		case `Use /btw to ask a quick side question without interrupting Claude's current work`:
			replacement.old = `Use /btw to ask a side question without interrupting Claude`
			replacement.replacement = `Use /btw to ask a side question without interrupting Codex`
		case `Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`:
			replacement.old = `Switch between Claude models. Your pick becomes the default for new sessions. For other or older models, use --model.`
			replacement.replacement = `Switch between Codex models. Your pick becomes the default for new sessions. For direct model names, use --model.`
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_326(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude326UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude326EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude326SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude326EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude326EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX326(")) && !bytes.Contains(entrySource, claude326RequiredLogoAnchor()) {
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

func disableClaude326EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude326EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude326RequiredLogoAnchor() []byte {
	return []byte("function myt(){let i=s.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_326(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function myt(){let o=s.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,l=Rpr(),t=s.DEMO_VERSION?"/code/claude":Rn()&&(qt("tengu_violin_wood",!1)||qt(S9s,!1))?os(Te()):Vo(ue()),a=s.CLAUDE_CODE_HIDE_CWD?"":l?` + "`${t} in ${l.replace(/^https?:\\/\\//,\"\")}`" + `:t,m="Codex Plan",p=yt().agent;return{version:o,cwd:a,billingType:m,agentName:p}}`
	return replaceClaude208Function(data, string(claude326RequiredLogoAnchor()), "function Xcs(i,r,t,n){", replacement)
}

func patchWhatsNewFeedFunction_2_1_326(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_326(data []byte) bool {
	const anchor = `function ke(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"windows":return"/api/oauth/usage?skip_spend=1";case"cedar_ember":return`
	if bytes.Count(data, []byte(anchor)) != 1 {
		return false
	}
	start := bytes.Index(data, []byte(anchor))
	return replaceClaude208Function(data[start:], "async function Pe(e,n,r,a){", "function me(e){", `async function Pe(e,n,r,a){return To(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let b=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(b+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);a?.(Date.parse(d.headers.get("date")));return await d.json()},he)}`)
}

func patchModelPickerOptions_2_1_326(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX326(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,z=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(s.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(s.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(z))?"sonnet":(t===n(s.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(s.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts326(e=!1){let n=s,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??` + quoteJSString(modelCfg.Opus) + `),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??` + quoteJSString(modelCfg.Sonnet) + `),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??` + quoteJSString(modelCfg.Haiku) + `),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??` + quoteJSString(modelCfg.Fable) + `)]}function CYe(e=!1,n=null){return CDXOpts326(e)}`
	return replaceClaude208Function(data, "function CYe(e=!1,n=null){", "function TE(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_326(data []byte) bool {
	const replacement = `function iPa(e,n){if(n==null)return void 0;let r=CDX326(n),s=e.find((h)=>h.value===r||CDX326(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function iPa(e,n){", "function Am(){", replacement)
}

func patchModelPickerDefaultFocus_2_1_326(data []byte) bool {
	const original = `function kF(h,x){return x===null?mP:iPa(h,x)??x}`
	const replacement = `function kF(h,x){return iPa(h,x??at())??x??mP}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchConfiguredModelRecognition_2_1_326(data []byte) bool {
	// The native picker already uses n0t to recognize the four configured
	// ANTHROPIC_DEFAULT_*_MODEL targets. Its request recognizer instead calls a
	// stub returning false, so valid Codex targets emit an unknown-model warning.
	// Reuse native configured-target matching; unknown IDs still fail recognition.
	const original = `function FHe(e,n=dt(e)){return RB(n)||D1(e,n)}`
	const replacement = `function FHe(e,n=dt(e)){return RB(n)||n0t(e)}`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement)
}

func patchAgentModelValidator_2_1_326(data []byte) bool {
	const original = `model:B(uFa).optional()`
	if bytes.Count(data, []byte(original)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_326(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function gi(){return UA()&&!s.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function gi(){return!s.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function $ft(e){`, `function DRe(){`, `function $ft(e){return"Codex"}`),
		replaceClaude208Function(data, `function DRe(){`, `function vor(e,n,r){`, `function DRe(){return"opus"}`),
		replaceFirstFixed(data, `function vor(e,n,r){if(!gi())return!1;return gJe(e,r)&&(St()||NN()||n)}`, `function vor(e,n,r){return gi()&&!!e}`),
		replaceClaude208Function(data, `function Tor(e){`, `function ZIt(e,n){`, `function Tor(e){return gi()&&(ye("flagSettings")?.fastMode===!0||ZIt(yt()))}`),
		replaceClaude208Function(data, `function ZIt(e,n){`, `function PZr(e,n,r){`, `function ZIt(e,n){return e.fastMode===true}`),
		replaceClaude208Function(data, `function T_(e){let n=ve(),r=LA[n];`, `function $N(e,n){`, `function T_(e){return gi()}`),
		replaceClaude208Function(data, `function $N(e,n){`, `function TW(e,n){`, `function $N(e,n){return gi()&&(n!==void 0?!!n:ZIt(yt()))}`),
		replaceFirstFixed(data, `k={model:r.model,...gi()&&{fastMode:r.fastMode}}`, `k={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...Ct.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `...gi()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...gi()&&{fastMode:xP}`, `fastMode:xP`),
		replaceFirstFixed(data, `...gi()&&{fastMode:Tor(Oe??null)}`, `fastMode:Tor(Oe??null)`),
		replaceFirstFixed(data, `...gi()?{fastMode:xP}:!1`, `fastMode:xP`),
		replaceFirstFixed(data, `HP=gi()&&Pe(()=>NN())&&!Bft()&&Pe(()=>T_(so))&&(ve()==="firstParty"||Pe(()=>T_(Pn.model)))&&!!Pn.fastMode&&wYt(so)&&!Ua($pt)&&!Aor(Pn.model);`, `HP=!!Pn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_326(data []byte) bool {
	return replaceClaude208Function(data, "function Mor(e){", "function HZr(e){", `function Mor(e){return"Codex priority"}`)
}

// Only Markdown opts in to resolving scheme-less destinations. The shared
// normalizer retains its native behavior for plugin URL and allowlist callers.
func patchRelativeMarkdownFileLinks_2_1_326(data []byte) bool {
	const original = `function cBr(r){let e=/^file:/i.test(r);if(!e){if(!(r.startsWith("/")&&!Ar(r)))return r}let t=e?r.slice(5):r;if(t.startsWith("//")){if(t=t.slice(2),t==="localhost")t="/";else if(t.startsWith("localhost/"))t=t.slice(9)}let s=t.search(/[#?]/),u=s===-1?"":t.slice(s),l=s===-1?t:t.slice(0,s);if(l==="")return null;try{l=decodeURIComponent(l)}catch{}if(l=h(l),!Iv(l))return null;let f=n(l)?l:c(ue(),l),i=Nq(f);if(i===null)return null;let o=i+u;return pai(o)?null:o}`
	const replacement = `function cBr(r,t){let e=/^file:/i.test(r);if(!e&&(t?/^[a-z][a-z0-9+.-]*:|^[#?]|^$/i.test(r)||Ar(r):!(r.startsWith("/")&&!Ar(r))))return r;let o=e?r.slice(5):r;if(o.startsWith("//"))o=o.slice(2).replace(/^localhost(?:\/|(?![\s\S]))/,"/");let a=o.search(/[#?]/),s=a<0?"":o.slice(a),l=a<0?o:o.slice(0,a);if(!l)return null;try{l=decodeURIComponent(l)}catch{}l=h(l);if(!Iv(l))return null;let f=Nq(n(l)?l:c(ue(),l));if(f===null)return null;return pai(f+=s)?null:f}`
	const originalMarkdown = `function Ke(e){let t=cBr(e);if(t===null||F3.test(t)||qe.test(t)||t!==t.trimEnd())return null;if(rt.test(t))return t;let n=mai(t),r=n.some((i)=>/^file:/i.test(fai(i)));return xKo(t,n)||r?null:t}`
	const replacementMarkdown = `function Ke(e){let t=cBr(e,1);if(t===null||F3.test(t)||qe.test(t)||t!==t.trimEnd())return null;if(rt.test(t))return t;let n=mai(t),r=n.some(i=>/^file:/i.test(fai(i)));return xKo(t,n)||r?null:t}`
	if bytes.Count(data, []byte(original)) != 1 || bytes.Count(data, []byte(originalMarkdown)) != 1 {
		return false
	}
	return replaceFirstFixed(data, original, replacement) && replaceFirstFixed(data, originalMarkdown, replacementMarkdown)
}

func patchContextWarningHint_2_1_326(data []byte) bool {
	return replaceClaude208Function(data, `function M(e,o,n){let{source:r,window:d}=Pb`, "export{uxs", `function M(e,o,n){return null}`)
}

// Suppress Anthropic's Opus/Sonnet 1M upgrade hint in both the context
// footer and tips. Claudodex resolves context limits from the Codex catalog.
func patchContextUpgradeHint_2_1_326(data []byte) bool {
	const start = `function $$e(e){let t=n();if(!t)return null;switch(e){case"warning":return`
	if bytes.Count(data, []byte(start)) != 1 {
		return false
	}
	return replaceClaude208Function(data, start, "\nexport{$$e};", `function $$e(e){return null}`)
}

func patchActiveHeaderBrand_2_1_326(data []byte) bool {
	const target = `Claude Code";function yGe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function yGe(h){`)
}

func patchActiveFastModeBrand_2_1_326(data []byte) bool {
	// 2.1.326 derives its fast label from the model rather than a static title.
	return replaceClaude208Function(data, `function QIt(){`, `function $ft(e){`, `function QIt(){return"Codex"}`)
}

func patchResumeCommandHints_2_1_326(data []byte) bool {
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
	const startMarker = `function b9a({sessionId:e,worktree:n,columns:r,envAssignment:a,launchFlags:l="",launcherProgram:d}){`
	const endMarker = `var Nn="anthropic-plugin-directory"`
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

func claude326SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_326(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_326},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_326},
		{"usage", patchUsageFetchFunction_2_1_326},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_326(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_326},
		{"model-default-focus", patchModelPickerDefaultFocus_2_1_326},
		{"configured-model-recognition", patchConfiguredModelRecognition_2_1_326},
		{"agent-model-validator", patchAgentModelValidator_2_1_326},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_326},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_326},
		{"fast-mode-pricing", patchFastModePricing_2_1_326},
		{"relative-markdown-file-links", patchRelativeMarkdownFileLinks_2_1_326},
		{"context-warning", patchContextWarningHint_2_1_326},
		{"context-upgrade-hint", patchContextUpgradeHint_2_1_326},
		{"resume-hints", patchResumeCommandHints_2_1_326},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude326RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude326UIBrandingReplacements)
		}},
	}
}

func claude326RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function lC(){return}`, `function s4(){`, `function lC(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function gRt(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function YWo(e){`, `var Y=`, `async function YWo(e){if(KU())return"managed_disabled";if(hG())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function wSn(){`, `function Z(){`, `async function wSn(){if(KU())return l("managed_disabled",wte);if(hG())return l("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return l("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(B1()||U7n()||w7s())return!0;`, `var r=`, `function e(){return!KU()&&!hG()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!B1()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude326Transformations(version string) []claude258Transformation {
	transformations := claude326SourceTransformationsForConfig(version, "2.1.326", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude326EmbeddedPatchedModuleBytecode})
}
