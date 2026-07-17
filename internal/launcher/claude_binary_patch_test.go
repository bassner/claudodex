package launcher

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
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
		"gpt-5.6-terra everyday",
		"gpt-5.6-luna quick code",
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
		"gpt-5.6-terra everyday",
		"gpt-5.6-luna quick code",
		` via Codex model \xB7 `,
		"Fast mode for Claudodex",
		"Fast mode (Codex priority)",
		"Uses Codex priority service tier when available.",
		"Codex priority",
		"Claudodex:",
		"https://github.com/bassner/claudodex",
		"Use /fast to toggle Fast mode.",
		"Codex priority remains active while available.",
		"Codex AI",
		`function Pj(H){return C4()}`,
		`function xUH(){return"opus"}`,
		`M="Codex priority",_[1]=M`,
		`return` + "`" + `${$} Fast mode ON${z} \xB7 Codex priority` + "`",
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
		`function ki3(H=!1){let _=[],q=qNK();if(q!==void 0)_.push(q);let K=HNK();if(K!==void 0)_.push(K);let O=TNK();if(O!==void 0)_.push(O);return _}function ZX(H){return H===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":H===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":H===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":H}`,
		`j=$q(),J=(q=ZX(q))??KN_,[M,D]`,
		`function v__(H=!1){let _=process.env,q=(O,T,$)=>({value:O,label:T,description:$});return[q("opus",_.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??_.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",_.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),q("sonnet",_.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??_.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",_.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),q("haiku",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??_.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"$.push(Ri3)",
		"kN6.Title",
		" to turn on Fast mode (",
		"model set to Opus 4.8",
		"model set to ${Bp()}",
		"Switching to other models turns off fast mode",
		"Yk(vGH(!0",
		"$10/$50 per Mtok",
		"https://code.claude.com/docs/en/fast-mode",
		"$.push(YNK)",
		"T.push(MNK(!1))",
		"T.push(JNK(!1))",
		"T.push(ONK(H))",
		"T.push(_NK())",
		"T.push(KNK())",
		"T.push(TNK()??jNK())",
		"let $=[cL6(H)]",
		"let T=[cL6(H)]",
		"let _=[cL6(H)]",
		"j=$q(),J=q===null?KN_:q,[M,D]",
		"j=$q(),q=qX(q),J=q??KN_,[M,D]",
		"j=$q(),J=(q=qX(q))??KN_,[M,D]",
		"function v__(H=!1){let _=ki3(H)",
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

func TestApplyClaudeUIPatches156SlowsCompactProgressBar(t *testing.T) {
	data := append(claude154PatchFixture(t), []byte(`
