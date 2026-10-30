package selfupdate

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func managerFixture(t *testing.T) (*Manager, *bytes.Buffer, *[]string) {
	t.Helper()
	t.Setenv("CLAUDODEX_CHILD", "")
	t.Setenv("CLAUDODEX_NO_AUTO_UPDATE", "")
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	manifest := Manifest{Schema: 1, Repository: Repository, Channel: Channel, KeyID: "test", Version: "1.1.0", ReleaseNotes: "A tested update."}
	data, _ := json.Marshal(manifest)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(private, data))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sig") {
			io.WriteString(w, sig)
		} else {
			w.Write(data)
		}
	}))
	t.Cleanup(server.Close)
	var out bytes.Buffer
	m := New(Config{Version: "1.0.0", Home: t.TempDir(), Stderr: &out})
	m.feed = feed{client: server.Client(), base: server.URL, keys: map[string]ed25519.PublicKey{"test": public}}
	m.path = filepath.Join(t.TempDir(), "current")
	target := filepath.Join(t.TempDir(), "claudodex")
	if err := os.WriteFile(target, []byte("verified test executable"), 0700); err != nil {
		t.Fatal(err)
	}
	m.interactive = func([]string, *os.File, io.Writer) bool { return true }
	m.isHomebrew = func(string) bool { return true }
	m.lookPath = func(string) (string, error) { return "/test/brew", nil }
	m.pinHelper = func(string, string) (string, error) { return "/test/pinned-helper", nil }
	m.buildInfo = func(string) (string, string, error) { return "1.1.0", "homebrew", nil }
	m.prompt = func(context.Context, io.Writer) (bool, error) { return true, nil }
	var calls []string
	m.command = func(_ context.Context, path string, args, env []string, live bool) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if path != "/test/brew" {
			t.Fatalf("unexpected executable %s", path)
		}
		if !strings.Contains(strings.Join(env, "\n"), "HOMEBREW_NO_AUTO_UPDATE=1") {
			t.Fatal("missing scoped brew environment")
		}
		switch args[0] {
		case "info":
			return []byte(`{"formulae":[{"full_name":"bassner/tap/claudodex","versions":{"stable":"1.1.0"}}]}`), nil
		case "--prefix":
			return []byte(filepath.Dir(filepath.Dir(target))), nil
		}
		return nil, nil
	}
	// The fake prefix must still have the standard bin/claudodex shape.
	target = filepath.Join(filepath.Dir(target), "bin", "claudodex")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("verified test executable"), 0700); err != nil {
		t.Fatal(err)
	}
	target, _ = filepath.EvalSymlinks(target)
	m.exec = func(path string, argv, env []string) error {
		calls = append(calls, "exec")
		if path != target {
			t.Fatalf("path=%s want %s", path, target)
		}
		if !reflect.DeepEqual(argv, []string{target, "--claudodex-opus-model", "custom-model", "--model", "opus"}) {
			t.Fatalf("lost original argv: %q", argv)
		}
		var token string
		for _, entry := range env {
			if strings.HasPrefix(entry, handoffEnv+"=") {
				token = strings.TrimPrefix(entry, handoffEnv+"=")
			}
		}
		if token == "" {
			t.Fatal("missing one-use handoff")
		}
		t.Setenv(handoffEnv, token)
		if !consumeHandoff(m.cfg.Home, path, "1.1.0") {
			t.Fatal("handoff did not validate")
		}
		if os.Getenv(handoffEnv) != "" || consumeHandoff(m.cfg.Home, path, "1.1.0") {
			t.Fatal("handoff was reusable or leaked")
		}
		return nil
	}
	return m, &out, &calls
}

func managerArgs() []string {
	return []string{"--claudodex-opus-model", "custom-model", "--model", "opus"}
}

func TestManagerAcceptedUpdateReexecutesOriginalInvocation(t *testing.T) {
	m, out, calls := managerFixture(t)
	result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
	if err != nil || !result.Reexecuted || result.StatusLineExecutable != "/test/pinned-helper" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	want := []string{"update", "info --json=v2 --formula " + Formula, "upgrade --formula --yes " + Formula, "--prefix " + Formula, "exec"}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("calls=%q", *calls)
	}
	if !strings.Contains(out.String(), "A tested update.") || !strings.Contains(out.String(), "[More") && !strings.Contains(out.String(), ReleaseURL("1.1.0")) {
		t.Fatalf("notes/link not shown: %s", out)
	}
}

