package launcher

import (
	"bytes"

	"github.com/bassner/claudodex/internal/modelconfig"
)

var claudeUIPatch_2_1_209 = claudeUIPatchSpec{
	Version: "2.1.209",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "59d2de7f49db2f75d5c33bbb46a6b8f288ad24d40b61e30602a502bb7ddc380c",
	Apply:   applyClaudeUIPatches_2_1_209,
}

type claude209UIBrandingReplacement struct {
	old           string
	replacement   string
	expectedCount int
}

var claude209UIBrandingReplacements = []claude209UIBrandingReplacement{
	{`Welcome to Claude Code for `, `Welcome to Claudodex for `, 2},
	{`Welcome to Claude Code`, `Welcome to Claudodex`, 10},
	{`No, and tell Claude what to do differently `, `No, and tell Codex what to do differently `, 4},
	{`and tell Claude what to do differently`, `and tell Codex what to do differently`, 12},
	{`and tell Claude what to do next`, `and tell Codex what to do next`, 6},
	{`Claude has context of `, `Codex has context of `, 2},
	{`To add hooks, edit settings.json directly or ask Claude`, `To add hooks, edit settings.json directly or ask Codex`, 4},
	{`To modify or remove this hook, edit settings.json directly or ask Claude to help.`, `To modify or remove this hook, edit settings.json directly or ask Codex to help.`, 2},
	{`Fix with Claude`, `Fix with Codex`, 3},
	{`Claude ended this conversation. Start a new session (or /clear) to continue.`, `Codex ended this conversation. Start a new session (or /clear) to continue.`, 12},
	{`Claude ended this conversation. Start a new session to continue.`, `Codex ended this conversation. Start a new session to continue.`, 2},
	{`Claude Code needs your input`, `Claudodex needs your input`, 4},
	{`[Image data detected and sent to Claude]`, `[Image data detected and sent to Codex]`, 4},
	{`Push when Claude decides`, `Push when Codex decides`, 5},
	{`was stopped by Claude`, `was stopped by Codex`, 2},
	{`get pinged when Claude finishes \xB7 enable push notifications in`, `get pinged when Codex finishes \xB7 enable push notifications in`, 1},
	{`Tell Claude what to change`, `Tell Codex what to change`, 2},
	{`Teach Claude your rules`, `Teach Codex your rules`, 2},
	{`Claude can make mistakes.`, `Codex can make mistakes.`, 2},
	{`You're responsible for Claude's actions and should always`, `You're responsible for Codex's actions and should always`, 2},
	{`Sorry, Claude encountered an error`, `Sorry, Codex encountered an error`, 2},
	{`You can grant Claude access to additional directories without changing your current working directory.`, `You can grant Codex access to additional directories without changing your current working directory.`, 2},
	{`You can hit Enter while Claude is working to queue a follow-up or steer it mid-turn \u2014 no need to wait for it to finish.`, `You can hit Enter while Codex is working to queue a follow-up or steer it mid-turn \u2014 no need to wait for it to finish.`, 1},
	{`Working on a plan or design doc? Ask Claude to publish it as an artifact \u2014 a polished web page you can open in your browser.`, `Working on a plan or design doc? Ask Codex to publish it as an artifact \u2014 a polished web page you can open in your browser.`, 1},
	{`Setting a 200K auto-compact window keeps sessions trimmed automatically \u2014 Claude summarizes earlier so each turn stays cheaper without manual /compact.`, `Claudodex derives its auto-compact window from live Codex model metadata.`, 1},
	{`Use /btw to ask a quick side question without interrupting Claude's current work`, `Use /btw to ask a quick side question without interrupting Codex's current work`, 2},
	{`Start with small features or bug fixes, tell Claude to propose a plan, and verify its suggested edits`, `Start with small features or bug fixes, tell Codex to propose a plan, and verify its suggested edits`, 2},
	{`Use git worktrees to run multiple Claude sessions in parallel.`, `Use git worktrees to run Claudodex sessions in parallel.`, 2},
	{`This conversation is already open in another running Claude session \u2014 use that one, or close it and try again`, `This conversation is open in another running Claudodex session \u2014 use that one, or close it and try again`, 1},
	{`Generate a report analyzing your Claude Code sessions`, `Generate a report analyzing your Claudodex sessions`, 4},
	{`This changes how Claude Code communicates with you`, `This changes how Claudodex communicates with you`, 2},
	{`Use Claude Code's terminal setup?`, `Use Claudodex's terminal setup?`, 2},
	{`Show Claude Code status including version, model, account, API connectivity, and tool statuses`, `Show Claudodex status including version, model, account, API connectivity, and tool statuses`, 2},
	{`Set up Claude Code's status line UI`, `Set up Claudodex's status line UI`, 2},
	{`What should Claude do instead?`, `What should Codex do instead?`, 2},
	{`Claude is now exploring and designing an implementation approach.`, `Codex is now exploring and designing an implementation approach.`, 2},
	{`Set the AI model for Claude Code`, `Set the AI model for Claudodex`, 4},
	{`WARNING: Claude Code running in Bypass Permissions mode`, `WARNING: Claudodex running in Bypass Permissions mode`, 2},
	{`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`, `In Bypass Permissions mode, Claudodex will not ask for your approval before running potentially dangerous commands.`, 2},
	{`No, exit Claude Code`, `No, exit Claudodex`, 2},
	{`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`, `Switch between Codex-backed models. Your pick becomes the default for new sessions. For direct model names, use --model.`, 2},
	{`Check the Claude Code changelog for updates`, `Bugs:\ngithub.com/bassner/claudodex/issues`, 3},
	{`Claude Max`, `Codex Plan`, 4},
	{`Hit Enter to queue up additional messages while Claude is working.`, `Hit Enter to queue up additional messages while Codex is working.`, 2},
	{`Send messages to Claude while it works to steer Claude in real-time`, `Send messages to Codex while it works to steer Codex in real-time`, 2},
	{`Claude will think before responding`, `Codex will think before responding`, 2},
	{`Claude will respond without extended thinking`, `Codex will respond without extended thinking`, 2},
	{`In plan mode, Claude will:`, `In plan mode, Codex will:`, 2},
	{`Claude has written up a plan. Would you like to review it as an artifact first?`, `Codex has written up a plan. Would you like to review it as an artifact first?`, 2},
	{`Claude has written up a plan and is ready to execute. Would you like to proceed?`, `Codex has written up a plan and is ready to execute. Would you like to proceed?`, 2},
	{`Claude wants to exit plan mode`, `Codex wants to exit plan mode`, 2},
	{`Claude needs your input`, `Codex needs your input`, 2},
	{`Claude is waiting for your input`, `Codex is waiting for your input`, 2},
	{`Claude wants to search the web for: `, `Codex wants to search the web for: `, 2},
	{`Claude wants to fetch content from `, `Codex wants to fetch content from `, 4},
	{`Claude needs your permission`, `Codex needs your permission`, 2},
	{`Approving lets Claude write to ANY file in this project without another prompt for up to 4 hours (new and changed file contents are not shown for approval). Deletes and CLAUDE.md/.claude paths still ask every time.`, `Approving lets Codex write to ANY file in this project without another prompt for up to 4 hours (new and changed file contents are not shown for approval). Deletes and CLAUDE.md/.claude paths still ask every time.`, 2},
	{`Approving lets Claude write to ANY file in the project "`, `Approving lets Codex write to ANY file in the project "`, 2},
	{`Claude wants to use your browser`, `Codex wants to use your browser`, 4},
	{`Claude wants to guide you through `, `Codex wants to guide you through `, 2},
	{`Claude is using your computer `, `Codex is using your computer `, 4},
	{`Claude has ended this chat.`, `Codex has ended this chat.`, 2},
	{`Claude recalled a memory:`, `Codex recalled a memory:`, 2},
	{`How is Claude doing this session? (optional)`, `How is Codex doing this session? (optional)`, 2},
	{`How was Claude's recollection?`, `How was Codex's recollection?`, 2},
	{`Claude can spawn copies of itself to work in parallel.`, `Codex can spawn copies of itself to work in parallel.`, 2},
	{`A different way to work with Claude:`, `A different way to work with Codex:`, 2},
	{`Claude completes coding tasks efficiently and provides concise responses`, `Codex completes coding tasks efficiently and provides concise responses`, 2},
}

