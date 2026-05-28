package launcher

import (
	"bytes"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestApplyClaudeUIPatchesBrandsHeaderAndModelPicker(t *testing.T) {
	data := []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Welcome back!`,
		`Opus 4.8 is here!`,
		`Opus 4.8 is now available!`,
		`Set the AI model for Claude Code`,
		`Claude Code'll be able to read, edit, and execute files here.`,
		`WARNING: Claude Code running in Bypass Permissions mode`,
		`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`,
		`No, exit Claude Code`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`Opus with 1M context`,
		`Most capable for complex work`,
		`Best for everyday tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`j4.createElement(V,{bold:!0},"Claude Code")`,
		`j4.createElement(V,{dimColor:!0},"v",E)`,
		`Lq("claude",d)("Claude Code")`,
		`Lq("inactive",d)(` + "`v${h}`" + `)`,
		`Lq("claude",d)(" Claude Code ")`,
		`w_=h4()?Y?P4.createElement(B):null:null`,
		`function P6_(){let H="2.1.153",_=Px6(),q="/code/claude",K=q,O=vq(),T=O!=="firstParty"?eYH[O]:Zq()?zH6():"API Usage Billing",z=o8().agent;return{version:H,cwd:K,billingType:T,agentName:z}}                                                                                                                                                                                                                                                                                                                                                                               function RuK(H,_,q){}`,
		`function muK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`function jl3(H=!1){if(Zq()){if(Re()||wAH()||IUH()){let z=[ML6(H)];if(!LP()&&X6H()&&!Zr8())z.push(lkK());if(z.push(Al3),Q5H())z.push(ckK());return z.push(nkK),z}function Jl3(H){}`,
	}, "\n"))

	if !applyClaudeUIPatches_2_1_153(data, "0.1.0", "2.1.153", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_153 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`Bugs:\ngithub.com/bassner/claudodex/issues`,
		"Info",
		"Welcome back",
		"Set the AI model for Claudodex",
		"Claudodex can read, edit, and execute files here.",
		"WARNING: Claudodex running in Bypass Permissions mode",
		"In Bypass Permissions mode, Claudodex will not ask",
		"No, exit Claudodex",
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"Experimental - treat it as such.",
		"If you run into issues, please file a report at",
		"https://github.com/bassner/claudodex/issues",
		"Codex Plan",
		"Switch between Codex-backed models.",
		"Codex model",
		"Default (Claudodex)",
		"default Codex work",
		"gpt-5.4 everyday coding",
		"gpt-5.4-mini quick code",
		` via Codex model \xB7 `,
		`"Claudodex  "`,
		`"0.1.0 using Claude Code v2.1.153"`,
		"w_=0?",
		"let z=[]",
		"return z",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"z.push(lkK())",
		"z.push(Al3)",
		"z.push(ckK())",
		"return z.push(nkK),z",
		"let z=[ML6(H)]",
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("patched data still contains %q:\n%s", notWant, got)
		}
	}
}