function Io7(H){let _=Math.max(0,H)/1000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}
`)...)

	if !applyClaudeUIPatches_2_1_156(data, "0.1.0", "2.1.156", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_156 reported no changes")
	}
	got := string(data)
	if !strings.Contains(got, `function Io7(H){let _=Math.max(0,H)/2000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`) {
		t.Fatalf("patched data missing slowed compact progress curve:\n%s", got)
	}
	if strings.Contains(got, `Math.max(0,H)/1000,q=1-Math.exp(-_/90)`) {
		t.Fatalf("patched data still contains original compact progress curve:\n%s", got)
	}
}

func TestApplyClaudeUIPatches156FailsWhenCriticalCompactProgressPatchMissing(t *testing.T) {
	data := claude154PatchFixture(t)

	if applyClaudeUIPatches_2_1_156(data, "0.1.0", "2.1.156", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_156 succeeded without the critical compact progress patch target")
	}
}

func TestApplyClaudeUIPatches159BrandsHeaderAndModelPicker(t *testing.T) {
	data := claude159PatchFixture(t)

	if !applyClaudeUIPatches_2_1_159(data, "0.1.0", "2.1.159", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_159 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`Bugs:\ngithub.com/bassner/claudodex/issues`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"Experimental - treat it as such.",
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
		"gpt-5.6-terra everyday",
		"gpt-5.6-luna quick code",
		` via Codex model \xB7 `,
		"Fast mode for Claudodex",
		"Fast mode (Codex priority)",
		"Uses Codex priority service tier when available.",
		"Codex priority",
		"Use /fast to toggle Fast mode.",
		"Codex AI",
		`function kj(H){return x4()}`,
		`function ip(){return"opus"}`,
		`M="Codex priority",_[1]=M`,
		`return H?` + "`" + `${Z2H(!0)} Fast mode ON \xB7 Codex priority` + "`" + `:"Fast mode OFF"`,
		`function xo_(H=!1){return"Default Codex route \xB7 default Codex work"}`,
		`CLAUDE_LOCAL_OAUTH_API_BASE`,
		`fetch(H+"/api/oauth/usage"`,
		`let j=0,J=Y,M=!uc()&&!K__(K,O),D=!1;`,
		`function Ht7(H){let _=Math.max(0,H)/2000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`,
		"Resume with:\nclaudodex ${O}--resume ${q}\n",
		" resume: claudodex --resume ",
		"Run claudodex --resume to resume a conversation",
		`y_.createElement(V,{bold:!0},"Claudodex`,
		`jq("claude",b)("Claudodex`,
		`"0.1.0 using Claude Code v2.1.159"`,
		`function yr3(H=!1){let _=[],q=yvK();if(q!==void 0)_.push(q);let K=vvK();if(K!==void 0)_.push(K);let O=CvK();if(O!==void 0)_.push(O);return _}function ZX(H){return H===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":H===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":H===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":H}`,
		`j=Aq(),J=(q=ZX(q))??bN_,[M,D]`,
		`function $6_(H=!1){let _=process.env,q=(O,T,$)=>({value:O,label:T,description:$});return[q("opus",_.ANTHROPIC_DEFAULT_OPUS_MODEL_NAME??_.ANTHROPIC_DEFAULT_OPUS_MODEL??"gpt-5.6-sol",_.ANTHROPIC_DEFAULT_OPUS_MODEL_DESCRIPTION??"Default Codex route"),q("sonnet",_.ANTHROPIC_DEFAULT_SONNET_MODEL_NAME??_.ANTHROPIC_DEFAULT_SONNET_MODEL??"gpt-5.6-terra",_.ANTHROPIC_DEFAULT_SONNET_MODEL_DESCRIPTION??"Everyday Codex coding route"),q("haiku",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME??_.ANTHROPIC_DEFAULT_HAIKU_MODEL??"gpt-5.6-luna",_.ANTHROPIC_DEFAULT_HAIKU_MODEL_DESCRIPTION??"Fast Codex coding route")]}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"function kj(H){if(!x4())return!1;",
		` \xB7 model set to ${ip()}`,
		`let j=w,J=Y,M=!uc()&&!K__(K,O),D=!1;`,
		`Math.max(0,H)/1000,q=1-Math.exp(-_/90)`,
		"Resume this session with:\nclaude ${O}--resume ${q}\n",
		"function yr3(H=!1){if(Gq())",
		"function $6_(H=!1){let _=yr3(H)",
		`j=Aq(),J=q===null?bN_:q,[M,D]`,
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("patched data still contains %q:\n%s", notWant, got)
		}
	}
}

func TestApplyClaudeUIPatches159FailsWhenCriticalUsagePatchMissing(t *testing.T) {
	data := []byte(strings.ReplaceAll(string(claude159PatchFixture(t)), "async function UXH()", "async function MISSING_UXH()"))

	if applyClaudeUIPatches_2_1_159(data, "0.1.0", "2.1.159", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_159 succeeded without the critical usage patch target")
	}
}

func TestApplyClaudeUIPatches159FailsWhenCriticalCompactProgressPatchMissing(t *testing.T) {
	data := []byte(strings.ReplaceAll(string(claude159PatchFixture(t)), "function Ht7(H){", "function MISSING_Ht7(H){"))

	if applyClaudeUIPatches_2_1_159(data, "0.1.0", "2.1.159", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_159 succeeded without the critical compact progress patch target")
	}
}

func TestApplyClaudeUIPatches195BrandsHeaderAndModelPicker(t *testing.T) {
	data := claude195PatchFixture(t)

	if !applyClaudeUIPatches_2_1_195(data, "0.1.0", "2.1.195", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_195 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"Experimental - treat it as such.",
		"Set the AI model for Claudodex",
		"WARNING: Claudodex running in Bypass Permissions mode",
		"Codex Plan",
		"Switch between Codex-backed models.",
		"Codex model",
		"default Codex work",
		"gpt-5.6-terra everyday",
		"gpt-5.6-luna quick code",
		`"0.1.0 using Claude Code v2.1.195"`,
		`function CDX195(e){if(e==null||e==="")return"opus";let t=String(e).replace(/(\[1m\])+$/i,"").trim();return t===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":t===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":e}`,
		`function Kap(e){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return iDe([n("opus"`,
		`CLAUDE_LOCAL_OAUTH_API_BASE`,
		`fetch(e+"/api/oauth/usage"`,
		`d=Ao(),p=CDX195(n)??vYt,[m,f]=M1e.useState(p)`,
		`!H.some((Dn)=>Dn.value===m)&&ka(n)`,
		`function sc(){return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`,
		`function W6(){return"Codex AI"}`,
		`function Y$e(){return"opus"}`,
		`f="Codex",t[1]=f`,
		`d="Codex",p=j9o()`,
		`function rh(e){return sc()}`,
		`function Naa(e,t,n){return null}`,
		`claudodex ${o}--resume ${n}`,
		`function CXa(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		"try /model opus",
		"try /model sonnet",
		`Math.max(0,e)/1000,n=1-Math.exp(-t/90)`,
		`d=Ao(),p=n===null?vYt:n,[m,f]=M1e.useState(p)`,
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("patched data still contains %q:\n%s", notWant, got)
		}
	}
}

func TestApplyClaudeUIPatches196UsesVerifiedPatchShape(t *testing.T) {
	data := claude196PatchFixture(t)

	if !applyClaudeUIPatches_2_1_196(data, "0.1.1", "2.1.196", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_196 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		"Claudodex Info",
		"Set the AI model for Claudodex",
		"Codex Plan",
		`"0.1.1 using Claude Code v2.1.196"`,
		`function CDX196(e){if(e==null||e==="")return"opus";let t=String(e).replace(/(\[1m\])+$/i,"").trim();return t===process.env.ANTHROPIC_DEFAULT_OPUS_MODEL?"opus":t===process.env.ANTHROPIC_DEFAULT_SONNET_MODEL?"sonnet":t===process.env.ANTHROPIC_DEFAULT_HAIKU_MODEL?"haiku":e}`,
		`function pmp(e=!1){let t=process.env,n=(r,o,s)=>({value:r,label:o,description:s,descriptionForModel:s});return[n("opus"`,
		`function fmp(e){let t=pmp(e),n=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION,r=CDX196(n);if(n&&r===n&&!t.some((l)=>l.value===n))t.push({value:n,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??` + "`" + `Custom model (${n})` + "`" + `});return t}`,
		`let H=x,P,n2=CDX196(n);P=n2!==null&&!H.some((e)=>e.value===n2)&&Ua(n2)?[...H,{value:n2,label:O9(n2),description:"Current model"}]:H;let O=P,N;`,
		`p=CDX196(p),M=D.some((e)=>e.value===p)?p:D[0]?.value`,
		`function uc(){return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`,
		`function c6(){return"Codex AI"}`,
		`function N9e(){return"opus"}`,
		`function ih(e){return uc()}`,
		`function Pme(){return Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`function eB(){return Pme()||Gs()?.accessToken}`,
		`function Dda(e,t,n){return null}`,
		`function Pw(){return!Yen()&&!W2()}`,
		`async function zGo(){return!Yen()&&!W2()&&!!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		`async function klr(){if(Yen())return"Remote Control is disabled by your organization's policy (managed setting ` + "`" + `disableRemoteControl` + "`" + `).";if(W2())return"Remote Control is not available inside a cloud session.";if(!Ne.CLAUDE_BRIDGE_OAUTH_TOKEN)return"Remote Control requires a normal Claude login. Run ` + "`" + `claude auth login` + "`" + ` outside Claudodex, then restart Claudodex.";return null}`,
		`Resume with:
claudodex `,
		`function Ptl(e){let t=Math.max(0,e)/2000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched data missing %q:\n%s", want, got)
		}
	}
	for _, notWant := range []string{
		`function pmp(e=!1){if(Eo())`,
		`for(let l of BAn())`,
		`availableModels:o`,
		`try /model opus`,
		`try /model sonnet`,
		`Math.max(0,e)/1000,n=1-Math.exp(-t/90)`,
		`D.some((Ee)=>Ee.value===p)?p:D[0]?.value??void 0`,
		`return!W2()&&$Ve()`,
		`await UU("tengu_ccr_bridge")`,
		`Remote Control requires feature-flag evaluation`,
		`function Pme(){return}function Ome(){return}`,
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
	modelOptions154 := `function v__(H=!1){let _=ki3(H),q=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION;if(q&&!_.some((z)=>z.value===q))_.push({value:q,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??q,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??` + "`" + `Custom model (${q})` + "`" + `});let O=null,T=be(),$=_7H();if(T!==void 0&&T!==null)O=T;else if($!==void 0&&$!==null)O=$;if(O===null||_.some((z)=>z.value===O))return MPH(_);else if(O==="opus")return MPH([..._,JNK(!1)]);return MPH(_)}` + strings.Repeat(" ", 900) + `function MPH(H){return H}`
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
		`Fast mode (research preview)`,
		`Draws from usage credits at a higher rate. Separate rate limits apply.`,
		`Billed as extra usage at a premium rate. Separate rate limits apply.`,
		`$10/$50 per Mtok`,
		`$30/$150 per Mtok`,
		`Learn more:`,
		`https://code.claude.com/docs/en/fast-mode`,
		`Use /fast to turn on Fast mode (Opus 4.8).`,
		`Opus 4.8`,
		`X4.createElement(V,{dimColor:!0},"Use ",X4.createElement(V,{bold:!0},"/fast")," to turn on Fast mode (",Bp(),").")`,
		`X4.createElement(B,{marginBottom:1},X4.createElement(V,{dimColor:!0},"Fast mode is ",X4.createElement(V,{bold:!0},"ON")," and available with"," ",Bp()," (/fast). Switching to other models turns off fast mode."))`,
		`function Pj(H){if(!C4())return!1;let _=H??M0(),K=e7(_).toLowerCase();if(hi())return K.includes("opus-4-6");return K.includes("opus-4-6")||K.includes("opus-4-7")||K.includes("opus-4-8")}`,
		`function xUH(){return(hi()?"claude-opus-4-6":"opus")+(VP()?"[1m]":"")}`,
		`let S=f7(),x=A7(S).includes("opus")?S:"claude-opus-4-8";M=Yk(vGH(!0,x)),_[1]=M`,
		`async function Lv6(H,_,q,K){let O=Ee();if(O)return` + "`" + `Fast mode unavailable: ${O}` + "`" + `;let{mainLoopModel:T}=_();if(Rv6(H,q),d("tengu_fast_mode_toggled",{enabled:H,source:K}),H){let $=T2H(!0),z=!Pj(T)?` + "`" + ` \xB7 model set to ${Bp()}` + "`" + `:"",Y=f7(),A=A7(Y).includes("opus")?Y:"claude-opus-4-8",w=Yk(vGH(!0,A));return` + "`" + `${$} Fast mode ON${z} \xB7 ${w}` + "`" + `}else return"Fast mode OFF"}var oHq=R(`,
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
		`function TCH(H){let _=yo8.c(102),{initial:q,sessionModel:K,onSelect:O,onSetDefault:T,onCancel:$,isStandaloneCommand:z,showFastModeNotice:Y,headerText:A,skipSettingsWrite:w}=H,j=$q(),J=q===null?KN_:q,[M,D]=DPH.useState(J)}`,
		modelOptions154,
	}, "\n"))
}

func claude159PatchFixture(t *testing.T) []byte {
	t.Helper()
	logo159 := `function K8_(){let H=process.env.DEMO_VERSION??` + "`" + `${{ISSUES_EXPLAINER:"report the issue at https://github.com/anthropics/claude-code/issues",PACKAGE_URL:"@anthropic-ai/claude-code",README_URL:"https://code.claude.com/docs/en/overview",VERSION:"2.1.159",FEEDBACK_CHANNEL:"https://github.com/anthropics/claude-code/issues",BUILD_TIME:"2026-05-31T16:22:50Z",GIT_SHA:"dd8c11fc8d05cea0b2b9fc8f5a99a5c5c5dffc9b"}.VERSION}${uS()}` + "`" + `,_=nu6(),q=process.env.DEMO_VERSION?"/code/claude":G5(C_()),K=bH(process.env.CLAUDE_CODE_HIDE_CWD)?"":_?` + "`" + `${q} in ${_.replace(/^https?:\/\//,"")}` + "`" + `:q,O=Zq(),T=O!=="firstParty"?yAH[O]:Gq()?H_6():"API Usage Billing",$=U8().agent;return{version:H,cwd:K,billingType:T,agentName:$}}                                                                                                                                                                                                                                                                                                                                                                               function PBK(H,_,q){}`
	defaultDescription159 := `function xo_(H=!1){if(te()||FAH()||AFH()){let q=QZ(),K=oM(aM(q))??"Opus",O=H&&kj(q);if(yP())return` + "`" + `${K} with 1M context \xB7 Most capable for complex work${O?cKH(!0,q):""}` + "`" + `;return` + "`" + `${K} \xB7 Most capable for complex work${O?cKH(!0,q):""}` + "`" + `}return` + "`" + `${oM(aM(mN()))??"Sonnet"} \xB7 Best for everyday tasks` + "`" + `}`
	modelOptions159 := `function $6_(H=!1){let _=yr3(H),q=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION;if(q&&!_.some((z)=>z.value===q))_.push({value:q,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??q,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??` + "`" + `Custom model (${q})` + "`" + `});for(let z of E_().additionalModelOptionsCache??[])if(!_.some((Y)=>Y.value===z.value))_.push(z);let O=null,T=ie(),$=f7H();if(T!==void 0&&T!==null)O=T;else if($!==void 0&&$!==null)O=$;if(O===null||_.some((z)=>z.value===O))return T6_(_);else if(O==="opus")return T6_([..._,mvK(!1)]);else{let z=Er3(O);if(z)_.push(z);else _.push({value:O,label:O,description:"Custom model"});return T6_(_)}}` + strings.Repeat(" ", 900) + `function T6_(H){return H}`
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
		`Fast mode (research preview)`,
		`Draws from usage credits at a higher rate. Separate rate limits apply.`,
		`Billed as extra usage at a premium rate. Separate rate limits apply.`,
		`$10/$50 per Mtok`,
		`Learn more:`,
		`https://code.claude.com/docs/en/fast-mode`,
		`Use /fast to turn on Fast mode (Opus 4.8).`,
		`Opus 4.8`,
		`y_.createElement(V,{bold:!0},"Claude Code")`,
		`y_.createElement(V,{dimColor:!0},"v",zg)`,
		`jq("claude",b)("Claude Code")`,
		`jq("inactive",b)(` + "`v${X}`" + `)`,
		`jq("claude",b)(" Claude Code ")`,
		logo159,
		defaultDescription159,
		`async function UXH(){return ZK("api_usage_fetch",async()=>{if(!Gq()||!wk())return{};let H=0,_=await _G(async()=>{H++,N(` + "`" + `fetchUtilization: GET /api/oauth/usage (attempt ${H})` + "`" + `);let q=await B7.get("/api/oauth/usage",{timeout:5000,headers:{"Content-Type":"application/json"},refreshOAuth:!0});if(!q.ok)throw Error(` + "`" + `Auth error: ${q.reason==="no-auth"?q.detail:q.reason}` + "`" + `);return q});return N(` + "`" + `fetchUtilization: 200 after ${H} attempt(s)${H>1?" (401\u2192refresh\u2192retry succeeded)":""}` + "`" + `),_.data})}`,
		`var zL_=R(()=>{Yq();cH();e2();$6();Mf()});`,
		`function hBK(H){let _=H.map((K)=>{return{text:K}}),q="Check the Claude Code changelog for updates";return{title:"What's new",lines:_,footer:_.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Use ",W4.createElement(V,{bold:!0},"/fast")," to turn on Fast mode (",ip(),")."))`,
		`W4.createElement(B,{marginBottom:1},W4.createElement(V,{dimColor:!0},"Fast mode is ",W4.createElement(V,{bold:!0},"ON")," and available with"," ",ip()," (/fast). Switching to other models turns off fast mode."))`,
		`function kj(H){if(!x4())return!1;let _=H??Z0(),K=qK(_).toLowerCase();if(ri())return K.includes("opus-4-6");return K.includes("opus-4-6")||K.includes("opus-4-7")||K.includes("opus-4-8")}`,
		`function ip(){return ri()?"Opus 4.6":"Opus 4.8"}`,
		`let C=A7(),x=n9(C).includes("opus")?C:"claude-opus-4-8";M=Jk(gGH(!0,x)),_[1]=M`,
		`async function Ah6(H,_,q,K){let O=ce();if(O)return` + "`" + `Fast mode unavailable: ${O}` + "`" + `;let{mainLoopModel:T}=_();if(Yh6(H,q),d("tengu_fast_mode_toggled",{enabled:H,source:K}),H){let $=Z2H(!0),z=!kj(T)?` + "`" + ` \xB7 model set to ${ip()}` + "`" + `:"",Y=A7(),A=n9(Y).includes("opus")?Y:"claude-opus-4-8",w=Jk(gGH(!0,A));return` + "`" + `${$} Fast mode ON${z} \xB7 ${w}` + "`" + `}else return"Fast mode OFF"}var i_q=R(`,
		`function h54(H){let _=v54.c(15),{tokenUsage:q,model:K}=H,O=f_(mGO),T;if(_[0]!==O||_[1]!==K||_[2]!==q)T=uSH(q,K,O),_[0]=O,_[1]=K,_[2]=q,_[3]=T;else T=_[3];let $=T,z=BS6();if($.level==="ok"||z)return null;let Y=$.pctLeft,A=RG(),w;if(_[4]===Symbol.for("react.memo_cache_sentinel"))w=$oH("warning"),_[4]=w;else w=_[4];let j=w,J=Y,M=!uc()&&!K__(K,O),D=!1;if(M||D){let G=NqH(K,O),W;if(_[5]!==G||_[6]!==q)W=Math.round((G-q)/G*100),_[5]=G,_[6]=q,_[7]=W;else W=_[7];J=Math.max(0,W)}let f=M?` + "`" + `${100-J}% context used` + "`" + `:` + "`" + `${J}% until auto-compact` + "`" + `;if(A)return null;return null}function mGO(H){return H.autoCompactWindow}`,
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
		`function Ht7(H){let _=Math.max(0,H)/1000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`,
		`function yr3(H=!1){if(Gq()){if(te()||FAH()||AFH()){let $=[mk6(H)];if(!yP()&&Xa()&&!Ma8())$.push(IvK());if($.push(vr3),J3H())$.push(bvK());return $.push(xvK),$}let T=[mk6(H)];if(J3H())T.push(bvK());if(yP())T.push(pvK(!1));else if(T.push(mvK(!1)),Xa()&&!Ma8())T.push(IvK());return T.push(xvK),T}if(l$()){let T=[mk6(H)],$=yvK();if($!==void 0)T.push($);else if(!yP()&&Xa()&&!Ma8())T.push(SvK(H));let z=vvK();if(z!==void 0)T.push(z);else if(T.push(hvK()),J3H())T.push(EvK());return T.push(CvK()??uvK()),T}let _=[mk6(H)],q=vvK();if(q!==void 0)_.push(q);else if(_.push(hvK()),J3H())_.push(EvK());let K=yvK();if(K!==void 0)_.push(K);else{if(_.push(Wr3()),_.push(Rr3()),Xa()&&!re(wO().opus48))_.push(SvK());if(_.push(Gr3()),Xa()&&!re(wO().opus47))_.push(kr3());if(_.push(Zr3()),Xa())_.push(Lr3(H))}let O=CvK();if(O!==void 0)_.push(O);else _.push(Nr3());return _}function Er3(H){return null}`,
		`function VCH(H){let _=Ra8.c(102),{initial:q,sessionModel:K,onSelect:O,onSetDefault:T,onCancel:$,isStandaloneCommand:z,showFastModeNotice:Y,headerText:A,skipSettingsWrite:w}=H,j=Aq(),J=q===null?bN_:q,[M,D]=bPH.useState(J)}`,
		modelOptions159,
	}, "\n"))
}

func claude195PatchFixture(t *testing.T) []byte {
	t.Helper()
	return []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Welcome back!`,
		`Set the AI model for Claude Code`,
		`Claude Code will be able to read files in this directory and make edits when auto-accept edits is on.`,
		`WARNING: Claude Code running in Bypass Permissions mode`,
		`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`,
		`No, exit Claude Code`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`Best for everyday, complex tasks`,
		`Efficient for routine tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`Fast mode (research preview)`,
		`Draws from usage credits at a higher rate. Separate rate limits apply.`,
		`Billed as extra usage at a premium rate. Separate rate limits apply.`,
		`$10/$50 per Mtok`,
		`Learn more:`,
		`https://code.claude.com/docs/en/fast-mode`,
		`/upgrade to keep using Claude Code`,
		`_b.jsx(v,{bold:!0,children:"Claude Code"})`,
		`xo("claude",O)("Claude Code")`,
		`xo("claude",O)(" Claude Code ")`,
		`children:["Claude Code"," "]`,
		`function pEt(){` + strings.Repeat(" ", 900) + `function ejl(e,t,n){}`,
		`function djl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function Vue(){return _l("api_usage_fetch",async()=>{if(!To()||!dx())return{};let e=0,t=await iD(async()=>{e++,C(` + "`" + `fetchUtilization: GET /api/oauth/usage (attempt ${e})` + "`" + `);let n=await Ns.get("/api/oauth/usage",{timeout:5000,headers:{"Content-Type":"application/json"},refreshOAuth:!0});if(!n.ok)throw Error(` + "`" + `Auth error: ${n.reason==="no-auth"?n.detail:n.reason}` + "`" + `);return n});return C(` + "`" + `fetchUtilization: 200 after ${e} attempt(s)${e>1?" (401\u2192refresh\u2192retry succeeded)":""}` + "`" + `),t.data})}` + strings.Repeat(" ", 400) + `var olp="tengu_usage_overage_included_models";`,
		`function Kap(e){` + strings.Repeat(" ", 2400) + `function zap(e){}`,
		`d=Ao(),p=n===null?vYt:n,[m,f]=M1e.useState(p)`,
		`!H.some((Dn)=>Dn.value===n)&&ka(n)`,
		`function sc(){if(mr()!=="firstParty")return!1;return!ut(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`,
		`function W6(){return"Opus 4.8"}`,
		`function Y$e(){return"opus"+(oC()?"[1m]":"")}`,
		`f=oU(znt(!0,F)),t[1]=f`,
		`d=oU(znt(!0,u)),p=j9o()`,
		`function rh(e){if(!sc())return!1;let t=e??$v(),n=Ko(t);if(tU(fo(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-6")||r.includes("opus-4-7")||r.includes("opus-4-8")}`,
		`function Naa(e,t,n){if(Pi()!=="pro")return null;if(e.rateLimitType!=="seven_day")return null;if(t.includes("fable"))return{lever:"model",text:"try /model opus \xB7 more runway"};if(t.includes("opus"))return{lever:"model",text:"try /model sonnet \xB7 ~2\xD7 runway"};if(!Yv(t))return null;let r=PL(t,n);if(r==="high"||r==="xhigh"||r==="max")return{lever:"effort",text:"try /effort medium"};return null}function Hio(e,t){}`,
		`function _go(){` + strings.Repeat(" ", 900) + `function ygo(e){}`,
		`Run claude --continue or claude --resume to resume a conversation`,
		`Open ` + "`" + `claude agents` + "`" + ` to attach to it, or stop it there first to resume here.`,
		`). Use ` + "`" + `claude agents` + "`" + ` to find and attach to it, or add --fork-session to branch off a copy.`,
		`function CXa(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`,
	}, "\n"))
}

func claude196PatchFixture(t *testing.T) []byte {
	t.Helper()
	return []byte(strings.Join([]string{
		`Check the Claude Code changelog for updates`,
		`What's new`,
		`Welcome back!`,
		`Set the AI model for Claude Code`,
		`Claude Code will be able to read files in this directory and make edits when auto-accept edits is on.`,
		`WARNING: Claude Code running in Bypass Permissions mode`,
		`In Bypass Permissions mode, Claude Code will not ask for your approval before running potentially dangerous commands.`,
		`No, exit Claude Code`,
		`Claude Max`,
		`Switch between Claude models. Your pick becomes the default for new sessions. For other/previous model names, specify with --model.`,
		`Select model`,
		`Default (recommended)`,
		`best for everyday, complex tasks`,
		`efficient for routine tasks`,
		`Fastest for quick answers`,
		` with 1M context \xB7 `,
		`Fast mode (research preview)`,
		`Draws from usage credits at a higher rate. Separate rate limits apply.`,
		`Billed as extra usage at a premium rate. Separate rate limits apply.`,
		`Draws from usage credits`,
		`$10/$50 per Mtok`,
		`$30/$150 per Mtok`,
		`Learn more:`,
		`https://code.claude.com/docs/en/fast-mode`,
		`/upgrade to keep using Claude Code`,
		`children:"Claude Code"`,
		`("Claude Code")`,
		`(" Claude Code ")`,
		`function yAt(){` + strings.Repeat(" ", 1200) + `function AGl(e,t,n){}`,
		`function OGl(e){let t=e.map((r)=>({text:r})),n="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function bde(){return Cl("api_usage_fetch",async()=>{if(!Eo()||!hx())return{};let e=0,t=await gD(async()=>{e++,C(` + "`" + `fetchUtilization: GET /api/oauth/usage (attempt ${e})` + "`" + `);let n=await Hs.get("/api/oauth/usage",{timeout:5000,headers:{"Content-Type":"application/json"},refreshOAuth:!0});if(!n.ok)throw Error(` + "`" + `Auth error: ${n.reason==="no-auth"?n.detail:n.reason}` + "`" + `);return n});return C(` + "`" + `fetchUtilization: 200 after ${e} attempt(s)${e>1?" (401\u2192refresh\u2192retry succeeded)":""}` + "`" + `),t.data})}` + strings.Repeat(" ", 400) + `var Cmp="tengu_usage_overage_included_models";`,
		`function pmp(e=!1){if(Eo()){if(zle()||Bke()||Y_e()){let a=[hBn(e)];if(!uC()&&Yre()&&!Ilo())a.push(Jua());if(a.push(sda),JSe())a.push(Yua());return a.push(Xua),Dlo(a,e)}let i=[hBn(e)];if(JSe())i.push(Yua());if(uC())i.push(oda());else if(i.push(Flo(!1)),Yre()&&!Ilo())i.push(Jua());return i.push(Xua),Dlo(i,e)}if(Ed()){let i=[hBn(e)],a=Gua();if(a!==void 0)i.push(a);else if(!uC()&&Yre()&&!Ilo())i.push(Kua(e));let l=jua();if(l!==void 0)i.push(l);else if(i.push(Plo()),JSe())i.push(Vua());i.push(zua()??nda());let c=Wua();if(c!==void 0)Jut(i,c);else if(_r()==="anthropicAws"&&QSe("fable5"))Jut(i,Olo());return Dlo(i,e)}let t=[hBn(e)],n=jua();if(n!==void 0)t.push(n);else if(QSe("sonnet46")){if(t.push(Plo()),JSe())t.push(Vua())}let r=Gua();if(r!==void 0)t.push(r);else{if(QSe("opus41"))t.push(omp());if(QSe("opus48")){if(t.push(tda()),Yre()&&!CU(Yp().opus48))t.push(Kua())}if(QSe("opus47")){if(t.push(imp()),Yre()&&!CU(Yp().opus47))t.push(lmp())}if(QSe("opus46")){if(t.push(smp()),Yre())t.push(amp(e))}}let o=zua();if(o!==void 0)t.push(o);else if(QSe("haiku45")||QSe("haiku35"))t.push(ump());let s=Wua();if(s!==void 0||QSe("fable5"))Jut(t,s??Olo());return t}` + strings.Repeat(" ", 900) + `function QSe(e){return true}`,
		`function fmp(e){let t=pmp(e),n=process.env.ANTHROPIC_CUSTOM_MODEL_OPTION;if(n&&!t.some((l)=>l.value===n))t.push({value:n,label:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_NAME??n,description:process.env.ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION??` + "`" + `Custom model (${n})` + "`" + `});for(let l of BAn())if(!t.some((c)=>gBn(c,l)))Jut(t,l);let r=_r();if(r==="firstParty"||r==="gateway"){let l=r==="gateway"||bu();for(let c of V_e()){if(c.disabled&&!l)continue;if(!t.some((u)=>gBn(u,c)))Jut(t,c)}}let{availableModels:o}=es()??{};if(o)for(let l of o){let c=l.trim();if(!c.startsWith("anthropic.")||t.some((u)=>u.value===c))continue;t.push({value:c,label:c,description:"Custom model"})}return VDe(t)}` + strings.Repeat(" ", 900) + `function VDe(e){return e}`,
		`let H=x,P;if(t[5]!==n||t[6]!==H){e:{if(n!==null&&!H.some((Ee)=>Ee.value===n)&&Ua(n)){let Ee={value:n,label:O9(n),description:"Current model"},we=H.findIndex(Z1m);if(we===-1){P=[...H,Ee];break e}P=[...H.slice(0,we),Ee,...H.slice(we)];break e}P=H}t[5]=n,t[6]=H,t[7]=P}else P=t[7];let O=P,N;`,
		`if(t[13]!==p||t[14]!==D)M=D.some((Ee)=>Ee.value===p)?p:D[0]?.value??void 0,t[13]=p,t[14]=D,t[15]=M;else M=t[15];`,
		`function uc(){if(_r()!=="firstParty")return!1;return!ct(process.env.CLAUDE_CODE_DISABLE_FAST_MODE)}`,
		`function c6(){return"Opus 4.8"}`,
		`function N9e(){return"opus"+(uC()?"[1m]":"")}`,
		`function ih(e){if(!uc())return!1;let t=e??Wv(),n=Wo(t);if(mU(io(n),"fast_mode"))return!0;let r=n.toLowerCase();return r.includes("opus-4-7")||r.includes("opus-4-8")}`,
		`function Pme(){return}function Ome(){return}function eB(){let e=Pme();if(e!==void 0)return e;if(!kc()||!Eo())return;return Gs()?.accessToken}function j7t(){return Ome()??Fs().BASE_API_URL}`,
		`function Pw(){if(mdr())return!0;if(Yen())return!1;return!W2()&&$Ve()}`,
		`async function zGo(){if(mdr())return!0;if(Yen())return!1;return zen()&&!W2()&&aRt()&&await UU("tengu_ccr_bridge")}`,
		`async function klr(){if(mdr())return null;if(!zen())return"Remote Control is only available when using Claude via api.anthropic.com.";if(W2())return"Remote Control is not available inside a cloud session.";if(Yen())return"Remote Control is disabled by your organization's policy (managed setting ` + "`" + `disableRemoteControl` + "`" + `).";if(!fdr())return"Remote Control requires a claude.ai subscription. Run ` + "`" + `claude auth login` + "`" + ` to sign in with your claude.ai account.";if(!L6()){let t=Kmn();if(t)return` + "`" + `Remote Control requires feature-flag evaluation, which is disabled because ${t} is set. Unset it (or run in a shell without it) to use Remote Control.` + "`" + `;return"Remote Control requires feature-flag evaluation, which is unavailable in this environment."}if(!await UU("tengu_ccr_bridge"))return"Remote Control is not yet enabled for your account.";return null}function ddf(){return""}`,
		`function Dda(e,t,n){if(Pi()!=="pro")return null;if(e.rateLimitType!=="seven_day")return null;if(t.includes("fable"))return{lever:"model",text:"try /model opus \xB7 more runway"};if(t.includes("opus"))return{lever:"model",text:"try /model sonnet \xB7 ~2\xD7 runway"};if(!Qv(t))return null;let r=tM(t,n);if(r==="high"||r==="xhigh"||r==="max")return{lever:"effort",text:"try /effort medium"};return null}function Vlo(e,t){}`,
		`
Resume this session with:
claude `,
		`Previous session saved \xB7 resume with: claude --resume `,
		`Run claude --continue or claude --resume to resume a conversation`,
		`Open ` + "`" + `claude agents` + "`" + ` to attach to it, or stop it there first to resume here.`,
		`). Use ` + "`" + `claude agents` + "`" + ` to find and attach to it, or add --fork-session to branch off a copy.`,
		`function Ptl(e){let t=Math.max(0,e)/1000,n=1-Math.exp(-t/90);return Math.min(95,Math.round(n*100))}`,
	}, "\n"))
}

func TestFindClaudeUIPatchRequiresVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.212", "09ecba2ab2df9b6ee5b0695e26f65dea60fb3b6af3d3542ee09f466838d1e574")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.212 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.211", "5a728a76198b6eca7f3c7cdbff43bab44b77b48c2108f7a3107d889773382629")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.211 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.209", "59d2de7f49db2f75d5c33bbb46a6b8f288ad24d40b61e30602a502bb7ddc380c")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.209 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.208", "051c7f28871b158132ac03a6140f2f2ab4046b18ecc4f7a91a2ac4d54774551e")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.208 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.207", "1397a062c6889675055e3314dd956376ac51262a7734ad9e819c26975d71547a")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.207 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.198", "ab6f7ee109816ede414f7c285446633f805b623aa609f425609a64266451d61e")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.198 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.153", "449d9c89d7a63b1d427d912a7bd6e6f23f9a7b363866697c9fa9a0012546b254")
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
	patch = findClaudeUIPatch("2.1.159", "5adf7b4d349f743d669cd5adf2ce76dbb5e146d8ab99b3a63c5aef2ef15595f9")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.159 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.195", "8b45adad93f336ab95f33e714494b19fd3377a494eb05c122c8677bc895876ad")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.195 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.196", "6fc6e61ab7582c2bf241225ff90d9f79e91d69380cb9589fc9dedd3a30070f5a")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.196 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	patch = findClaudeUIPatch("2.1.197", "8cc0c4d1e4eb1dca3b0cc92ab02ee3505de764e023f8c901761c167b72041fb8")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected local verified 2.1.197 darwin/arm64 patch to match")
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
	if got := findClaudeUIPatch("2.1.159", "9c1e8601031f5cbb3101e49dda22bf8ba31183692c705e267a6923585fa2ba09"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.195", "5adf7b4d349f743d669cd5adf2ce76dbb5e146d8ab99b3a63c5aef2ef15595f9"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.196", "8b45adad93f336ab95f33e714494b19fd3377a494eb05c122c8677bc895876ad"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.197", "6fc6e61ab7582c2bf241225ff90d9f79e91d69380cb9589fc9dedd3a30070f5a"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.198", "8cc0c4d1e4eb1dca3b0cc92ab02ee3505de764e023f8c901761c167b72041fb8"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.207", "ab6f7ee109816ede414f7c285446633f805b623aa609f425609a64266451d61e"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.208", "1397a062c6889675055e3314dd956376ac51262a7734ad9e819c26975d71547a"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.209", "051c7f28871b158132ac03a6140f2f2ab4046b18ecc4f7a91a2ac4d54774551e"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.211", "59d2de7f49db2f75d5c33bbb46a6b8f288ad24d40b61e30602a502bb7ddc380c"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.212", "5a728a76198b6eca7f3c7cdbff43bab44b77b48c2108f7a3107d889773382629"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.153", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); got != nil {
		t.Fatalf("patch matched unsupported sha: %#v", got)
	}
}

