package launcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestClaude323PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.323", claude323SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.323 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.323", claude296SHA); got != nil {
		t.Fatalf("Claude 2.1.323 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.296", claude323SHA); got != nil {
		t.Fatalf("Claude 2.1.323 SHA matched wrong version: %#v", got)
	}
}

func TestClaude323WrongSHAFailsStrictSelection(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.323"
	if err := os.WriteFile(claudePath, []byte("not the verified binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	_, err := selectClaudeExecutable(context.Background(), t.TempDir(), claudePath, "test", modelconfig.Default(), &stderr)
	if err == nil || !strings.Contains(err.Error(), "claude install "+claudeUIPatches[0].Version) {
		t.Fatalf("wrong-SHA source did not fail closed with an exact install command: %v", err)
	}
}

func TestClaude323AgentValidatorRejectsAmbiguousTargets(t *testing.T) {
	const target = `model:B(Zua).optional()`
	for _, input := range []string{"missing", target + ";" + target} {
		data := []byte(input)
		if patchAgentModelValidator_2_1_323(data) || string(data) != input {
			t.Fatal("ambiguous agent validator changed")
		}
	}
	data := []byte(target)
	if !patchAgentModelValidator_2_1_323(data) || len(data) != len(target) {
		t.Fatal("unique agent validator failed to fit")
	}
}

func TestClaude323PatchTargetsMaintenanceBinary(t *testing.T) {
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	if version := detectClaudeVersion(context.Background(), path); version != "2.1.323" {
		t.Logf("maintenance Claude version is %s; this test covers only 2.1.323", version)
		return
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(source); got != claude323SHA {
		t.Fatalf("maintenance Claude SHA = %s, want %s", got, claude323SHA)
	}
	for _, transformation := range claude323Transformations("test") {
		t.Run(transformation.name, func(t *testing.T) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Fatalf("%s target does not match Claude 2.1.323", transformation.name)
			}
		})
	}
	if !validateClaude209UIBrandingReplacements(source, claude323UIBrandingReplacements) {
		for _, replacement := range claude323UIBrandingReplacements {
			if got := bytes.Count(source, []byte(replacement.old)); got != replacement.expectedCount {
				t.Errorf("branding count for %q = %d, want %d", replacement.old, got, replacement.expectedCount)
			}
		}
		t.FailNow()
	}
	records, hashes, ok := claude323EmbeddedBunModuleHashes(source)
	if !ok {
		t.Fatal("Claude 2.1.323 Bun module table is unavailable")
	}
	patched := append([]byte(nil), source...)
	if !applyClaudeUIPatches_2_1_323(patched, "test", "2.1.323", modelconfig.Default()) {
		t.Fatal("complete Claude 2.1.323 patch did not apply")
	}
	start := bytes.Index(patched, []byte("function CDX323("))
	if start < 0 {
		t.Fatal("patched picker marker is absent")
	}
	end := bytes.Index(patched[start:], []byte("function rw("))
	if end < 0 || strings.Count(string(patched[start:start+end]), `r("`) != 4 {
		t.Fatal("patched picker does not contain exactly four tiers")
	}
	for _, forbidden := range []string{`r("mythos",`, "Mythos 5"} {
		if bytes.Contains(patched[start:start+end], []byte(forbidden)) {
			t.Fatalf("patched picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
	changedModules, parsedModules := 0, 0
	for index, record := range records {
		current := sha256.Sum256(patched[record.contentOffset : record.contentOffset+record.contentLength])
		if current == hashes[index] {
			continue
		}
		changedModules++
		// Branding also changes embedded Markdown assets. Parse each changed
		// JavaScript module, while retaining bytecode invalidation for all assets.
		content := patched[record.contentOffset : record.contentOffset+record.contentLength]
		if bytes.HasPrefix(content, []byte("// @bun")) {
			parsedModules++
			node, err := exec.LookPath("node")
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(node, "--input-type=module", "--check")
			cmd.Stdin = bytes.NewReader(content)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("changed Bun module %d has invalid syntax: %v: %s", index, err, output)
			}
		}
		if bytecodeLength := binary.LittleEndian.Uint32(patched[record.bytecodeLength : record.bytecodeLength+4]); bytecodeLength != 0 {
			t.Errorf("changed Bun module %d retained %d bytes of stale bytecode", index, bytecodeLength)
		}
	}
	if parsedModules < 2 {
		t.Fatalf("parsed JavaScript module count = %d, want multiple patched modules", parsedModules)
	}
	if changedModules < 2 {
		t.Fatalf("changed Bun module count = %d, want multiple patched modules", changedModules)
	}
	broken := bytes.Replace(append([]byte(nil), source...), claude323RequiredLogoAnchor(), []byte("function MISSING_TARGET(){"), 1)
	if applyClaudeUIPatches_2_1_323(broken, "test", "2.1.323", modelconfig.Default()) {
		t.Fatal("patch succeeded without the required logo transformation")
	}
}

func TestClaude323RemoteControlTargetsMaintenanceBinary(t *testing.T) {
	path := os.Getenv("CLAUDODEX_MAINTENANCE_CLAUDE_REALPATH")
	if path == "" {
		t.Skip("maintenance Claude path is unavailable")
	}
	if version := detectClaudeVersion(context.Background(), path); version != "2.1.323" {
		t.Logf("maintenance Claude version is %s; this test covers only 2.1.323", version)
		return
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, transformation := range claude323RemoteControlTransformations() {
		t.Run(transformation.name, func(t *testing.T) {
			candidate := append([]byte(nil), source...)
			if !transformation.apply(candidate) {
				t.Fatalf("remote-control %s target does not match Claude 2.1.323", transformation.name)
			}
		})
	}
}

func TestClaude323ReleaseNotesPatchPreservesFollowingModules(t *testing.T) {
	const picker = `function J(i){return "release notes picker"}export{J as ReleaseNotesPicker,te as call};`
	const nextModule = "\x00// @bun @bytecode\nfunction J(i){return i}"
	data := []byte(`var te=async(i,t)=>{try{` + strings.Repeat(" ", 400) + `}catch{}};` + picker + nextModule)
	if !patchWhatsNewFeedFunction_2_1_323(data) {
		t.Fatal("release notes patch did not apply")
	}
	if !bytes.HasSuffix(data, []byte(picker+nextModule)) {
		t.Fatal("release notes patch overwrote the picker, exports or the following module")
	}
	if !bytes.Contains(data, []byte("Claudodex Info")) {
		t.Fatal("release notes branding is missing")
	}
}

func TestClaude323ResumeHintsChangeGeneratedCommand(t *testing.T) {
	const command = "function aRa({sessionId:e,worktree:n,columns:r,envAssignment:s,launchFlags:l=\"\",launcherProgram:c}){let g=s===void 0?\"claude\":`${s} claude`;return c}import{sep as Vp}"
	data := []byte(strings.Repeat("Resume this session with:\n", 3) +
		"Previous session saved \\xB7 resume with: claude --resume \n" +
		strings.Repeat("Run claude --continue or claude --resume to resume a conversation\n", 2) +
		strings.Repeat("Run claude --resume to pick a session, or start a new one.\n", 2) +
		"resume it with: claude --resume \n" + command)
	size := len(data)
	if !patchResumeCommandHints_2_1_323(data) {
		t.Fatal("resume hint patch did not apply")
	}
	if len(data) != size {
		t.Fatal("resume hint patch changed binary size")
	}
	if !bytes.Contains(data, []byte(`let g=(s===void 0?"":s+" ")+"claudodex"`)) {
		t.Fatal("generated resume command still uses Claude")
	}
	if bytes.Contains(data, []byte("claude --resume")) {
		t.Fatal("a resume hint still launches Claude")
	}
}

func TestClaude323ConfiguredModelRecognitionPreservesUnknownModels(t *testing.T) {
	const original = `function tIe(e,n=at(e)){return G1(n)||pz(e,n)}function pz(e,n){return!1}`
	data := []byte(original)
	if !patchConfiguredModelRecognition_2_1_323(data) || len(data) != len(original) {
		t.Fatal("configured model recognizer did not apply without changing binary size")
	}
	if patchConfiguredModelRecognition_2_1_323(data) {
		t.Fatal("recognition patch accepted an already changed target")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable for the JavaScript behavior check")
	}
	// Compose recognition with the exact catalog overlay used at launch.
	catalog := []byte(`function pvt(e){return[i.ANTHROPIC_DEFAULT_FABLE_MODEL,i.ANTHROPIC_DEFAULT_OPUS_MODEL,i.ANTHROPIC_DEFAULT_SONNET_MODEL,i.ANTHROPIC_DEFAULT_HAIKU_MODEL].some((n)=>n!==void 0&&BT(n,e))}`)
	if !patchClaudeCatalogRecognition(catalog, claudeSource{Version: "2.1.323", SHA256: claude323SHA}) {
		t.Fatal("catalog recognition overlay failed")
	}
	script := `const configured=["gpt-6.1-sol","gpt-6.1-terra","gpt-6-luna","gpt-6-astra"],i={};["OPUS","SONNET","HAIKU","FABLE"].forEach((role,index)=>i["ANTHROPIC_DEFAULT_"+role+"_MODEL"]=configured[index]);process.env.CLAUDODEX_MODEL_IDS="gpt-8.2-sol";function BT(a,b){return a===b}function at(e){return e}function G1(e){return e==="claude-opus-native"};` + string(catalog) + string(data) + `;for(const model of [...configured,"gpt-8.2-sol","claude-opus-native"]){if(!tIe(model))throw Error("configured or catalog model rejected: "+model)}for(const model of ["gpt-unconfigured","gpt-6-astra-extra","","other"]){if(tIe(model))throw Error("unknown model accepted: "+model)}`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("configured recognition behavior failed: %v: %s", err, output)
	}
}

func TestClaude323DefaultModelCursorMatchesResolvedModel(t *testing.T) {
	picker := []byte(`function cVe(e=!1,n=null){` + strings.Repeat(" ", 5000) + `}function rw(e,n){}function ona(e,n){` + strings.Repeat(" ", 500) + `}function xp(){}`)
	if !patchModelPickerOptions_2_1_323(picker, modelconfig.Default()) || !patchModelPickerSelectionValue_2_1_323(picker) {
		t.Fatal("picker fixture did not patch")
	}
	focus := []byte(`function tL(h,P){return P===null?LT:ona(h,P)??P}`)
	if !patchModelPickerDefaultFocus_2_1_323(focus) {
		t.Fatal("default cursor target did not patch")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable for picker behavior check")
	}
	script := `const i={};` + string(picker) + string(focus) + `;let LT="__NO_PREFERENCE__",effective="gpt-6.1-terra";function kd(){return effective}function focused(model){return tL(cVe(),model)}if(focused(null)!=="sonnet")throw Error("default Sonnet cursor starts elsewhere");if(focused("gpt-6.1-terra[1m]")!=="sonnet")throw Error("explicit Sonnet mapping changed");if(focused("fable")!=="fable")throw Error("explicit Fable mapping changed");effective="gpt-6-astra";if(focused(null)!=="fable")throw Error("default Fable cursor starts elsewhere");if(focused("unknown")!=="unknown")throw Error("unknown fallback changed");`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("default cursor behavior: %v: %s", err, output)
	}
}

func TestClaude323PickerLabelsAndOverrides(t *testing.T) {
	cfg := modelconfig.Config{Opus: "custom-sol", Sonnet: "custom-terra", Haiku: "custom-luna", Fable: "custom-astra"}
	data := []byte(`function cVe(e=!1,n=null){` + strings.Repeat(" ", 5000) + `}function rw(e,n){}`)
	size := len(data)
	if !patchModelPickerOptions_2_1_323(data, cfg) || len(data) != size {
		t.Fatal("picker override changed binary size or failed to fit")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	script := `const i={ANTHROPIC_DEFAULT_OPUS_MODEL_NAME:"live-sol",ANTHROPIC_DEFAULT_SONNET_MODEL_NAME:"live-terra",ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME:"live-luna",ANTHROPIC_DEFAULT_FABLE_MODEL_NAME:"live-astra"};` + string(data) + `;const rows=cVe();if(JSON.stringify(rows.map(x=>[x.value,x.label,x.description]))!==JSON.stringify([["opus","Opus","live-sol"],["sonnet","Sonnet","live-terra"],["haiku","Haiku","live-luna"],["fable","Fable","live-astra"]]))throw Error("four live role labels changed");for(const [model,role] of [["custom-sol","opus"],["custom-terra","sonnet"],["custom-luna","haiku"],["custom-astra","fable"]])if(CDX323(model)!==role)throw Error("override lost");for(const key of Object.keys(i))delete i[key];if(JSON.stringify(cVe().map(x=>x.description))!==JSON.stringify(["custom-sol","custom-terra","custom-luna","custom-astra"]))throw Error("configured fallback labels lost");`
	if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("picker behavior: %v: %s", err, output)
	}
	original := []byte(`function cVe(e=!1,n=null){return []}function rw(e,n){}`)
	overflow := append([]byte(nil), original...)
	cfg.Opus = strings.Repeat("x", 10000)
	if patchModelPickerOptions_2_1_323(overflow, cfg) || !bytes.Equal(overflow, original) {
		t.Fatal("overflowing picker replacement did not fail without modifying bytes")
	}
}

func TestClaude323UsagePatchPreservesModuleSyntax(t *testing.T) {
	const anchor = `function Ae(e){switch(e.read){case void 0:case"plain":return"/api/oauth/usage";case"cedar_ember":return"/api/oauth/usage?cedar_ember=1&skip_spend=1";case"at_wall":return"/api/oauth/usage?skip_spend=1"}}`
	const following = `function oe(e){return e}export{Re,oe};`
	data := []byte(anchor + `async function Re(e,n,r){` + strings.Repeat(" ", 2000) + `return null}` + following)
	if !patchUsageFetchFunction_2_1_323(data) || !bytes.HasSuffix(data, []byte(following)) {
		t.Fatal("usage replacement failed or damaged following module")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--input-type=module", "--check")
	cmd.Stdin = bytes.NewReader(data)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("usage module syntax: %v: %s", err, output)
	}
}

func TestClaude323RemoteControlPreservesExportedAuthHelper(t *testing.T) {
	const helper = `function AFa(){return}async function Q(){return(await gan())?.reason??null}export{gan,AFa};`
	data := []byte(`async function gan(){` + strings.Repeat(" ", 1000) + `return null}` + helper)
	transformation := claude323RemoteControlTransformations()[2]
	if !transformation.apply(data) || !bytes.HasSuffix(data, []byte(helper)) {
		t.Fatal("Remote Control replacement removed its exported auth helper")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--input-type=module", "--check")
	cmd.Stdin = bytes.NewReader(data)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Remote Control module syntax: %v: %s", err, output)
	}
}