func applyClaudeUIPatches_2_1_209(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	if !validateClaude209UIBrandingReplacements(data, claude209UIBrandingReplacements) {
		return false
	}
	versionPatched := patchLogoDisplayDataFunction_2_1_209(data, claudodexVersion, claudeVersion)
	whatsNewPatched := patchWhatsNewFeedFunction_2_1_209(data)
	usagePatched := patchUsageFetchFunction_2_1_209(data)
	modelOptionsPatched := patchModelPickerOptions_2_1_209(data)
	modelExtraOptionsPatched := patchModelPickerExtraOptions_2_1_209(data)
	modelSelectionPatched := patchModelPickerSelectionValue_2_1_209(data)
	fastModePatched := patchFastModeRuntimeFunctions_2_1_209(data)
	fastModePricingPatched := patchFastModePricing_2_1_209(data)
	contextWarningHintPatched := patchContextWarningHint_2_1_209(data)
	resumeHintsPatched := patchResumeCommandHints_2_1_196(data)
	compactProgressPatched := patchCompactProgressCurve_2_1_209(data)
	remoteControlPatched := patchRemoteControlRuntimeFunctions_2_1_209(data)
	brandingPatched := applyClaude209UIBrandingReplacements(data, claude209UIBrandingReplacements)

	changed := versionPatched || whatsNewPatched || usagePatched || modelOptionsPatched || modelExtraOptionsPatched || modelSelectionPatched || fastModePatched || fastModePricingPatched || contextWarningHintPatched || resumeHintsPatched || compactProgressPatched || remoteControlPatched || brandingPatched
	// The 2.1.209 table is authoritative for overlapping UI strings. This shared
	// legacy pass only handles remaining fixed replacements inherited from 2.1.208.
	changed = applyClaudeUIFixedReplacements_2_1_208(data, modelCfg) || changed

	if !versionPatched || !whatsNewPatched || !usagePatched || !modelOptionsPatched || !modelExtraOptionsPatched || !modelSelectionPatched || !fastModePatched || !fastModePricingPatched || !contextWarningHintPatched || !resumeHintsPatched || !compactProgressPatched || !remoteControlPatched || !brandingPatched {
		return false
	}
	return changed
}