func TestClaude209UIBrandingReplacements(t *testing.T) {
	for _, replacement := range claude209UIBrandingReplacements {
		replacement := replacement
		t.Run(replacement.old, func(t *testing.T) {
			data := []byte(strings.Repeat(replacement.old+"\x00", replacement.expectedCount))
			originalLength := len(data)
			if !validateClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
				t.Fatal("valid exact-count replacement failed validation")
			}
			if !applyClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
				t.Fatal("valid exact-count replacement was not applied")
			}
			if len(data) != originalLength {
				t.Fatalf("patched length = %d, want %d", len(data), originalLength)
			}
			if bytes.Contains(data, []byte(replacement.old)) {
				t.Fatalf("patched data retained old literal %q", replacement.old)
			}
			if !bytes.Contains(data, []byte(replacement.replacement)) {
				t.Fatalf("patched data missing replacement %q", replacement.replacement)
			}
		})
	}
}

func TestClaude211UIBrandingReplacements(t *testing.T) {
	for _, replacement := range claude211UIBrandingReplacements {
		data := []byte(strings.Repeat(replacement.old+"\x00", replacement.expectedCount))
		originalLength := len(data)
		if !validateClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
			t.Fatalf("valid exact-count replacement failed validation: %q", replacement.old)
		}
		if !applyClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
			t.Fatalf("valid exact-count replacement was not applied: %q", replacement.old)
		}
		if len(data) != originalLength {
			t.Fatalf("patched length = %d, want %d for %q", len(data), originalLength, replacement.old)
		}
	}
}