func TestManagerNoConsentHasNoMutations(t *testing.T) {
	for _, answer := range []string{"no", "eof", "cancel"} {
		t.Run(answer, func(t *testing.T) {
			m, _, calls := managerFixture(t)
			m.prompt = func(context.Context, io.Writer) (bool, error) {
				switch answer {
				case "eof":
					return false, io.EOF
				case "cancel":
					return false, context.Canceled
				}
				return false, nil
			}
			m.pinHelper = func(string, string) (string, error) { t.Fatal("pin before consent"); return "", nil }
			result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
			if err != nil || result.Reexecuted || len(*calls) != 0 {
				t.Fatalf("result=%+v calls=%q err=%v", result, *calls, err)
			}
		})
	}
}

func TestManagerPipedInputAndNestedAreReadOnly(t *testing.T) {
	for _, kind := range []string{"pipe", "nested", "nested-manual-yes"} {
		t.Run(kind, func(t *testing.T) {
			m, _, calls := managerFixture(t)
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			io.WriteString(w, "the Claude prompt\n")
			w.Close()
			m.cfg.Stdin = r
			m.interactive = func([]string, *os.File, io.Writer) bool { return kind != "pipe" }
			if strings.HasPrefix(kind, "nested") {
				t.Setenv("CLAUDODEX_CHILD", "1")
			}
			m.prompt = func(context.Context, io.Writer) (bool, error) { t.Fatal("unexpected prompt"); return false, nil }
			result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs(), Manual: kind == "nested-manual-yes", Yes: kind == "nested-manual-yes"})
			if kind == "nested-manual-yes" && err == nil {
				t.Fatal("manual nested consent should fail")
			}
			if result.Reexecuted || len(*calls) != 0 {
				t.Fatalf("mutated: %+v %q", result, *calls)
			}
			remaining, _ := io.ReadAll(r)
			if string(remaining) != "the Claude prompt\n" {
				t.Fatalf("input consumed: %q", remaining)
			}
		})
	}
}

func TestManagerSourceInstallOnlyNotifies(t *testing.T) {
	m, out, calls := managerFixture(t)
	m.isHomebrew = func(string) bool { return false }
	m.prompt = func(context.Context, io.Writer) (bool, error) { t.Fatal("prompt on source install"); return false, nil }
	result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
	if err != nil || result.Reexecuted || len(*calls) > 0 || !strings.Contains(out.String(), "updates manually") {
		t.Fatalf("result=%+v err=%v output=%s", result, err, out)
	}
}

func TestManagerUpgradeFailureRetainsHelperAndCurrentProcess(t *testing.T) {
	m, _, _ := managerFixture(t)
	m.command = func(context.Context, string, []string, []string, bool) ([]byte, error) {
		return nil, errors.New("fixture failure")
	}
	m.exec = func(string, []string, []string) error { t.Fatal("exec after failure"); return nil }
	result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
	if err == nil || result.Reexecuted || result.StatusLineExecutable == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestManagerRejectsUnexpectedInstalledVersion(t *testing.T) {
	m, _, calls := managerFixture(t)
	m.buildInfo = func(string) (string, string, error) { return "1.2.0", "homebrew", nil }
	result, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()})
	if err == nil || result.Reexecuted || (*calls)[len(*calls)-1] == "exec" {
		t.Fatalf("result=%+v calls=%v err=%v", result, *calls, err)
	}
}

func TestChangedFormulaNeedsFreshConsentBeforeUpgrade(t *testing.T) {
	for _, scriptYes := range []bool{false, true} {
		t.Run(map[bool]string{false: "terminal", true: "script"}[scriptYes], func(t *testing.T) {
			m, out, _ := managerFixture(t)
			pub, key, _ := ed25519.GenerateKey(rand.Reader)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				v := "1.1.0"
				if strings.Contains(r.URL.Path, "/v1.2.0/") {
					v = "1.2.0"
				}
				manifest := Manifest{Schema: 1, Repository: Repository, Channel: Channel, KeyID: "test", Version: v, ReleaseNotes: "Notes for " + v}
				data, _ := json.Marshal(manifest)
				if strings.HasSuffix(r.URL.Path, ".sig") {
					io.WriteString(w, base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)))
				} else {
					w.Write(data)
				}
			}))
			defer server.Close()
			m.feed = feed{base: server.URL, client: server.Client(), keys: map[string]ed25519.PublicKey{"test": pub}}
			upgraded := false
			m.command = func(_ context.Context, _ string, args, _ []string, _ bool) ([]byte, error) {
				if args[0] == "info" {
					return []byte(`{"formulae":[{"full_name":"bassner/tap/claudodex","versions":{"stable":"1.2.0"}}]}`), nil
				}
				if args[0] == "upgrade" {
					upgraded = true
				}
				return nil, nil
			}
			prompts := 0
			m.prompt = func(context.Context, io.Writer) (bool, error) { prompts++; return prompts == 1, nil }
			m.exec = func(string, []string, []string) error { t.Fatal("executed without fresh consent"); return nil }
			_, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs(), Manual: scriptYes, Yes: scriptYes})
			if err == nil || upgraded || (!scriptYes && prompts != 2) || (scriptYes && prompts != 0) || !strings.Contains(out.String(), "Notes for 1.2.0") {
				t.Fatalf("err=%v upgraded=%v prompts=%d output=%s", err, upgraded, prompts, out)
			}
		})
	}
}

