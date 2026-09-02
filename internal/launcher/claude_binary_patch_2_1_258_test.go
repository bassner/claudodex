package launcher

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/bassner/claudodex/internal/modelconfig"
)

const claude258SHA = "b63136194160791c27cfa7b0403060d85eb0752991625fde8c09f9acacb17c78"

func TestClaude258PatchRequiresExactVersionOSArchAndSHA(t *testing.T) {
	patch := findClaudeUIPatch("2.1.258", claude258SHA)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if patch == nil {
			t.Fatal("expected verified Claude 2.1.258 darwin/arm64 patch to match")
		}
	} else if patch != nil {
		t.Fatalf("patch matched unsupported runtime %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if got := findClaudeUIPatch("2.1.258", claude252SHA); got != nil {
		t.Fatalf("Claude 2.1.258 patch matched wrong SHA: %#v", got)
	}
	if got := findClaudeUIPatch("2.1.252", claude258SHA); got != nil {
		t.Fatalf("Claude 2.1.258 SHA matched wrong version: %#v", got)
	}
}

func TestClaude258WrongSHAFallsBackToUnpatchedExecutable(t *testing.T) {
	claudePath := t.TempDir() + "/2.1.258"
	if err := os.WriteFile(claudePath, []byte("not the verified binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	got := prepareClaudeExecutable(context.Background(), t.TempDir(), claudePath, "test", modelconfig.Default(), &stderr)
	if got != claudePath {
		t.Fatalf("unsupported executable path = %q, want original %q", got, claudePath)
	}
	if !strings.Contains(stderr.String(), "no verified UI patch") || !strings.Contains(stderr.String(), "sha256:") {
		t.Fatalf("unsupported fallback warning = %q", stderr.String())
	}
}

func TestClaude258ModelPickerContainsExactlyThreeCodexTiers(t *testing.T) {
	data := []byte(`function Xun(e=!1){` + strings.Repeat(" ", 5000) + `function X9r(e,n){}`)
	if !patchModelPickerOptions_2_1_258(data, modelconfig.Default()) {
		t.Fatal("patchModelPickerOptions_2_1_258 reported no changes")
	}
	got := string(data)
	for _, want := range []string{`r("opus",`, `r("sonnet",`, `r("haiku",`, "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	if tiers := strings.Count(got, `r("`); tiers != 3 {
		t.Fatalf("model picker tier count = %d, want 3", tiers)
	}
	if !strings.Contains(got, `.replaceAll("[1m]","")`) || strings.Contains(got, `replace(/(`) {
		t.Fatal("model picker normalizer does not strip Claude's long-context suffix")
	}
	for _, forbidden := range []string{"fable", "Fable", "mythos", "Mythos", "ANTHROPIC_DEFAULT_FABLE_MODEL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained forbidden fourth-tier marker %q", forbidden)
		}
	}
}

func TestClaude258ModelPickerRejectsOverrideAndCustomRows(t *testing.T) {
	data := []byte(claude258ModelOptionsOverrideTarget + claude258ModelSelectionSourceTarget + claude258ModelExtraOptionsTarget + claude258ModelPickerValueTarget)
	if !patchModelPickerExtraOptions_2_1_258(data) {
		t.Fatal("model picker extra-options patch did not apply")
	}
	got := string(data)
	for _, want := range []string{`tt=z(()=>Xun(lt),[lt])`, `Et=Ye??T,ft=CDX258(Et===null?ww:Et),`, `po=tt.slice(0,3)`, `selectedValue:CDX258(ft)`, `options:kn.slice(0,3)`} {
		if !strings.Contains(got, want) {
			t.Fatalf("model picker missing %q", want)
		}
	}
	for _, forbidden := range []string{"Current model", "Base model", "Re??Xun"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("model picker retained custom-row path %q", forbidden)
		}
	}
}

func TestClaude258ModelPickerResolverReturnsOnlyThreeCanonicalRows(t *testing.T) {
	data := []byte(`function eXr(e,n){` + strings.Repeat(" ", 5000) + `function Yun(e){}`)
	if !patchModelPickerResolver_2_1_258(data) {
		t.Fatal("model picker resolver patch did not apply")
	}
	got := string(data)
	if !strings.Contains(got, `function eXr(e,n){return fTe(e).slice(0,3)}`) {
		t.Fatal("model picker resolver retained noncanonical option sources")
	}
	for _, forbidden := range []string{"Custom model", "fable", "mythos"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(forbidden)) {
			t.Fatalf("model picker resolver retained %q", forbidden)
		}
	}
}

func TestClaude258LogoPatchFailsClosedOnOverflow(t *testing.T) {
	data := []byte(`function fHe(){let l=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function DJt(l,b,O){}`)
	original := append([]byte(nil), data...)
	if patchLogoDisplayDataFunction_2_1_258(data, strings.Repeat("x", 4000), "2.1.258") {
		t.Fatal("oversized executable replacement unexpectedly succeeded")
	}
	if !bytes.Equal(data, original) {
		t.Fatal("overflowing executable replacement mutated the input")
	}
}

func TestClaude258LogoPatchEmitsClosedURLRegex(t *testing.T) {
	data := []byte(`function fHe(){let l=a.DEMO_VERSION??` + strings.Repeat(" ", 1000) + `function DJt(l,b,O){}`)
	if !patchLogoDisplayDataFunction_2_1_258(data, "0.3.16", "2.1.258") {
		t.Fatal("logo patch did not apply")
	}
	if !bytes.Contains(data, []byte(`b.replace(/^https?:\/\//,"")`)) {
		t.Fatal("logo patch emitted a URL regular expression without its closing delimiter")
	}
}

func claude258Transformations(version string) []claude258Transformation {
	return []claude258Transformation{
		{"logo", func(data []byte) bool { return patchLogoDisplayDataFunction_2_1_258(data, version, "2.1.258") }},
		{"active-header-brand", patchActiveHeaderBrand_2_1_258},
		{"default-tier-label", patchDefaultTierLabel_2_1_258},
		{"whats-new", patchWhatsNewFeedFunction_2_1_258},
		{"usage", patchUsageFetchFunction_2_1_258},
		{"model-options", func(data []byte) bool { return patchModelPickerOptions_2_1_258(data, modelconfig.Default()) }},
		{"model-resolver", patchModelPickerResolver_2_1_258},
		{"model-extra-options", patchModelPickerExtraOptions_2_1_258},
		{"model-selection", patchModelPickerSelectionValue_2_1_258},
		{"agent-model-validator", patchAgentModelValidator_2_1_258},
		{"fast-mode", patchFastModeRuntimeFunctions_2_1_258},
		{"active-fast-mode-brand", patchActiveFastModeBrand_2_1_258},
		{"fast-mode-pricing", patchFastModePricing_2_1_258},
		{"context-warning", patchContextWarningHint_2_1_258},
		{"resume-hints", patchResumeCommandHints_2_1_258},
		{"compact-progress", patchCompactProgressCurve_2_1_258},
		{"remote-control", patchRemoteControlRuntimeFunctions_2_1_258},
		{"branding", func(data []byte) bool {
			return applyClaude209UIBrandingReplacements(data, claude258UIBrandingReplacements)
		}},
	}
}
