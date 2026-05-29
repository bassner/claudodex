package launcher

import "github.com/bassner/claudodex/internal/modelconfig"

var claudeUIPatch_2_1_156 = claudeUIPatchSpec{
	Version: "2.1.156",
	GOOS:    "darwin",
	GOARCH:  "arm64",
	SHA256:  "9c1e8601031f5cbb3101e49dda22bf8ba31183692c705e267a6923585fa2ba09",
	Apply:   applyClaudeUIPatches_2_1_156,
}

func applyClaudeUIPatches_2_1_156(data []byte, claudodexVersion, claudeVersion string, modelCfg modelconfig.Config) bool {
	basePatched := applyClaudeUIPatches_2_1_154(data, claudodexVersion, claudeVersion, modelCfg)
	compactProgressPatched := patchCompactProgressCurve_2_1_156(data)
	return basePatched && compactProgressPatched
}

func patchCompactProgressCurve_2_1_156(data []byte) bool {
	const old = `function Io7(H){let _=Math.max(0,H)/1000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`
	const replacement = `function Io7(H){let _=Math.max(0,H)/2000,q=1-Math.exp(-_/90);return Math.min(95,Math.round(q*100))}`
	return replaceFirstFixed(data, old, replacement)
}
