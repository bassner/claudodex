package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude314SHA = "21d6e7b9a6f07b4cfcb7df13760b3b72aa25e084e13a5354413382987ba832ed"

var claudeUIPatch_2_1_314 = claudeUIPatchSpec{
	Version: "2.1.314",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  claude314SHA,
	Apply:   applyClaudeUIPatches_2_1_314,
}

var claude314UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude296UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "What should Claude do instead?" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

func applyClaudeUIPatches_2_1_314(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude314UIBrandingReplacements) {
		return false
	}
	records, hashes, ok := claude314EmbeddedBunModuleHashes(data)
	if !ok {
		return false
	}
	for _, transformation := range claude314SourceTransformationsForConfig(claudodexVersion, claudeVersion, modelCfg) {
		if !transformation.apply(data) {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return disableClaude281ChangedEmbeddedModuleBytecode(data, records, hashes)
}

func claude314EmbeddedBunModuleHashes(data []byte) ([]claude259BunModuleRecord, [][sha256.Size]byte, bool) {
	return claude281EmbeddedBunModuleHashes(data)
}

func claude314EmbeddedPatchedModuleRecord(data []byte) (claude259BunModuleRecord, bool) {
	records, ok := claude281EmbeddedBunModuleRecords(data)
	if !ok {
		return claude259BunModuleRecord{}, false
	}
	var found *claude259BunModuleRecord
	for _, record := range records {
		entrySource := data[record.contentOffset : record.contentOffset+record.contentLength]
		if !bytes.Contains(entrySource, []byte("function CDX314(")) && !bytes.Contains(entrySource, claude314RequiredLogoAnchor()) {
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

func disableClaude314EmbeddedPatchedModuleBytecode(data []byte) bool {
	record, ok := claude314EmbeddedPatchedModuleRecord(data)
	if !ok || binary.LittleEndian.Uint32(data[record.bytecodeLength:record.bytecodeLength+4]) == 0 {
		return false
	}
	binary.LittleEndian.PutUint32(data[record.bytecodeOffset:record.bytecodeOffset+4], 0)
	binary.LittleEndian.PutUint32(data[record.bytecodeLength:record.bytecodeLength+4], 0)
	return true
}

func claude314RequiredLogoAnchor() []byte {
	return []byte("function HXe(){let o=i.DEMO_VERSION??")
}

func patchLogoDisplayDataFunction_2_1_314(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function HXe(){let o=i.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,a=e_s(),t=i.DEMO_VERSION?"/code/claude":On()&&(An("tengu_violin_wood",!1)||An(N4o,!1))?xs(ve()):Cs(pe()),l=i.CLAUDE_CODE_HIDE_CWD?"":a?` + "`${t} in ${a.replace(/^https?:\\/\\//,\"\")}`" + `:t,c="Codex Plan",m=pt().agent;return{version:o,cwd:l,billingType:c,agentName:m}}`
	return replaceClaude208Function(data, string(claude314RequiredLogoAnchor()), "function lho(o,a,t,l){", replacement)
}

func patchWhatsNewFeedFunction_2_1_314(data []byte) bool {
	const replacement = `var te=async(i,t)=>{return c("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,i),null};`
	return replaceClaude208Function(data, "var te=async(i,t)=>{try{", "function X(i){", replacement)
}

func patchUsageFetchFunction_2_1_314(data []byte) bool {
	const anchor = `function ce(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return`
	const startMarker = "async function fe(e,n,r){"
	const endMarker = "function q(e){"
	const replacement = `async function fe(e,n,r){return qr(n==="at_wall"?"api_usage_fetch_at_wall":n==="cedar_ember"?"api_usage_fetch_cedar_ember":n==="starry_waterfall"?"api_usage_fetch_starry_waterfall":"api_usage_fetch",async()=>{let s=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),d=await fetch(s+r,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()},J)}`
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

func patchModelPickerOptions_2_1_314(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX314(e){let n=(r)=>String(r??"").replaceAll("[1m]","").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `,f=` + quoteJSString(modelCfg.Fable) + `;return(t===n(i.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(i.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(i.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":(t===n(i.ANTHROPIC_DEFAULT_FABLE_MODEL)||t===n(f))?"fable":e}function CDXOpts314(e=!1){let n=i,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus","Opus",n.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??n.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-6.1-sol"),r("sonnet","Sonnet",n.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??n.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-6.1-terra"),r("haiku","Haiku",n.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??n.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-6-luna"),r("fable","Fable",n.ANTHROPIC_DEFAULT_FABLE_MODEL_NAME??n.ANTHROPIC_DEFAULT_FABLE_MODEL??"gpt-6-astra")]}function FFe(e=!1,n=null){return CDXOpts314(e)}`
	return replaceClaude208Function(data, "function FFe(e=!1,n=null){", "function X_(e,n){", replacement)
}

func patchModelPickerSelectionValue_2_1_314(data []byte) bool {
	const replacement = `function Jor(e,n){if(n==null)return void 0;let r=CDX314(n),s=e.find((h)=>h.value===r||CDX314(h.value)===r);return s?.value??r}`
	return replaceClaude208Function(data, "function Jor(e,n){", "function kc(){", replacement)
}

func patchAgentModelValidator_2_1_314(data []byte) bool {
	return replaceFirstFixed(data, `model:F(dmi).optional()`, `model:o().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_314(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function gs(){if(Te()!=="firstParty")return!1;return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function gs(){return!i.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceClaude208Function(data, `function Pq(){`, `function Xwe(){`, `function Pq(){return"Codex"}`),
		replaceFirstFixed(data, `function Xwe(){return"opus"+(I9()?"[1m]":"")}`, `function Xwe(){return"opus"}`),
		replaceFirstFixed(data, `function WTn(e,n,r){if(!gs())return!1;return EUe(e,r)&&(vt()||mI()||n)}`, `function WTn(e,n,r){return gs()&&!!e}`),
		replaceFirstFixed(data, `function GTn(e){if(!gs())return!1;if(!mI(e))return!1;if(!py(e))return!1;return Eat(pt())}`, `function GTn(e){return gs()&&(Se("flagSettings")?.fastMode===!0||Eat(pt()))}`),
		replaceFirstFixed(data, `function Eat(e,n){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(Se("policySettings")?.fastModePerSessionOptIn===!0)return!1;return(n?n.flagFastMode:Se("flagSettings")?.fastMode)===!0}`, `function Eat(e,n){return e.fastMode===true}`),
		replaceFirstFixed(data, `function py(e){if(!gs())return!1;let n=e??Mw(),r=Ut(n),s=nw(Qe(r),"fast_mode",r);if(s!==void 0)return s;let h=r.toLowerCase();return h.includes("opus-4-8")||h.includes("opus-5")}`, `function py(e){return gs()}`),
		replaceFirstFixed(data, `function m0(e,n){if(vt()){if(e===null)return!!n;return!!n&&py(e)}if(!py(e))return!1;return!!n||GTn(e)}`, `function m0(e,n){return gs()&&(n!==void 0?!!n:Eat(pt()))}`),
		replaceFirstFixed(data, `b={model:r.model,...gs()&&{fastMode:r.fastMode}}`, `b={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...pt.gates.fastModeEnabled&&{fastMode:k.options.fastMode}`, `fastMode:k.options.fastMode`),
		replaceFirstFixed(data, `...gs()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
		replaceFirstFixed(data, `...gs()&&{fastMode:GTn(le??null)}`, `fastMode:GTn(le??null)`),
		replaceFirstFixed(data, `...gs()?{fastMode:OA}:!1`, `fastMode:OA`),
		replaceFirstFixed(data, `Lw=gs()&&be(()=>mI())&&!q4e()&&be(()=>py(Jt))&&!!Wn.fastMode&&!Vr(pPt)&&!VTn(Wn.model);`, `Lw=!!Wn.fastMode;`),
	}
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchFastModePricing_2_1_314(data []byte) bool {
	return replaceClaude208Function(data, "function CUe(e){", "function mbi(e){", `function CUe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_314(data []byte) bool {
	return replaceClaude208Function(data, `function u(o,e,n){let{source:d,window:r}=Db`, "export{ZSo", `function u(o,e,n){return null}`)
}

func patchActiveHeaderBrand_2_1_314(data []byte) bool {
	const target = `Claude Code";function THe(h){`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `Claudodex  ";function THe(h){`)
}

func patchActiveFastModeBrand_2_1_314(data []byte) bool {
	const target = `"Opus 5.5",nr("claude",yo)("Opus 5.5")`
	if bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, `"Codex+  ",nr("claude",yo)("Codex+  ")`)
}

func patchResumeCommandHints_2_1_314(data []byte) bool {
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
	const startMarker = "function dRi({sessionId:e,worktree:n,columns:r,envAssignment:s}){"
	const endMarker = "import{sep as tf}"
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return false
	}
	end := bytes.Index(data[start:], []byte(endMarker))
	if end < 0 {
		return false
	}
	old := string(data[start : start+end])
	const oldCommand = "let a=s===void 0?\"claude\":`${s} claude`"
	const newCommand = `let a=(s===void 0?"":s+" ")+"claudodex"`
	if strings.Count(old, oldCommand) != 1 {
		return false
	}
	return replaceFirstFixed(data, old, strings.Replace(old, oldCommand, newCommand, 1))
}

func claude314SourceTransformationsForConfig(claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool {
			return patchLogoDisplayDataFunction_2_1_314(data, claudodexVersion, claudeVersion)
		}},
		{"active-header-brand", patchActiveHeaderBrand_2_1_314},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_314},
		{"usage", patchUsageFetchFunction_2_1_314},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_314(data, modelCfg) }},
		{"model-selection", patchModelPickerSelectionValue_2_1_314},
		{"agent-model-validator", patchAgentModelValidator_2_1_314},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_314},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_314},
		{"fast-mode-pricing", patchFastModePricing_2_1_314},
		{"context-warning", patchContextWarningHint_2_1_314},
		{"resume-hints", patchResumeCommandHints_2_1_314},
		{"remote-control", func(data []byte) bool {
			for _, transformation := range claude314RemoteControlTransformations() {
				if !transformation.apply(data) {
					return false
				}
			}
			return true
		}},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude314UIBrandingReplacements)
		}},
	}
}

func claude314RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, `function AT(){return}`, `function nB(){`, `function AT(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function Fot(e,n){return!1}`)
		}},
		{"available", func(data []byte) bool {
			return replaceClaude208Function(data, `async function F5r(e){`, `var K=`, `async function F5r(e){if(KK())return"managed_disabled";if(j$())return"cloud_session";return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN?null:"not_signed_in"}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `async function _Kt(){`, `async function Q(){`, `async function _Kt(){if(KK())return s("managed_disabled",Cie);if(j$())return s("cloud_session","Remote Control is not available inside a cloud session.");if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return s("not_signed_in","Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.");return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceClaude208Function(data, `function e(){if(dD()||z9o())return!0;`, `var r=`, `function e(){return!KK()&&!j$()}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!dD()}`, `get isHidden(){return!e()}`)
		}},
	}
}

func claude314Transformations(version string) []claude258Transformation {
	transformations := claude314SourceTransformationsForConfig(version, "2.1.314", modelconfig.Default())
	return append(transformations, claude258Transformation{"patched-module-bytecode", disableClaude314EmbeddedPatchedModuleBytecode})
}
