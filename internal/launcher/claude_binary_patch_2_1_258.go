package launcher

import (
	"bytes"
	"strings"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_258 = claudeUIPatchSpec{
	Version: "2.1.258",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "b63136194160791c27cfa7b0403060d85eb0752991625fde8c09f9acacb17c78",
	Apply:   applyClaudeUIPatches_2_1_258,
}

const (
	claude258ActiveHeaderBrandTarget    = "\x0b\x00\x00\x80\xc9/\x86\x00Claude Code\x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle"
	claude258ActiveFastModeBrandTarget  = "\x09\x00\x00\x80\x63\xc9\xdd\x00Fable 5.1\x00\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	claude258ModelOptionsOverrideTarget = `tt=z(()=>Re??Xun(lt),[Re,lt])`
	claude258ModelSelectionSourceTarget = `Et=Ye??T,ft=Et===null?ww:$ft(tt,Et)??Et,`
	claude258ModelExtraOptionsTarget    = `po=z(()=>{let qi=[];for(let[ii,Ni,Qs]of[[to.current,to.value,"Current model"],[to.sessionOverride===null?null:T,T===null?ww:$ft(tt,T)??T,"Base model"]])if(ii!==null&&!tt.some((Ms)=>Ms.value===Ni)&&!qi.some((Ms)=>Ms.value===ii)&&Mr(ii))qi.push({value:ii,label:cv(ii),description:Qs});if(qi.length===0)return tt;let ms=tt.findIndex((ii)=>ii.disabled===!0);if(ms===-1)return[...tt,...qi];return[...tt.slice(0,ms),...qi,...tt.slice(ms)]},[tt,to,T])`
	claude258ModelPickerValueTarget     = `defaultValue:ft,selectedValue:ft,defaultFocusValue:Fo,options:kn,`
)

var claude258UIBrandingReplacements = func() []claude209UIBrandingReplacement {
	replacements := append([]claude209UIBrandingReplacement(nil), claude251UIBrandingReplacements...)
	for index := range replacements {
		if replacements[index].old == "Claude Code needs your input" {
			replacements[index].expectedCount = 2
		}
	}
	return replacements
}()

type claude258Transformation struct {
	name  string
	apply func([]byte) bool
}

func applyClaudeUIPatches_2_1_258(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude258UIBrandingReplacements) {
		return false
	}
	transformations := []bool{
		patchLogoDisplayDataFunction_2_1_258(data, claudodexVersion, claudeVersion),
		patchActiveHeaderBrand_2_1_258(data),
		patchDefaultTierLabel_2_1_258(data),
		patchWhatsNewFeedFunction_2_1_258(data),
		patchUsageFetchFunction_2_1_258(data),
		patchModelPickerOptions_2_1_258(data, modelCfg),
		patchModelPickerResolver_2_1_258(data),
		patchModelPickerExtraOptions_2_1_258(data),
		patchModelPickerSelectionValue_2_1_258(data),
		patchAgentModelValidator_2_1_258(data),
		patchFastModeRuntimeFunctions_2_1_258(data),
		patchActiveFastModeBrand_2_1_258(data),
		patchFastModePricing_2_1_258(data),
		patchContextWarningHint_2_1_258(data),
		patchResumeCommandHints_2_1_258(data),
		patchCompactProgressCurve_2_1_258(data),
		patchRemoteControlRuntimeFunctions_2_1_258(data),
		applyClaude209UIBrandingReplacements(data, claude258UIBrandingReplacements),
	}
	for _, transformed := range transformations {
		if !transformed {
			return false
		}
	}
	applyClaudeUIFixedReplacements_2_1_208(data, modelCfg)
	return true
}

