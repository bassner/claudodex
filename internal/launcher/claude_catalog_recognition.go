package launcher

const catalogModelIDsEnv = "CLAUDODEX_MODEL_IDS"

// These exact registered binaries recognize only their four configured role
// targets. Extend that UI check to the validated catalog without changing the
// historical patches or mapping an explicit GPT ID to a different role. The
// launcher replaces the allowlist environment variable on every launch; request
// validation and all critical routing remain outside this optional UI check.
func patchClaudeCatalogRecognition(data []byte, source claudeSource) bool {
	var function, matches, expectedSHA string
	switch source.Version {
	case "2.1.322":
		function, matches, expectedSHA = "fSt", "Jv", claude322SHA
	case "2.1.319":
		function, matches, expectedSHA = "nyt", "Rv", claude319SHA
	case "2.1.318":
		function, matches, expectedSHA = "Lmt", "bv", claude318SHA
	case "2.1.317":
		function, matches, expectedSHA = "Ect", "Lv", claude317SHA
	case "2.1.314":
		function, matches, expectedSHA = "kat", "uv", claude314SHA
	case "2.1.316":
		function, matches, expectedSHA = "tlt", "wv", claude316SHA
	default:
		return true
	}
	if source.SHA256 != expectedSHA {
		return true // No overlay for any other source; strict selection owns verification.
	}
	original := `function ` + function + `(e){return[i.ANTHROPIC_DEFAULT_FABLE_MODEL,i.ANTHROPIC_DEFAULT_OPUS_MODEL,i.ANTHROPIC_DEFAULT_SONNET_MODEL,i.ANTHROPIC_DEFAULT_HAIKU_MODEL].some((n)=>n!==void 0&&` + matches + `(n,e))}`
	// i is Claude's typed environment view and does not export our own key.
	replacement := `function ` + function + `(e){return["FABLE","OPUS","SONNET","HAIKU"].map(n=>i["ANTHROPIC_DEFAULT_"+n+"_MODEL"]).concat((process.env.` + catalogModelIDsEnv + `??"").split(",")).some(n=>n&&` + matches + `(n,e))}`
	return replaceFirstFixed(data, original, replacement)
}