func TestClaude212UIBrandingReplacements(t *testing.T) {
	for _, replacement := range claude212UIBrandingReplacements {
		data := []byte(strings.Repeat(replacement.old+"\x00", replacement.expectedCount))
		originalLength := len(data)
		if !validateClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
			t.Fatalf("valid exact-count replacement failed validation: %q", replacement.old)
		}
		if !applyClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
			t.Fatalf("valid exact-count replacement was not applied: %q", replacement.old)
		}
		if len(data) != originalLength {
			t.Fatalf("patched length = %d, want %d for %q", len(data), originalLength, replacement.old)
		}
	}
}

func TestClaude209UIBrandingReplacementsApplyTogether(t *testing.T) {
	var data []byte
	for _, replacement := range claude209UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("earlier fixtures overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	originalLength := len(data)
	if !validateClaude209UIBrandingReplacements(data, claude209UIBrandingReplacements) {
		t.Fatal("complete exact-count replacement table failed validation")
	}
	if !applyClaude209UIBrandingReplacements(data, claude209UIBrandingReplacements) {
		t.Fatal("complete exact-count replacement table was not applied")
	}
	if len(data) != originalLength {
		t.Fatalf("patched length = %d, want %d", len(data), originalLength)
	}
	for _, replacement := range claude209UIBrandingReplacements {
		if bytes.Contains(data, []byte(replacement.old)) {
			t.Fatalf("patched data retained old literal %q", replacement.old)
		}
		if !bytes.Contains(data, []byte(replacement.replacement)) {
			t.Fatalf("patched data missing replacement %q", replacement.replacement)
		}
	}
}

func TestClaude209UIBrandingReplacementCountsFailClosed(t *testing.T) {
	for _, replacement := range claude209UIBrandingReplacements {
		replacement := replacement
		t.Run(replacement.old, func(t *testing.T) {
			tests := []struct {
				name  string
				count int
			}{
				{name: "missing", count: 0},
				{name: "extra", count: replacement.expectedCount + 1},
			}
			if replacement.expectedCount > 1 {
				tests = append(tests, struct {
					name  string
					count int
				}{name: "partial", count: replacement.expectedCount - 1})
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					data := []byte(strings.Repeat(replacement.old+"\x00", tt.count))
					original := append([]byte(nil), data...)
					if validateClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
						t.Fatalf("count %d unexpectedly passed validation", tt.count)
					}
					if !bytes.Equal(data, original) {
						t.Fatal("failed validation mutated input")
					}
				})
			}
		})
	}
}