func TestBrewEnvironmentDoesNotPassModelCredentials(t *testing.T) {
	env := brewEnvironment([]string{"PATH=/bin", "OPENAI_API_KEY=secret", "ANTHROPIC_API_KEY=secret", "CLAUDE_CODE_OAUTH_TOKEN=secret", handoffEnv + "=secret", "HOMEBREW_NO_AUTO_UPDATE=0"})
	if strings.Contains(strings.Join(env, "\n"), "secret") || !strings.Contains(strings.Join(env, "\n"), "PATH=/bin") {
		t.Fatalf("bad environment %q", env)
	}
}

func TestPlainNotesCannotControlTerminal(t *testing.T) {
	notes := PlainNotes("Hello\x1b[31m\a\r\u009b\u202e\u2066world\n" + strings.Repeat("x", 10000))
	for _, r := range notes {
		if r == 27 || r == 7 || r == 13 || r == 0x9b || r == 0x202e || r == 0x2066 {
			t.Fatalf("control %U survived", r)
		}
	}
	if len(notes) > 4200 || !strings.Contains(notes, "More in the release link") {
		t.Fatalf("notes not bounded: %d", len(notes))
	}
}

func TestNewConsumesInvalidMarkerBeforeChildren(t *testing.T) {
	t.Setenv(handoffEnv, "invalid")
	m := New(Config{Version: "1.0.0", Home: t.TempDir()})
	if m.handedOff || os.Getenv(handoffEnv) != "" {
		t.Fatal("invalid marker leaked")
	}
}

func TestNestedInvocationDoesNotWaitForNetwork(t *testing.T) {
	m, _, _ := managerFixture(t)
	t.Setenv("CLAUDODEX_CHILD", "1")
	m.feed.base = "http://127.0.0.1:1"
	if _, err := m.BeforeLaunch(context.Background(), Request{}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.BeforeLaunch(context.Background(), Request{Manual: true, Yes: true}); err == nil {
		t.Fatal("nested manual update allowed")
	}
}

func TestNewSchemaAutomaticNoticeIsActionable(t *testing.T) {
	m, out, _ := managerFixture(t)
	pub, key := newFixtureKey(t)
	manifest := fixtureManifest()
	manifest.Schema = 2
	data, sig := signFixture(t, manifest, key)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sig") {
			w.Write(sig)
		} else {
			w.Write(data)
		}
	}))
	defer server.Close()
	m.feed = feed{base: server.URL, client: server.Client(), keys: map[string]ed25519.PublicKey{fixtureKeyID: pub}}
	if _, err := m.BeforeLaunch(context.Background(), Request{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "needs a newer client; run brew upgrade") {
		t.Fatalf("missing recovery: %s", out)
	}
}

func TestManualHandoffReportsInstalledVersion(t *testing.T) {
	m, out, _ := managerFixture(t)
	m.handedOff = true
	if _, err := m.BeforeLaunch(context.Background(), Request{Manual: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Updated Claudodex to 1.0.0.") {
		t.Fatalf("silent manual handoff: %s", out)
	}
}

func TestPackageManagerCannotConsumePipedPrompt(t *testing.T) {
	m, _, _ := managerFixture(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	io.WriteString(w, "preserve this prompt")
	w.Close()
	m.cfg.Stdin = r
	cat, err := exec.LookPath("cat")
	if err != nil {
		t.Fatal(err)
	}
	output, err := m.runCommand(context.Background(), cat, nil, os.Environ(), false)
	if err != nil || len(output) > 0 {
		t.Fatalf("package operation read stdin: %q %v", output, err)
	}
	remaining, _ := io.ReadAll(r)
	if string(remaining) != "preserve this prompt" {
		t.Fatalf("stdin consumed: %q", remaining)
	}
}