func patchLogoDisplayDataFunction_2_1_258(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function fHe(){let l=a.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,b=GIn(),O=a.DEMO_VERSION?"/code/claude":Io(ne()),T=a.CLAUDE_CODE_HIDE_CWD?"":b?` + "`${O} in ${b.replace(/^https?:\\/\\//,\"\")}`" + `:O,w="Codex Plan",S=Je().agent;return{version:l,cwd:T,billingType:w,agentName:S}}`
	return replaceClaude208Function(data, "function fHe(){let l=a.DEMO_VERSION??", "function DJt(l,b,O){", replacement)
}

func patchDefaultTierLabel_2_1_258(data []byte) bool {
	const target = "Default (recommended)"
	if bytes.Count(data, []byte(target)) != 4 {
		return false
	}
	return replaceAllFixed(data, target, "Sonnet")
}

func patchActiveHeaderBrand_2_1_258(data []byte) bool {
	const target = "Claude Code\x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle"
	if bytes.Count(data, []byte(claude258ActiveHeaderBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Claudodex  \x00\x0e\x00\x00\x80I*\x8e\x00aiSessionTitle")
}

func patchWhatsNewFeedFunction_2_1_258(data []byte) bool {
	const replacement = `var ee=async(s,t)=>{return y("Claudodex Info\nThank you for using Claudodex!\nExperimental - treat it as such.\nhttps://github.com/bassner/claudodex/issues",t.applyMessageOp,s),null};`
	return replaceClaude208Function(data, "var ee=async(s,t)=>{try{", "function D(L){", replacement)
}

func patchUsageFetchFunction_2_1_258(data []byte) bool {
	const replacement = `async function BO(e,{atWall:n=!1}={}){return kr(n?"api_usage_fetch_at_wall":"api_usage_fetch",async()=>{let r=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),o=n?"/api/oauth/usage?at_wall=1&skip_spend=1":"/api/oauth/usage",d=await fetch(r+o,{headers:{"Content-Type":"application/json"}});if(!d.ok)throw Error("Auth error: "+d.status);return await d.json()})}`
	return replaceClaude208Function(data, "async function BO(e,{atWall:n=!1}={}){", "var Tze=", replacement)
}

func patchModelPickerOptions_2_1_258(data []byte, modelCfg modelconfig.Config) bool {
	modelCfg = modelCfg.Normalize()
	replacement := `function CDX258(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e),o=` + quoteJSString(modelCfg.Opus) + `,s=` + quoteJSString(modelCfg.Sonnet) + `,h=` + quoteJSString(modelCfg.Haiku) + `;return(t===n(a.ANTHROPIC_DEFAULT_OPUS_MODEL)||t===n(o))?"opus":(t===n(a.ANTHROPIC_DEFAULT_SONNET_MODEL)||t===n(s))?"sonnet":(t===n(a.ANTHROPIC_DEFAULT_HAIKU_MODEL)||t===n(h))?"haiku":e}function Xun(e=!1){return fTe(e)}function fTe(e=!1,n=null){let t=a,r=(v,l,d)=>({value:v,label:l,description:d,descriptionForModel:d});return[r("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),r("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),r("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	replacement = strings.Replace(replacement, `.replace(/(\[1m\])+$/i,"")`, `.replaceAll("[1m]","")`, 1)
	return replaceClaude208Function(data, "function Xun(e=!1){", "function X9r(e,n){", replacement)
}

func patchModelPickerResolver_2_1_258(data []byte) bool {
	return replaceClaude208Function(data, "function eXr(e,n){", "function Yun(e){", `function eXr(e,n){return fTe(e).slice(0,3)}`)
}

func patchModelPickerExtraOptions_2_1_258(data []byte) bool {
	if bytes.Count(data, []byte(claude258ModelOptionsOverrideTarget)) != 1 ||
		bytes.Count(data, []byte(claude258ModelSelectionSourceTarget)) != 1 ||
		bytes.Count(data, []byte(claude258ModelExtraOptionsTarget)) != 1 ||
		bytes.Count(data, []byte(claude258ModelPickerValueTarget)) != 1 {
		return false
	}
	return replaceFirstFixed(data, claude258ModelOptionsOverrideTarget, `tt=z(()=>Xun(lt),[lt])`) &&
		replaceFirstFixed(data, claude258ModelSelectionSourceTarget, `Et=Ye??T,ft=CDX258(Et===null?ww:Et),`) &&
		replaceFirstFixed(data, claude258ModelExtraOptionsTarget, "po=tt.slice(0,3)") &&
		replaceFirstFixed(data, claude258ModelPickerValueTarget, `selectedValue:CDX258(ft),options:kn.slice(0,3),`)
}

func patchModelPickerSelectionValue_2_1_258(data []byte) bool {
	const replacement = `function $ft(e,n){let r=CDX258(n),o=e.find((d)=>d.value===r||CDX258(d.value)===r);return o?.value??r}`
	return replaceClaude208Function(data, "function $ft(e,n){if(e.some((d)=>d.value===n))return n;", "function wen(){", replacement)
}

func patchAgentModelValidator_2_1_258(data []byte) bool {
	return replaceFirstFixed(data, `model:ee(["sonnet","opus","haiku","fable"]).optional()`, `model:i().optional()`)
}

func patchFastModeRuntimeFunctions_2_1_258(data []byte) bool {
	checks := []bool{
		replaceFirstFixed(data, `function $r(){if(Me()!=="firstParty")return!1;return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`, `function $r(){return!a.CLAUDE_CODE_DISABLE_FAST_MODE}`),
		replaceFirstFixed(data, `function CR(){return"Opus 5"}`, `function CR(){return"Codex"}`),
		replaceFirstFixed(data, `function OFe(){return"opus"+(HT()?"[1m]":"")}`, `function OFe(){return"opus"}`),
		replaceFirstFixed(data, `function xEn(e,n){if(!$r())return!1;return!!e&&(Zt()||cS()||n)}`, `function xEn(e,n){return $r()&&!!e}`),
		replaceFirstFixed(data, `function Kwt(e){if(!$r())return!1;if(!cS(e))return!1;if(!ff(e))return!1;return HEn(Je())}`, `function Kwt(e){return $r()&&(be("flagSettings")?.fastMode===!0||HEn(Je()))}`),
		replaceFirstFixed(data, `function HEn(e){if(e.fastMode!==!0)return!1;if(!e.fastModePerSessionOptIn)return!0;if(be("policySettings")?.fastModePerSessionOptIn===!0)return!1;return be("flagSettings")?.fastMode===!0}`, `function HEn(e){return e.fastMode===!0}`),
		replaceFirstFixed(data, `function ff(e){if(!$r())return!1;let n=e??Ah(),r=At(n);if($f(ze(r),"fast_mode",r))return!0;let o=r.toLowerCase();return o.includes("opus-4-8")||o.includes("opus-5")}`, `function ff(e){return $r()}`),
		replaceFirstFixed(data, `function wb(e,n){if(Zt()){if(e===null)return!!n;return!!n&&ff(e)}if(!ff(e))return!1;return!!n||Kwt(e)}`, `function wb(e,n){return $r()&&(n!==void 0?!!n:HEn(Je()))}`),
		replaceFirstFixed(data, `...$r()&&{fastMode:Kwt(ot??null)}`, `fastMode:Kwt(ot??null)`),
		replaceFirstFixed(data, `...Ne.gates.fastModeEnabled&&{fastMode:Ht.options.fastMode}`, `fastMode:Ht.options.fastMode`),
		replaceFirstFixed(data, `d={model:r.model,...$r()&&{fastMode:r.fastMode}}`, `d={model:r.model,fastMode:r.fastMode}`),
		replaceFirstFixed(data, `...$r()&&{fastMode:n.fastMode}`, `fastMode:n.fastMode`),
	}
	if bytes.Count(data, []byte(`...$r()&&{fastMode:Sc}`)) != 2 {
		return false
	}
	checks = append(checks,
		replaceAllFixed(data, `...$r()&&{fastMode:Sc}`, `fastMode:Sc`),
		replaceFirstFixed(data, `...$r()?{fastMode:Sc}:!1`, `fastMode:Sc`),
		replaceFirstFixed(data, `if($r()&&_(()=>cS())&&!Joe()&&_(()=>ff(_e))&&!!to.fastMode)Ye="fast";`, `if(to.fastMode)Ye="fast";`),
	)
	for _, check := range checks {
		if !check {
			return false
		}
	}
	return true
}

func patchActiveFastModeBrand_2_1_258(data []byte) bool {
	const target = "Fable 5.1\x00\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok"
	if bytes.Count(data, []byte(claude258ActiveFastModeBrandTarget)) != 1 || bytes.Count(data, []byte(target)) != 1 {
		return false
	}
	return replaceFirstFixed(data, target, "Codex+   \x00\x00\x00\x1b\x00\x00\x80\x5c\x46\x9a\x00\\$[\\d.]+\\/\\$[\\d.]+ per Mtok")
}

func patchFastModePricing_2_1_258(data []byte) bool {
	return replaceFirstFixed(data, "function tie(e){return`${nD(e.inputTokens)}/${nD(e.outputTokens)} per Mtok`}", `function tie(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_258(data []byte) bool {
	return replaceClaude208Function(data, "function jS(w,T,I){", "var $S=", `function jS(w,T,I){return null}`)
}

func patchResumeCommandHints_2_1_258(data []byte) bool {
	required := []struct {
		old           string
		replacement   string
		expectedCount int
	}{
		{"\nResume this session with:\nclaude ", "\nResume with:\nclaudodex ", 2},
		{"Previous session saved \\xB7 resume with: claude --resume ", "Previous session saved \\xB7 resume: claudodex --resume ", 1},
		{"Run claude --continue or claude --resume to resume a conversation", "Run claudodex --resume to resume a conversation", 2},
		{"Run claude --resume to pick a session, or start a new one.", "Run claudodex --resume to pick a session, or start a new one.", 2},
	}
	for _, target := range required {
		if bytes.Count(data, []byte(target.old)) != target.expectedCount {
			return false
		}
	}
	for _, target := range required {
		if !replaceAllFixed(data, target.old, target.replacement) {
			return false
		}
	}
	return true
}

func patchCompactProgressCurve_2_1_258(data []byte) bool {
	return replaceFirstFixed(data, `function So(t){let i=Math.max(0,t)/1000,c=1-Math.exp(-i/90);return Math.min(95,Math.round(c*100))}`, `function So(t){let i=Math.max(0,t)/2000,c=1-Math.exp(-i/90);return Math.min(95,Math.round(c*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_258(data []byte) bool {
	for _, transformation := range claude258RemoteControlTransformations() {
		if !transformation.apply(data) {
			return false
		}
	}
	return true
}

func claude258RemoteControlTransformations() []claude258Transformation {
	return []claude258Transformation{
		{"token", func(data []byte) bool {
			return replaceClaude208Function(data, "function QH(){return}function uG(){return}", "function qTr(e){", `function QH(){return process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}function uG(){return}function v_(){return QH()||Jt()?.accessToken}async function JC(e){return v_()}function wfe(){return uG()??Kt().BASE_API_URL}function uoe(){let e=process.env.CLAUDE_REMOTE_CONTROL_SESSION_NAME_PREFIX||n();return qTr(e)||"remote-control"}`)
		}},
		{"visible", func(data []byte) bool {
			return replaceFirstFixed(data, `function Sb(){if(s())return!0;if(RG())return!1;return!xA()&&fFe()}`, `function Sb(){return!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"available", func(data []byte) bool {
			return replaceFirstFixed(data, `function bwn(){if(s())return!0;return!RG()&&!xA()&&bj()}`, `function bwn(){return!!QH()}`)
		}},
		{"enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `async function Twn(){if(s())return!0;if(RG())return!1;return bj()&&!xA()&&u()&&await $p("tengu_ccr_bridge")}`, `async function Twn(){return!RG()&&!xA()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
		}},
		{"error", func(data []byte) bool {
			return replaceClaude208Function(data, "async function jqt(){", "function T(){", `async function jqt(){if(RG())return"Remote Control is disabled by your organization's policy (managed setting disableRemoteControl).";if(xA())return"Remote Control is not available inside a cloud session.";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run claude auth login outside Claudodex, then restart Claudodex.";return null}`)
		}},
		{"command-enabled", func(data []byte) bool {
			return replaceFirstFixed(data, `function e(){if(Sb())return!0;try{return bj()&&!xA()&&!RG()&&ql().source==="none"&&Fp({skipRetrievingKeyFromApiKeyHelper:!0}).source==="none"&&!w_n.isC4EUpsellCommandEnabled()}catch{return!1}}`, `function e(){return!0}`)
		}},
		{"command-visible", func(data []byte) bool {
			return replaceFirstFixed(data, `get isHidden(){return!Sb()}`, `get isHidden(){return!1}`)
		}},
	}
}