func TestClaude209UIBrandingRejectsNonFixedLengthReplacement(t *testing.T) {
	replacement := claude209UIBrandingReplacement{old: "Claude", replacement: "Claudodex", expectedCount: 1}
	data := []byte(replacement.old)
	if validateClaude209UIBrandingReplacements(data, []claude209UIBrandingReplacement{replacement}) {
		t.Fatal("replacement longer than its exact binary slot passed validation")
	}
}

func TestClaude207ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Beh(e=!1){` + strings.Repeat(" ", 2400) + `function XSe(e){}`)
	if !patchModelPickerOptions_2_1_207(data) {
		t.Fatal("patchModelPickerOptions_2_1_207 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched picker missing %q", want)
		}
	}
	if strings.Contains(got, `n("fable",`) || strings.Contains(got, "ANTHROPIC_DEFAULT_FABLE_MODEL") {
		t.Fatalf("patched picker retained a Fable tier:\n%s", got)
	}
}

func TestClaude212ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function fkh(e=!1){` + strings.Repeat(" ", 2400) + `function mCe(e){}`)
	if !patchModelPickerOptions_2_1_212(data) {
		t.Fatal("patchModelPickerOptions_2_1_212 reported no changes")
	}
	got := string(data)
	for _, want := range []string{
		`n("opus",`, `??"gpt-5.6-sol"`,
		`n("sonnet",`, `??"gpt-5.6-terra"`,
		`n("haiku",`, `??"gpt-5.6-luna"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched picker missing %q", want)
		}
	}
	if tiers := strings.Count(got, `n("`); tiers != 3 {
		t.Fatalf("patched picker tier count = %d, want 3:\n%s", tiers, got)
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q:\n%s", forbidden, got)
		}
	}
}

