package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func publishScript(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "packaging", "homebrew", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPublicationAssetsResumeWithoutReplacement(t *testing.T) {
	script := publishScript(t, "publish-asset.sh")
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote")
	if err := os.Mkdir(remote, 0755); err != nil {
		t.Fatal(err)
	}
	gh := `#!/usr/bin/env bash
set -euo pipefail
echo "$*" >> "$TEST_RELEASE_ROOT/calls"
case "$2" in
  view)
    python3 - "$TEST_RELEASE_ROOT/remote" <<'PY'
import json, pathlib, sys
print(json.dumps({"assets": [{"name": p.name} for p in pathlib.Path(sys.argv[1]).iterdir()]}))
PY
    ;;
  upload)
    cp "$4" "$TEST_RELEASE_ROOT/remote/$(basename "$4")"
    ;;
  download)
    while [[ $# -gt 0 ]]; do
      case "$1" in --pattern) name=$2; shift 2;; --dir) dest=$2; shift 2;; *) shift;; esac
    done
    cp "$TEST_RELEASE_ROOT/remote/$name" "$dest/$name"
    ;;
  *) exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(gh), 0755); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(dir, "source.tar.gz")
	if err := os.WriteFile(asset, []byte("reviewed bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	run := func() ([]byte, error) {
		cmd := exec.Command("bash", script, "v0.3.40", asset)
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "TEST_RELEASE_ROOT="+dir)
		return cmd.CombinedOutput()
	}
	for i := 0; i < 2; i++ {
		if output, err := run(); err != nil {
			t.Fatalf("resume publication: %v %s", err, output)
		}
	}
	if err := os.WriteFile(asset, []byte("different bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(); err == nil {
		t.Fatal("replaced a published asset")
	}
	calls, err := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(calls), "release upload") != 1 || strings.Contains(string(calls), "--clobber") {
		t.Fatalf("unexpected upload commands: %s", calls)
	}
	unchanged, err := os.ReadFile(filepath.Join(remote, "source.tar.gz"))
	if err != nil || string(unchanged) != "reviewed bytes" {
		t.Fatal("published bytes changed")
	}
}

func TestOldTagCannotBecomeLatest(t *testing.T) {
	script := publishScript(t, "check-latest.sh")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$TEST_LATEST_TAG\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		offered, latest string
		pass            bool
	}{
		{"0.3.40", "v0.3.39", true}, {"0.3.40", "v0.3.40", true},
		{"0.3.40", "v0.3.41", false}, {"0.3.40", "v0.3.4-malformed", false},
	} {
		cmd := exec.Command("bash", script, tc.offered)
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "TEST_LATEST_TAG="+tc.latest)
		output, err := cmd.CombinedOutput()
		if (err == nil) != tc.pass {
			t.Fatalf("offered=%s latest=%s: %v %s", tc.offered, tc.latest, err, output)
		}
	}
}

func TestTapPublicationPreservesConcurrentCommitsAndRejectsDowngrade(t *testing.T) {
	script := publishScript(t, "publish-tap.sh")
	dir := t.TempDir()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	remote, other := filepath.Join(dir, "remote.git"), filepath.Join(dir, "other")
	gitEnv := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(realGit, args...)
		cmd.Env = gitEnv
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, output)
		}
		return string(output)
	}
	git("init", "--bare", "--initial-branch=main", remote)
	git("clone", remote, other)
	git("-C", other, "config", "user.name", "Publication test")
	git("-C", other, "config", "user.email", "test@example.invalid")
	if err := os.Mkdir(filepath.Join(other, "Formula"), 0755); err != nil {
		t.Fatal(err)
	}
	initial := "class Claudodex < Formula\n  version \"0.3.39\"\nend\n"
	if err := os.WriteFile(filepath.Join(other, "Formula", "claudodex.rb"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	git("-C", other, "add", "--", "Formula/claudodex.rb")
	git("-C", other, "commit", "-m", "Initial formula")
	git("-C", other, "push", "origin", "main")
	// Rewrite only the fixed tap URL to the private test remote. The first push
	// races an unrelated commit, forcing the script's normal fetch/rebase retry.
	wrapper := `#!/usr/bin/env bash
set -euo pipefail
args=("$@")
for i in "${!args[@]}"; do
  if [[ "${args[$i]}" == https://github.com/bassner/homebrew-tap.git ]]; then args[$i]=$TEST_TAP_REMOTE; fi
done
if [[ "$*" == *'push origin HEAD'* && ! -e "$TEST_RACE_MARKER" ]]; then
  touch "$TEST_RACE_MARKER"
  printf 'concurrent\n' > "$TEST_OTHER_CHECKOUT/concurrent.txt"
  "$TEST_REAL_GIT" -C "$TEST_OTHER_CHECKOUT" add -- concurrent.txt
  "$TEST_REAL_GIT" -C "$TEST_OTHER_CHECKOUT" commit -qm 'Unrelated concurrent change'
  "$TEST_REAL_GIT" -C "$TEST_OTHER_CHECKOUT" push --quiet origin main
fi
exec "$TEST_REAL_GIT" "${args[@]}"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(wrapper), 0755); err != nil {
		t.Fatal(err)
	}
	formula := filepath.Join(dir, "claudodex.rb")
	wanted := strings.Replace(initial, "0.3.39", "0.3.40", 1)
	if err := os.WriteFile(formula, []byte(wanted), 0644); err != nil {
		t.Fatal(err)
	}
	run := func(version string) ([]byte, error) {
		cmd := exec.Command("bash", script, formula, version)
		cmd.Env = append(gitEnv, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "TEST_REAL_GIT="+realGit, "TEST_TAP_REMOTE="+remote, "TEST_OTHER_CHECKOUT="+other, "TEST_RACE_MARKER="+filepath.Join(dir, "race"))
		return cmd.CombinedOutput()
	}
	if output, err := run("0.3.40"); err != nil {
		t.Fatalf("publish after concurrent commit: %v %s", err, output)
	}
	if got := git("--git-dir", remote, "show", "main:Formula/claudodex.rb"); got != wanted {
		t.Fatalf("published formula = %q", got)
	}
	if got := git("--git-dir", remote, "show", "main:concurrent.txt"); got != "concurrent\n" {
		t.Fatal("concurrent tap commit was lost")
	}
	if output, err := run("0.3.39"); err == nil {
		t.Fatalf("tap formula downgrade succeeded: %s", output)
	}
}
