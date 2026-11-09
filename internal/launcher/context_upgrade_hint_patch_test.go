package launcher

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestContextUpgradeHintPatch(t *testing.T) {
	for _, test := range []struct {
		version, name string
		patch         func([]byte) bool
	}{
		{"2.1.314", "RCe", patchContextUpgradeHint_2_1_314},
		{"2.1.316", "tTe", patchContextUpgradeHint_2_1_316},
		{"2.1.319", "AOe", patchContextUpgradeHint_2_1_319},
		{"2.1.318", "pPe", patchContextUpgradeHint_2_1_318},
		{"2.1.317", "oAe", patchContextUpgradeHint_2_1_317},
	} {
		t.Run(test.version, func(t *testing.T) {
			// Exact helper body from the SHA-verified native version. The footer
			// consumes warning; the conversation tip consumes tip.
			original := `function ` + test.name + `(e){let t=n();if(!t)return null;switch(e){case"warning":return` + "`/model ${t.alias}`" + `;case"tip":return` + "`Tip: You have access to ${t.name} with ${t.multiplier}x more context`" + `;default:return null}}`
			suffix := "\nexport{" + test.name + "};\n\x00// @bun @bytecode\nfollowing module"
			data := []byte(original + suffix)
			if !test.patch(data) || len(data) != len(original+suffix) {
				t.Fatal("context upgrade patch failed or changed the binary size")
			}
			if !bytes.HasSuffix(data, []byte(suffix)) {
				t.Fatal("patch changed module exports or the following module")
			}
			if test.patch(data) || test.patch([]byte(strings.Replace(original+suffix, "let t=n()", "let t=changed()", 1))) {
				t.Fatal("patch accepted an altered target")
			}
			node, err := exec.LookPath("node")
			if err != nil {
				t.Skip("node unavailable for helper behavior check")
			}
			patched := string(data[:len(original)])
			script := `function n(){return {alias:"opus[1m]",name:"Opus 1M",multiplier:5}};(()=>{` + original + `;if(` + test.name + `("warning")!=="/model opus[1m]"||!` + test.name + `("tip").includes("5x more context"))throw Error("fixture did not reproduce the recommendation")})();(()=>{` + patched + `;for(const mode of ["warning","tip","unknown"]){if(` + test.name + `(mode)!==null)throw Error("recommendation remained: "+mode)}})()`
			if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
				t.Fatalf("context hint behavior: %v: %s", err, output)
			}
		})
	}
}