func validateClaude209UIBrandingReplacements(data []byte, replacements []claude209UIBrandingReplacement) bool {
	for _, replacement := range replacements {
		if replacement.old == "" || replacement.replacement == "" || replacement.expectedCount < 1 || len(replacement.replacement) > len(replacement.old) {
			return false
		}
		if bytes.Count(data, []byte(replacement.old)) != replacement.expectedCount {
			return false
		}
	}
	return true
}

func applyClaude209UIBrandingReplacements(data []byte, replacements []claude209UIBrandingReplacement) bool {
	changed := false
	for _, replacement := range replacements {
		changed = replaceAllFixed(data, replacement.old, replacement.replacement) || changed
	}
	for _, replacement := range replacements {
		if bytes.Contains(data, []byte(replacement.old)) || !bytes.Contains(data, []byte(replacement.replacement)) {
			return false
		}
	}
	return changed
}

func patchLogoDisplayDataFunction_2_1_209(data []byte, claudodexVersion, claudeVersion string) bool {
	replacement := `function npt(){let e=process.env.DEMO_VERSION??` + quoteJSString(claudodexLogoVersion(claudodexVersion, claudeVersion)) + `,t=wVo(),r=process.env.DEMO_VERSION?"/code/claude":Zd(At()),n=Se.CLAUDE_CODE_HIDE_CWD?"":t?` + "`${r} in ${t.replace(/^https?:\\/\\//,\"\")}`" + `:r,o="Codex Plan",i=Jn().agent;return{version:e,cwd:n,billingType:o,agentName:i}}`
	return replaceClaude208Function(data, "function npt(){", "function jta(", replacement)
}

func patchWhatsNewFeedFunction_2_1_209(data []byte) bool {
	const old = `function Qta(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`
	const replacement = `function Qta(e){return{title:"Claudodex Info",lines:["Thank you for using Claudodex!","Experimental - treat it as such.","https://github.com/bassner/claudodex/issues"].map(text=>({text}))}}`
	return replaceFirstFixed(data, old, replacement)
}