func TestApplyClaudeUIPatches154BrandsHeaderAndModelPicker(t *testing.T) {
	data := claude154PatchFixture(t)

	if !applyClaudeUIPatches_2_1_154(data, "0.1.0", "2.1.154", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_154 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`Bugs:\ngithub.com/bassner/claudodex/issues`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"Experimental - treat it as such.",
		"Codex backend on",
		"Codex backend active",
		"Set the AI model for Claudodex",
		"Claudodex can read files here",
		"WARNING: Claudodex running in Bypass Permissions mode",
		"In Bypass Permissions mode, Claudodex will not ask",
		"No, exit Claudodex",
		"Codex Plan",
		"Switch between Codex-backed models.",
		"Codex model",
		"Default (Claudodex)",
		"Default Codex route",
		"default Codex work",
		"default Codex reasoning tasks",
		"gpt-5.4 everyday coding",
		"gpt-5.4-mini quick code",
		` via Codex model \xB7 `,
		"Fast mode for Claudodex",
		"Use /fast to toggle Fast mode.",
		`function or_(H=!1){return"Default Codex route \xB7 default Codex work"}`,
		`CLAUDE_LOCAL_OAUTH_API_BASE`,
		`fetch(H+"/api/oauth/usage"`,
		`let j=0,J=Y,M=!Gc()&&!VH_(K,O),D=!1;`,
		"Resume with:\nclaudodex ${O}--resume ${q}\n",
		" resume: claudodex --resume ",
		"Run claudodex --resume to resume a conversation",
		"Open `claudodex agents` to attach",
		`command:` + "`" + `cd ${AK([H.projectPath])}; claudodex --resume ${T}` + "`",
		`kO.default.createElement(V,{bold:!0},"claudodex agents")," or run:"`,
		`kO.default.createElement(V,null,$,"claudodex --resume ",q," --fork-session")`,
		`r=Y4.createElement(V,{bold:!0},"Claudodex")`,
		`"Claudodex  "`,
		`"0.1.0 using Claude Code v2.1.154"`,
		"let $=[cL6(H)];return $",
		"let T=[cL6(H)];return T",
		"return _",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"$.push(Ri3)",
		"kN6.Title",
		" to turn on Fast mode (",
		"$.push(YNK)",
		"T.push(MNK(!1))",
		"T.push(JNK(!1))",
		"T.push(ONK(H))",
		"T.push(_NK())",
		"T.push(KNK())",
		"T.push(TNK()??jNK())",
		"_.push(fi3())",
		"_.push(Pi3())",
		"_.push(ONK())",
		"_.push(ANK())",
		"_.push(wNK())",
		"_.push(Xi3())",
		"_.push(Wi3(H))",
		"_.push(Gi3())",
		`let j=w,J=Y,M=!Gc()&&!VH_(K,O),D=!1;`,
		"Resume this session with:\nclaude ${O}--resume ${q}\n",
		" resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents`",
		`command:` + "`" + `cd ${AK([H.projectPath])} ${ce8()} claude --resume ${T}` + "`",
		`kO.default.createElement(V,{bold:!0},"claude agents")," to attach to it, or run:"`,
		`kO.default.createElement(V,null," ",$,"claude --resume ",q," --fork-session")`,
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("patched data still contains %q:\n%s", notWant, got)
		}
	}
}

func TestApplyClaudeUIPatches154FailsWhenCriticalUsagePatchMissing(t *testing.T) {
	data := []byte(strings.ReplaceAll(string(claude154PatchFixture(t)), "async function WXH()", "async function MISSING_WXH()"))

	if applyClaudeUIPatches_2_1_154(data, "0.1.0", "2.1.154", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_154 succeeded without the critical usage patch target")
	}
}

func TestApplyClaudeUIPatches154FailsWhenCriticalContextHintPatchMissing(t *testing.T) {
	data := []byte(strings.ReplaceAll(string(claude154PatchFixture(t)), "function d44(H){", "function MISSING_d44(H){"))

	if applyClaudeUIPatches_2_1_154(data, "0.1.0", "2.1.154", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_154 succeeded without the critical context hint patch target")
	}
}

func TestApplyClaudeUIPatches154FailsWhenCriticalResumeHintPatchMissing(t *testing.T) {
	data := []byte(strings.ReplaceAll(string(claude154PatchFixture(t)), "Resume this session with:", "Resume target missing:"))

	if applyClaudeUIPatches_2_1_154(data, "0.1.0", "2.1.154", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_154 succeeded without the critical resume hint patch target")
	}
}