func TestApplyClaudeUIPatches212RequiresAndAppliesEveryTransformation(t *testing.T) {
	transformations := []struct {
		name  string
		apply func([]byte) bool
	}{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_212(data, "0.1.0", "2.1.212") }},
		{"whats-new", patchWhatsNewFeedFunction_2_1_212},
		{"usage", patchUsageFetchFunction_2_1_212},
		{"model-options", patchModelPickerOptions_2_1_212},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_212},
		{"model-selection", patchModelPickerSelectionValue_2_1_212},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_212},
		{"fast-mode-pricing", patchFastModePricing_2_1_212},
		{"context-warning", patchContextWarningHint_2_1_212},
		{"resume-hints", patchResumeCommandHints_2_1_212},
		{"compact-progress", patchCompactProgressCurve_2_1_212},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_212},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude212UIBrandingReplacements)
		}},
	}
	for _, transformation := range transformations {
		t.Run("transformation/"+transformation.name, func(t *testing.T) {
			fixture := claude212PatchFixture(t)
			if !transformation.apply(fixture) {
				if transformation.name == "resume-hints" {
					for _, old := range []string{
						"\nResume this session with:\nclaude ",
						"Previous session saved \xB7 resume with: claude --resume ",
						"Run claude --continue or claude --resume to resume a conversation",
						"Open `claude agents` to attach to it, or stop it there first to resume here.",
						"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
					} {
						t.Logf("remaining resume target count for %q: %d", old, bytes.Count(fixture, []byte(old)))
					}
				}
				t.Fatalf("required %s transformation did not match complete fixture", transformation.name)
			}
		})
	}

	data := claude212PatchFixture(t)
	if !applyClaudeUIPatches_2_1_212(data, "0.1.0", "2.1.212", modelconfig.Default()) {
		t.Fatal("applyClaudeUIPatches_2_1_212 reported no changes for a complete fixture")
	}
	got := string(data)
	for _, want := range []string{
		`"0.1.0 using Claude Code v2.1.212"`,
		"Claudodex Info",
		"Thank you for using Claudodex!",
		"CLAUDE_LOCAL_OAUTH_API_BASE",
		`n("opus",`,
		`??"gpt-5.6-sol"`,
		`n("sonnet",`,
		`??"gpt-5.6-terra"`,
		`n("haiku",`,
		`??"gpt-5.6-luna"`,
		`function gS(e){return ul()}`,
		`function o9e(e){return"Codex priority"}`,
		`function IVi(e,t,r){return null}`,
		"Run claudodex --resume to resume a conversation",
		`Math.max(0,e)/2000`,
		`function w0(){return!!Z.CLAUDE_BRIDGE_OAUTH_TOKEN}`,
		"Welcome to Claudodex",
		"Codex wants to exit plan mode",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched fixture missing %q", want)
		}
	}
	if tiers := strings.Count(got, `n("`); tiers != 3 {
		t.Fatalf("patched picker tier count = %d, want 3", tiers)
	}
	for _, forbidden := range []string{`n("fable",`, `n("mythos",`, "ANTHROPIC_DEFAULT_FABLE_MODEL", "Fable 5", "Mythos 5"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("patched fixture retained forbidden fourth-tier marker %q", forbidden)
		}
	}

	requiredTargets := []string{
		"function Omt(){",
		"function yaa(e){",
		"async function rwe(){",
		"function fkh(e=!1){",
		"function gkh(e){",
		`a2y=lBe===null?ize:Xki(cBe,lBe)??lBe`,
		`function ul(){if(xn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function o6(){return"Opus 4.8"}`,
		`function vvt(){return"opus"+(C1()?"[1m]":"")}`,
		`function gS(e){if(!ul())return!1;let t=e??OF(),r=oi(t);if(D9(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`,
		"function o9e(e){return`${$oc(e.inputTokens)}/${$oc(e.outputTokens)} per Mtok`}",
		"function IVi(e,t,r){",
		"\nResume this session with:\nclaude ",
		"Previous session saved \xB7 resume with: claude --resume ",
		"Run claude --continue or claude --resume to resume a conversation",
		"Open `claude agents` to attach to it, or stop it there first to resume here.",
		"). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.",
		`function ZWd(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function w0(){if(Mwo())return!0;if(dPt())return!1;return!B6()&&VRt()}`,
		`async function Zks(){if(Mwo())return!0;if(dPt())return!1;return uPt()&&!B6()&&ccr()&&await Oj("tengu_ccr_bridge")}`,
		"async function Nwo(){",
		claude212UIBrandingReplacements[0].old,
	}
	for _, target := range requiredTargets {
		t.Run(target, func(t *testing.T) {
			fixture := string(claude212PatchFixture(t))
			broken := []byte(strings.Replace(fixture, target, "MISSING_PATCH_TARGET", 1))
			if applyClaudeUIPatches_2_1_212(broken, "0.1.0", "2.1.212", modelconfig.Default()) {
				t.Fatalf("patch succeeded without required target %q", target)
			}
		})
	}
}