func patchUsageFetchFunction_2_1_209(data []byte) bool {
	const replacement = `function bwe(){return Dc("api_usage_fetch",async()=>{let e=(process.env.CLAUDE_LOCAL_OAUTH_API_BASE||"https://api.anthropic.com").replace(/\/$/,""),t=await fetch(e+"/api/oauth/usage",{headers:{"Content-Type":"application/json"}});if(!t.ok)throw Error("Auth error: "+t.status);return await t.json()})}`
	return replaceClaude208Function(data, "function bwe(){", "var _Ig=", replacement)
}

func patchModelPickerOptions_2_1_209(data []byte) bool {
	const replacement = `function CDX209(e){let n=(r)=>String(r??"").replace(/(\[1m\])+$/i,"").trim();if(e==null||e==="")return"opus";let t=n(e);return t===n(process.env.ANTHROPIC_DEFAULT_OPUS_MODEL)?"opus":t===n(process.env.ANTHROPIC_DEFAULT_SONNET_MODEL)?"sonnet":t===n(process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL)?"haiku":e}function mch(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus",t.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??t.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",t.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),n("sonnet",t.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??t.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",t.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),n("haiku",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??t.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",t.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`
	return replaceClaude208Function(data, "function mch(e=!1){", "function jEe(", replacement)
}

func patchModelPickerExtraOptions_2_1_209(data []byte) bool {
	const replacement = "function ych(e){let t=mch(e),r=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,n=CDX209(r);if(r&&n===r&&!t.some((l)=>l.value===r))t.push({value:r,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??r,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??`Custom model (${r})`});return t}"
	return replaceClaude208Function(data, "function ych(e){", "function ESi(", replacement)
}

func patchModelPickerSelectionValue_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, `WWy=TNe===null?$8e:ESi(SNe,TNe)??TNe`, `WWy=TNe===null?$8e:CDX209(TNe)`)
}

func patchFastModeRuntimeFunctions_2_1_209(data []byte) bool {
	rlPatched := replaceFirstFixed(data, `function rl(){if(En()!=="firstParty")return!1;return!dt(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`, `function rl(){return!dt(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`)
	e9Patched := replaceFirstFixed(data, `function e9(){return"Opus 4.8"}`, `function e9(){return"Codex AI"}`)
	ybtPatched := replaceFirstFixed(data, `function Ybt(){return"opus"+(z1()?"[1m]":"")}`, `function Ybt(){return"opus"}`)
	wtPatched := replaceFirstFixed(data, `function WT(e){if(!rl())return!1;let t=e??QO(),r=Qo(t);if(MW(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`, `function WT(e){return rl()}`)
	return rlPatched && e9Patched && ybtPatched && wtPatched
}

func patchFastModePricing_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, "function GUe(e){return`${IVl(e.inputTokens)}/${IVl(e.outputTokens)} per Mtok`}", `function GUe(e){return"Codex priority"}`)
}

func patchContextWarningHint_2_1_209(data []byte) bool {
	return replaceClaude208Function(data, "function TJi(e,t,r){", "function cZn(", `function TJi(e,t,r){return null}`)
}

func patchCompactProgressCurve_2_1_209(data []byte) bool {
	return replaceFirstFixed(data, `function Iod(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`, `function Iod(e){let t=Math.max(0,e)/2000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`)
}

func patchRemoteControlRuntimeFunctions_2_1_209(data []byte) bool {
	visiblePatched := replaceFirstFixed(data, `function Ox(){if(PHo())return!0;if(UOt())return!1;return!n2()&&DCt()}`, `function Ox(){return!!Se.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	enabledPatched := replaceFirstFixed(data, `async function P7s(){if(PHo())return!0;if(UOt())return!1;return BOt()&&!n2()&&jpr()&&await Lq("tengu_ccr_bridge")}`, `async function P7s(){return!UOt()&&!n2()&&!!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN}`)
	errorPatched := replaceClaude208Function(data, "async function OHo(){", "function BS_(", "async function OHo(){if(UOt())return\"Remote Control is disabled by your organization's policy (managed setting `disableRemoteControl`).\";if(n2())return\"Remote Control is not available inside a cloud session.\";if(!process.env.CLAUDE_BRIDGE_OAUTH_TOKEN)return\"Remote Control requires a normal Claude login. Run `claude auth login` outside Claudodex, then restart Claudodex.\";return null}")
	return visiblePatched && enabledPatched && errorPatched
}