func claude154PatchFixture(t *testing.T) []byte {
	t.Helper()
	logo154 := `function L6_(){let H=process.env.DEMO_VERSION??` + "`" + `${{ISSUES_EXPLAINER:"report the issue at https://github.com/anthropics/claude-code/issues",PACKAGE_URL:"@anthropic-ai/claude-code",README_URL:"https://code.claude.com/docs/en/overview",VERSION:"2.1.154",FEEDBACK_CHANNEL:"https://github.com/anthropics/claude-code/issues",BUILD_TIME:"2026-05-28T12:27:24Z",GIT_SHA:"b84d2da9ada13121515426fc644786a303e9ac53"}.VERSION}${fS()}` + "`" + `,_=Yu6(),q=process.env.DEMO_VERSION?"/code/claude":s5(b_()),K=xH(process.env.CLAUDE_CODE_HIDE_CWD)?"":_?` + "`" + `${q} in ${_.replace(/^https?:\/\//,"")}` + "`" + `:q,O=Zq(),T=O!=="firstParty"?wAH[O]:Lq()?VH6():"API Usage Billing",$=i8().agent;return{version:H,cwd:K,billingType:T,agentName:$}}                                                                                                                                                                                                                                                                                                                                                                               function gmK(H,_,q){}`
	defaultDescription154 := `function or_(H=!1){if(pe()||RAH()||UUH()){let q=LR(),K=HJ(NP(q))??"Opus",O=H&&Pj(q);if(VP())return` + "`" + `${K} with 1M context \xB7 Most capable for complex work${O?EKH(!0,q):""}` + "`" + `;return` + "`" + `${K} \xB7 Most capable for complex work${O?EKH(!0,q):""}` + "`" + `}return` + "`" + `${HJ(NP(EN()))??"Sonnet"} \xB7 Best for everyday tasks` + "`" + `}`
	return []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Welcome back!`,
		`Opus 4.8 is here!`,
		`Opus 4.8 is now available!`,
		`Set the AI model for Claude Code`,
		`Set the AI model for Claude Code (currently `,
		`Claude Code will be able to read files in this directory and make edits when auto-accept edits is on.`,
		`WARNING: Claude Code running in Bypass Permissions mode`,
		`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`,
		`No, exit Claude Code`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`Opus with 1M context`,
		`Most capable for complex work`,
		`Most capable for complex reasoning tasks`,
		`Best for everyday tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`Fast mode for Claude Code uses Claude Opus with faster output (it does not downgrade to a smaller model). It can be toggled with /fast and is available on Opus 4.8/4.7/4.6.`,
		`Use /fast to turn on Fast mode (Opus 4.8).`,
		`X4.createElement(V,{dimColor:!0},"Use ",X4.createElement(V,{bold:!0},"/fast")," to turn on Fast mode (",Bp(),").")`,
		`r=kN6?Y4.createElement(kN6.Title,null):Y4.createElement(V,{bold:!0},"Claude Code")`,
		`Y4.createElement(V,{bold:!0},"Claude Code")`,
		`Y4.createElement(V,{dimColor:!0},"v",E)`,
		`Pq("claude",U)("Claude Code")`,
		`Pq("claude",U)(" Claude Code ")`,
		`Pq("claude",d)("Claude Code")`,
		`Pq("inactive",d)(` + "`v${h}`" + `)`,
		`Pq("claude",d)(" Claude Code ")`,
		logo154,
		defaultDescription154,
		`async function WXH(){return DK("api_usage_fetch",async()=>{if(!Lq()||!$k())return{};let H=0,_=await iZ(async()=>{H++,N(` + "`" + `fetchUtilization: GET /api/oauth/usage (attempt ${H})` + "`" + `);let q=await r7.get("/api/oauth/usage",{timeout:5000,headers:{"Content-Type":"application/json"},refreshOAuth:!0});if(!q.ok)throw Error(` + "`" + `Auth error: ${q.reason==="no-auth"?q.detail:q.reason}` + "`" + `);return q});return N(` + "`" + `fetchUtilization: 200 after ${H} attempt(s)${H>1?" (401\u2192refresh\u2192retry succeeded)":""}` + "`" + `),_.data})}`,
		`var bR_=R(()=>{Jq();lH();t2();Y6();fX()});`,
		`function qpK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`function d44(H){let _=Q44.c(15),{tokenUsage:q,model:K}=H,O=M_(NZO),T;if(_[0]!==O||_[1]!==K||_[2]!==q)T=XSH(q,K,O),_[0]=O,_[1]=K,_[2]=q,_[3]=T;else T=_[3];let $=T,z=sE6();if($.level==="ok"||z)return null;let Y=$.pctLeft,A=fG(),w;if(_[4]===Symbol.for("react.memo_cache_sentinel"))w=CrH("warning"),_[4]=w;else w=_[4];let j=w,J=Y,M=!Gc()&&!VH_(K,O),D=!1;if(M||D){let Z=$qH(K,O),W;if(_[5]!==Z||_[6]!==q)W=Math.round((Z-q)/Z*100),_[5]=Z,_[6]=q,_[7]=W;else W=_[7];J=Math.max(0,W)}let f=M?` + "`" + `${100-J}% context used` + "`" + `:` + "`" + `${J}% until auto-compact` + "`" + `;if(A){let Z=j?` + "`" + `${f} \xB7 ${j}` + "`" + `:f,W;if(_[9]!==Z)W=ly_.createElement(V,{dimColor:!0,wrap:"truncate"},Z),_[9]=Z,_[10]=W;else W=_[10];return W}let X;if(_[11]!==Y)X=j?` + "`" + `Context low (${Y}% remaining) \xB7 ${j}` + "`" + `:xH(process.env.DISABLE_COMPACT)?` + "`" + `Context low (${Y}% remaining)` + "`" + `:` + "`" + `Context low (${Y}% remaining) \xB7 Run /compact to compact & continue` + "`" + `,_[11]=Y,_[12]=X;else X=_[12];let P;if(_[13]!==X)P=ly_.createElement(V,{color:"error",wrap:"truncate"},X),_[13]=X,_[14]=P;else P=_[14];return P}function NZO(H){return H.autoCompactWindow}`,
		`Resume this session with:
claude ${O}--resume ${q}
`,
		`Previous session saved \xB7 resume with: claude --resume ${I}`,
		`Run claude --continue or claude --resume to resume a conversation`,
		`Open ` + "`" + `claude agents` + "`" + ` to attach to it, or stop it there first to resume here.`,
		`). Use ` + "`" + `claude agents` + "`" + ` to find and attach to it, or add --fork-session to branch off a copy.`,
		`command:` + "`" + `cd ${AK([H.projectPath])} ${ce8()} claude --resume ${T}` + "`",
		`kO.default.createElement(V,{bold:!0},"claude agents")," to attach to it, or run:"`,
		`kO.default.createElement(V,null," ",$,"claude --resume ",q," --fork-session")`,
		`function ki3(H=!1){if(Lq()){if(pe()||RAH()||UUH()){let $=[cL6(H)];if(!VP()&&to()&&!Lo8())$.push(zNK());if($.push(Ri3),s5H())$.push($NK());return $.push(YNK),$}let T=[cL6(H)];if(s5H())T.push($NK());if(VP())T.push(MNK(!1));else if(T.push(JNK(!1)),to()&&!Lo8())T.push(zNK());return T.push(YNK),T}if(UT()){let T=[cL6(H)],$=qNK();if($!==void 0)T.push($);else if(!VP()&&to()&&!Lo8())T.push(ONK(H));let z=HNK();if(z!==void 0)T.push(z);else if(T.push(_NK()),s5H())T.push(KNK());return T.push(TNK()??jNK()),T}let _=[cL6(H)],q=HNK();if(q!==void 0)_.push(q);else if(_.push(_NK()),s5H())_.push(KNK());let K=qNK();if(K!==void 0)_.push(K);else{if(_.push(fi3()),_.push(Pi3()),to()&&!Ie(zO().opus48))_.push(ONK());if(_.push(ANK()),to()&&!Ie(zO().opus47))_.push(wNK());if(_.push(Xi3()),to())_.push(Wi3(H))}let O=TNK();if(O!==void 0)_.push(O);else _.push(Gi3());return _}function Vi3(H){}`,
	}, "\n"))
}

func TestFindClaudeUIPatchRequiresVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.153", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.153 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.154", "bc9881b107d7be1743c64c8b72dd66798f5d0947dbc48ed0d77964c473661fd4")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.154 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.156", "9c1e8601031f5cbb3101e49dda22bf8ba31183692c705e267a6923585fa2ba09")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.156 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.154", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.156", "bc9881b107d7be1743c64c8b72dd66798f5d0947dbc48ed0d77964c473661fd4"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.153", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
}

func TestWarnClaudePatchSkippedMentionsPatchTargetAndIssue(t *testing.T) {
	var stderr bytes.Buffer
	warnClaudePatchSkipped(&stderr, "2.1.154", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", errClaudePatchUnsupported)
	got := stderr.String()
	for _, want := range []string{
		"warning: Claudodex has no verified UI patch",
		"Claude Code 2.1.154",
		runtime.GOOS + "/" + runtime.GOARCH,
		"sha256:ffffffffffff",
		"launching with the unpatched Claude Code UI",
		"https://github.com/bassner/claudodex/issues",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("warning missing %q:\n%s", want, got)
		}
	}
}

func TestWarnClaudePatchSkippedHandlesPatchFailures(t *testing.T) {
	var stderr bytes.Buffer
	warnClaudePatchSkipped(&stderr, "2.1.153", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254", errors.New("boom"))
	got := stderr.String()
	if !strings.Contains(got, "could not prepare the UI patch") || !strings.Contains(got, "boom") {
		t.Fatalf("warning = %q", got)
	}
}

func TestDetectClaudeVersionFromResolvedPath(t *testing.T) {
	if !looksLikeVersion("2.1.153") {
		t.Fatal("version-like path was rejected")
	}
	if looksLikeVersion("claude") {
		t.Fatal("non-version path was accepted")
	}
}