func claude212PatchFixture(t *testing.T) []byte {
	t.Helper()
	parts := []string{
		`function Omt(){` + strings.Repeat(" ", 900) + `function Xsa(e,t,r){}`,
		`function yaa(e){let t=e.map((n)=>({text:n})),r="Check the Claude Code changelog for updates";return{title:"What's new",lines:t,footer:t.length>0?"/release-notes for more":void 0,emptyMessage:"Check the Claude Code changelog for updates"}}`,
		`async function rwe(){` + strings.Repeat(" ", 1000) + `var OEg="fixture";`,
		`function fkh(e=!1){` + strings.Repeat(" ", 1800) + `function mCe(e){}`,
		`function gkh(e){` + strings.Repeat(" ", 2200) + `function Xki(e,t){}`,
		`a2y=lBe===null?ize:Xki(cBe,lBe)??lBe`,
		`function ul(){if(xn()!=="firstParty")return!1;return!Z.CLAUDE_CODE_DISABLE_FAST_MODE}`,
		`function o6(){return"Opus 4.8"}`,
		`function vvt(){return"opus"+(C1()?"[1m]":"")}`,
		`function gS(e){if(!ul())return!1;let t=e??OF(),r=oi(t);if(D9(lo(r),"fast_mode"))return!0;let n=r.toLowerCase();return n.includes("opus-4-7")||n.includes("opus-4-8")}`,
		"function o9e(e){return`${$oc(e.inputTokens)}/${$oc(e.outputTokens)} per Mtok`}",
		`function IVi(e,t,r){` + strings.Repeat(" ", 600) + `function DQn(e,t){}`,
		strings.Repeat("\nResume this session with:\nclaude ", 2),
		"Previous session saved \xB7 resume with: claude --resume ",
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\x00", 2),
		strings.Repeat("Open `claude agents` to attach to it, or stop it there first to resume here.\x00", 2),
		strings.Repeat("). Use `claude agents` to find and attach to it, or add --fork-session to branch off a copy.\x00", 2),
		`function ZWd(e){let t=Math.max(0,e)/1000,r=1-Math.exp(-t/90);return Math.min(95,Math.round(r*100))}`,
		`function w0(){if(Mwo())return!0;if(dPt())return!1;return!B6()&&VRt()}`,
		`async function Zks(){if(Mwo())return!0;if(dPt())return!1;return uPt()&&!B6()&&ccr()&&await Oj("tengu_ccr_bridge")}`,
		`async function Nwo(){` + strings.Repeat(" ", 3000) + `function Gq_(){}`,
	}
	data := []byte(strings.Join(parts, "\x00"))
	for _, replacement := range claude212UIBrandingReplacements {
		remaining := replacement.expectedCount - bytes.Count(data, []byte(replacement.old))
		if remaining < 0 {
			t.Fatalf("functional fixture overproduced %q by %d occurrences", replacement.old, -remaining)
		}
		data = append(data, []byte(strings.Repeat(replacement.old+"\x00", remaining))...)
	}
	if !validateClaude209UIBrandingReplacements(data, claude212UIBrandingReplacements) {
		t.Fatal("complete Claude 2.1.212 fixture failed branding-count validation")
	}
	return data
}

func TestFitReplacementRejectsTruncation(t *testing.T) {
	if got, ok := fitReplacement([]byte("short"), "too long"); ok || got != nil {
		t.Fatalf("fitReplacement silently truncated replacement: got %q, ok=%v", got, ok)
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

func TestPrepareClaudeExecutableFallsBackWhenUIPatchUnsupported(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '9.9.9'; exit 0; fi\necho fake claude\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	got := prepareClaudeExecutable(context.Background(), dir, claudePath, "test", modelconfig.Default(), &stderr)
	if got != claudePath {
		t.Fatalf("prepareClaudeExecutable() = %q, want original executable %q", got, claudePath)
	}
	for _, want := range []string{
		"warning: Claudodex has no verified UI patch",
		"Claude Code 9.9.9",
		"launching with the unpatched Claude Code UI",
		"https://github.com/bassner/claudodex/issues",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("warning missing %q:\n%s", want, stderr.String())
		}
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
