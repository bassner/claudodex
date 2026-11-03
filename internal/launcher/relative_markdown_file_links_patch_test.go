package launcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestRelativeMarkdownFileLinksPatch(t *testing.T) {
	for _, test := range []struct {
		version, name, network, normalize, validate, defang string
		control, decode, unescape, unsafe, plugin           string
		patch                                               func([]byte) bool
	}{
		{"2.1.314", "WJn", "Jn", "Bo", "mW", "kZo", "M3", "CZo", "vZo", "v7r", "hp", patchRelativeMarkdownFileLinks_2_1_314},
		{"2.1.316", "yZn", "Qn", "Po", "DW", "Nts", "X3", "Fts", "Lts", "SQr", "Dp", patchRelativeMarkdownFileLinks_2_1_316},
	} {
		t.Run(test.version, func(t *testing.T) {
			// Exact normalizer from each SHA-verified source. Its injected path
			// and safety dependencies below make calls to the native validators
			// observable without requiring an installed Claude binary in CI.
			original := fmt.Sprintf(`function %s(e){let n=/^file:/i.test(e);if(!n){if(!(e.startsWith("/")&&!%s(e)))return e}let r=n?e.slice(5):e;if(r.startsWith("//")){if(r=r.slice(2),r==="localhost")r="/";else if(r.startsWith("localhost/"))r=r.slice(9)}let o=r.search(/[#?]/),s=o===-1?"":r.slice(o),a=o===-1?r:r.slice(0,o);if(a==="")return null;try{a=decodeURIComponent(a)}catch{}a=%s(a);let u=bn(a)?a:kn(pe(),a),h=%s(u);if(h===null)return null;let f=h+s;return %s(f)?null:f}`, test.name, test.network, test.normalize, test.validate, test.defang)
			markdown := fmt.Sprintf(`function Jt(e){let t=%s(e);if(t===null||%s.test(t)||je.test(t)||t!==t.trimEnd())return null;if(ze.test(t))return t;let n=%s(t),r=n.some((o)=>/^file:/i.test(%s(o)));return %s(t,n)||r?null:t}`, test.name, test.control, test.decode, test.unescape, test.unsafe)
			plugin := `function ` + test.plugin + `(e){let r=/^www\./i.test(e)?` + "`http://${e}`" + `:` + test.name + `(e)??e;try{return new URL(r).href}catch{return r}}`
			const prefix, suffix = "untouched prefix;", "function nextModule(){return 1}"
			fixture := prefix + original + plugin + markdown + suffix
			data := []byte(fixture)
			if !test.patch(data) || len(data) != len(fixture) {
				t.Fatal("local-link patch failed or changed the binary size")
			}
			if !bytes.HasPrefix(data, []byte(prefix)) || !bytes.HasSuffix(data, []byte(suffix)) || !bytes.Contains(data, []byte(plugin)) {
				t.Fatal("local-link patch changed surrounding source")
			}
			for _, altered := range [][]byte{data, []byte(fixture + original), []byte(fixture + markdown),
				[]byte(strings.Replace(fixture, "decodeURIComponent(a)", "changed(a)", 1)),
				[]byte(strings.Replace(fixture, test.name+"(e);if(t===null", test.name+"(changed);if(t===null", 1))} {
				before := append([]byte(nil), altered...)
				if test.patch(altered) || !bytes.Equal(altered, before) {
					t.Fatal("patch accepted or mutated an altered/duplicate target")
				}
			}
			node, err := exec.LookPath("node")
			if err != nil {
				t.Skip("node unavailable for local-link behavior check")
			}
			patched := string(data[len(prefix) : len(prefix)+len(original)])
			markdownStart := len(prefix) + len(original) + len(plugin)
			patchedMarkdown := string(data[markdownStart : markdownStart+len(markdown)])
			script := `const assert=require("node:assert/strict"),path=require("node:path"),{pathToFileURL}=require("node:url");
const pe=()=>"/workspace/repo",bn=path.isAbsolute,kn=path.resolve;
let validated=[],defanged=[],rejectURI=false,rejectFinal=false;
const NETWORK=e=>/^[\\/]{2}/.test(e),NORMALIZE=e=>e;
const VALIDATE=e=>{validated.push(e);return rejectURI?null:pathToFileURL(e).href};
const DEFANG=e=>{defanged.push(e);return rejectFinal};
const CONTROL=/[\x00-\x1f\x7f]/,je=/^\s/,ze=/^[a-z][a-z0-9+.-]*:/i;
const DECODE=e=>[e,e],UNESCAPE=e=>e,UNSAFE=e=>false;
const native=(()=>{ORIGINAL;MARKDOWN;PLUGIN;return {normalize:NAME,markdown:Jt,plugin:PLUGINNAME}})();
const patched=(()=>{PATCHED;PATCHEDMARKDOWN;PLUGIN;return {normalize:NAME,markdown:Jt,plugin:PLUGINNAME}})();
const relative=[
 [".local/student-review-2026-11-03/correction-plan-v2.md","/workspace/repo/.local/student-review-2026-11-03/correction-plan-v2.md"],
 [".local/student-review-2026-11-03/plan-consultation-result.md","/workspace/repo/.local/student-review-2026-11-03/plan-consultation-result.md"],
 ["./README.md","/workspace/repo/README.md"],["../outside.md","/workspace/outside.md"],
 ["README.md","/workspace/repo/README.md"],["docs/guide.md","/workspace/repo/docs/guide.md"],
 ["docs/hello world.md","/workspace/repo/docs/hello world.md"],
 ["docs/Grüße 日本語.md","/workspace/repo/docs/Grüße 日本語.md"],
 ["docs/hello%20world.md","/workspace/repo/docs/hello world.md"],
 ["docs/Gr%C3%BC%C3%9Fe%20%E6%97%A5%E6%9C%AC%E8%AA%9E.md","/workspace/repo/docs/Grüße 日本語.md"],
 ["docs/%23hash.md","/workspace/repo/docs/#hash.md"],
 ["docs/100%broken.md","/workspace/repo/docs/100%broken.md"]
];
for(const [input,resolved]of relative){
 assert.equal(native.normalize(input),input,"native fixture must reproduce broken relative destination");
 for(const suffix of ["","?view=1#section","#section"]){
  validated=[];defanged=[];
  const expected=pathToFileURL(resolved).href+suffix;
  assert.equal(patched.normalize(input+suffix,true),expected,input+suffix);
  assert.deepEqual(validated,[resolved],"opt-in URI validation must still run");
  assert.deepEqual(defanged,[expected],"opt-in final defang must still run");
  validated=[];defanged=[];
  assert.equal(native.markdown(input+suffix),input+suffix,"native Markdown must reproduce the failure");
  assert.equal(patched.markdown(input+suffix),expected,"Markdown caller did not opt in");
  assert.deepEqual(validated,[resolved],"native URI validation must still run");
  assert.deepEqual(defanged,[expected],"native final defang must still run");
 }
}
const unchanged=["https://example.org/a?q=1#part","HTTP://example.org","mailto:me@example.org","tel:+123","vscode://file/a","custom+v1.2:target","data:text/plain,hello","javascript:alert(1)","//example.org/a","\\\\server\\share","#section","?view=1","","/absolute/a b.md","file:///absolute/a%20b.md","FILE:/absolute/a.md","file://localhost/a.md","file://remote/path.md","file:","file://localhost","file://localhost/","file://localhost//a","file://localhostx/a","file://LOCALHOST/a","file://localhost#part","file://localhost?view=1","file://localhost%2Fa"];
const nativeFiles=["localhost","localhost/","localhost//","localhostx/","LOCALHOST/","remote/","","localhost\n","localhost\r\n","localhost\u2028","localhost\u2029"].flatMap(host=>["","a","a?view=1#section","%20a","%2Fa","../a"].map(tail=>"file://"+host+tail));
const allInputs=[...unchanged,...nativeFiles,"www.example.org",...relative.flatMap(([input])=>[input,input+"?view=1#section",input+"#section"])];
for(const input of allInputs){
 validated=[];defanged=[];const expected=native.normalize(input),calls=JSON.stringify([validated,defanged]);
 for(const flag of [undefined,false,0]){
  validated=[];defanged=[];assert.equal(patched.normalize(input,flag),expected,"shared helper default changed: "+input);
  assert.equal(JSON.stringify([validated,defanged]),calls,"existing validation behavior changed: "+input);
 }
 assert.equal(patched.plugin(input),native.plugin(input),"hosted plugin URL/allowlist normalization changed: "+input);
}
for(const input of [...unchanged,...nativeFiles]){
 assert.equal(patched.normalize(input,true),native.normalize(input),"opt-in changed a native URL: "+input);
 assert.equal(patched.markdown(input),native.markdown(input),"Markdown URL safety behavior changed: "+input);
}
rejectURI=true;validated=[];defanged=[];
assert.equal(patched.normalize("docs/blocked.md",true),null);assert.equal(validated.length,1);assert.equal(defanged.length,0);
rejectURI=false;rejectFinal=true;
assert.equal(patched.markdown("docs/blocked.md#section"),null);
`
			script = strings.NewReplacer("ORIGINAL", original, "PATCHEDMARKDOWN", patchedMarkdown, "PATCHED", patched, "MARKDOWN", markdown, "PLUGINNAME", test.plugin, "PLUGIN", plugin,
				"NAME", test.name, "NETWORK", test.network, "NORMALIZE", test.normalize, "VALIDATE", test.validate, "DEFANG", test.defang,
				"CONTROL", test.control, "DECODE", test.decode, "UNESCAPE", test.unescape, "UNSAFE", test.unsafe).Replace(script)
			if output, err := exec.Command(node, "-e", script).CombinedOutput(); err != nil {
				t.Fatalf("local-link behavior: %v: %s", err, output)
			}
		})
	}
}
