package launcher

import (
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

func TestClaudodexCommandOnlyFirstArg(t *testing.T) {
	cmd, rest, ok := ClaudodexCommand([]string{"clx:doctor", "--verbose"})
	if !ok || cmd != "clx:doctor" || len(rest) != 1 || rest[0] != "--verbose" {
		t.Fatalf("unexpected command parse: %q %#v %v", cmd, rest, ok)
	}

	if _, _, ok := ClaudodexCommand([]string{"-p", "clx:doctor"}); ok {
		t.Fatal("clx token inside prompt args must pass through to claude")
	}
}

func TestForcePassThrough(t *testing.T) {
	if !ForcePassThrough([]string{"--", "clx:doctor"}) {
		t.Fatal("-- must force claude passthrough")
	}
}

func TestDirectClaudeFastPath(t *testing.T) {
	if !DirectClaudeFastPath([]string{"--help"}) {
		t.Fatal("--help should be direct claude fast path")
	}
	if !DirectClaudeFastPath([]string{"--version"}) {
		t.Fatal("--version should be direct claude fast path")
	}
	if DirectClaudeFastPath([]string{"--help", "extra"}) {
		t.Fatal("multi-arg help must not be fast-pathed")
	}
}

func TestAutomationModeDetectsEqualsOutputFormat(t *testing.T) {
	if !hasAutomationMode([]string{"--output-format=json"}) {
		t.Fatal("--output-format=json should be automation mode")
	}
	if hasAutomationMode([]string{"--output-format=text"}) {
		t.Fatal("--output-format=text should not force automation mode")
	}
}

func TestRewriteClaudeModelArgsUsesLongContextCodexModel(t *testing.T) {
	got := RewriteClaudeModelArgs([]string{
		"--model", "claude-sonnet-4-6",
		"--fallback-model=gpt-5.6-sol[1m]",
		"-p", "leave --model text alone",
	})
	want := []string{
		"--model", "gpt-5.6-terra[1m]",
		"--fallback-model=gpt-5.6-sol[1m]",
		"-p", "leave --model text alone",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q; args=%#v", i, got[i], want[i], got)
		}
	}
}

func TestRewriteClaudeModelArgsUsesConfiguredModelTargets(t *testing.T) {
	got := RewriteClaudeModelArgsWithConfig([]string{"--model", "sonnet"}, modelconfig.Config{Sonnet: "gpt-next-sonnet"})
	want := []string{"--model", "gpt-next-sonnet[1m]"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q; args=%#v", i, got[i], want[i], got)
		}
	}
}

func TestExplicitModelArgFindsOnlyPrimaryModel(t *testing.T) {
	got, ok := explicitModelArg([]string{"--fallback-model=gpt-5.5[1m]", "--model", "gpt-5.4[1m]"})
	if !ok || got != "gpt-5.4[1m]" {
		t.Fatalf("explicitModelArg = %q, %v", got, ok)
	}
}
