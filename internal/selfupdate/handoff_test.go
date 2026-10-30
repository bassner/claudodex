package selfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandoffRejectsTamperingAndExpires(t *testing.T) {
	for _, kind := range []string{"hash", "version", "expired", "permissions"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(t.TempDir(), "claudodex")
			if err := os.WriteFile(path, []byte("trusted binary"), 0700); err != nil {
				t.Fatal(err)
			}
			token, cleanup, err := createHandoff(home, path, "1.1.0")
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			dir, _ := handoffDir(home)
			record := filepath.Join(dir, token+".json")
			switch kind {
			case "hash":
				os.WriteFile(path, []byte("changed binary"), 0700)
			case "expired":
				data, _ := os.ReadFile(record)
				var h handoff
				json.Unmarshal(data, &h)
				h.Expires = time.Now().Add(-time.Minute)
				data, _ = json.Marshal(h)
				os.WriteFile(record, data, 0600)
			case "permissions":
				os.Chmod(record, 0644)
			}
			version := "1.1.0"
			if kind == "version" {
				version = "1.2.0"
			}
			t.Setenv(handoffEnv, token)
			if consumeHandoff(home, path, version) || os.Getenv(handoffEnv) != "" {
				t.Fatal("unsafe handoff accepted or marker leaked")
			}
		})
	}
}

func TestBuildInfoReadsRecordedLinkerValuesWithoutExecution(t *testing.T) {
	path := buildHandoffFixture(t)
	version, method, err := installedBuild(path)
	if err != nil || version != "1.1.0" || method != "homebrew" {
		t.Fatalf("version=%s method=%s err=%v", version, method, err)
	}
}

func buildHandoffFixture(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(filepath.Join(root, "cmd"), ".update-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	source := `package main
import("encoding/json";"os";"io";"github.com/bassner/claudodex/internal/selfupdate")
var version="dev"
var installMethod="unknown"
func main(){
 selfupdate.New(selfupdate.Config{Version:version,InstallMethod:installMethod,Home:os.Getenv("CLX_HANDOFF_TEST_HOME")})
 input,_:=io.ReadAll(os.Stdin);cwd,_:=os.Getwd()
 json.NewEncoder(os.Stdout).Encode(map[string]any{"args":os.Args[1:],"cwd":cwd,"input":string(input),"marker":os.Getenv("CLAUDODEX_UPDATE_HANDOFF"),"custom":os.Getenv("CLX_HANDOFF_CUSTOM"),"pid":os.Getpid()})
 os.Exit(23)
}`
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bin", "claudodex")
	os.MkdirAll(filepath.Dir(path), 0700)
	cmd := exec.Command("go", "build", "-ldflags", "-X main.version=1.1.0 -X main.installMethod=homebrew", "-o", path, file)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, out)
	}
	path, _ = filepath.EvalSymlinks(path)
	return path
}

func TestSameInvocationExecPreservesProcessAndIO(t *testing.T) {
	if target := os.Getenv("CLX_HANDOFF_TEST_TARGET"); target != "" {
		m, _, _ := managerFixture(t)
		m.cfg.Home = os.Getenv("CLX_HANDOFF_TEST_HOME")
		m.buildInfo = installedBuild
		original := m.command
		m.command = func(ctx context.Context, path string, args, env []string, live bool) ([]byte, error) {
			if args[0] == "--prefix" {
				return []byte(filepath.Dir(filepath.Dir(target))), nil
			}
			return original(ctx, path, args, env, live)
		}
		m.exec = replaceProcess
		if _, err := m.BeforeLaunch(context.Background(), Request{Args: managerArgs()}); err != nil {
			t.Fatal(err)
		}
		t.Fatal("successful exec returned")
	}
	target := buildHandoffFixture(t)
	home := t.TempDir()
	cwd := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSameInvocationExecPreservesProcessAndIO$")
	cmd.Env = append(os.Environ(), "CLX_HANDOFF_TEST_TARGET="+target, "CLX_HANDOFF_TEST_HOME="+home, "CLX_HANDOFF_CUSTOM=preserved")
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader("piped prompt survives")
	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	err := cmd.Wait()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 23 {
		t.Fatalf("exit=%v out=%s errout=%s", err, &output, &stderr)
	}
	var result struct {
		Args                  []string
		Cwd                   string
		Input, Marker, Custom string
		PID                   int
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v: %s", err, &output)
	}
	actualCwd, _ := filepath.EvalSymlinks(cwd)
	if strings.Join(result.Args, "|") != strings.Join(managerArgs(), "|") || result.Cwd != actualCwd || result.Input != "piped prompt survives" || result.Marker != "" || result.Custom != "preserved" || result.PID != pid {
		t.Fatalf("handoff lost state: %+v originalpid=%d", result, pid)
	}
	dir, _ := handoffDir(home)
	files, _ := os.ReadDir(dir)
	if len(files) > 0 {
		t.Fatal("handoff record was not consumed")
	}
}
