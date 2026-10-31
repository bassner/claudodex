# Contributing To Claudodex

Thanks for helping improve Claudodex. This project has one job: make the real
Claude Code binary work well with OpenAI Codex subscription models. Keep changes
focused on that path.

## Ground Rules

- Do not add broad provider abstractions unless they are directly needed for
  Codex compatibility.
- Do not send Anthropic payment or account headers upstream.
- Keep OpenAI/Codex auth isolated from normal Claude Code auth.
- Preserve normal Claude Code settings behavior wherever possible.
- Prefer live metadata from Codex over hardcoded limits when the data is
  available.
- Treat Claude Code binary patching as versioned compatibility code. Patch only
  verified OS, architecture, version, and binary SHA combinations.

## Development Setup

Use Go 1.25 or newer (matching `go.mod`).

```sh
git clone https://github.com/bassner/claudodex.git
cd claudodex
go build -o ./claudodex ./cmd/claudodex
```

You also need a callable `claude` binary on `PATH` for launcher smoke tests.
Release-tool tests use Git, Python 3, and `jq`.

## Checks

Run these before opening a pull request:

```sh
git ls-files '*.go' | xargs gofmt -w
go test -count=1 ./...
go vet ./...
go build -o ./claudodex ./cmd/claudodex
```

Optional installed-Claude smoke test:

```sh
CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 \
  go test -count=1 ./internal/launcher \
  -run TestInstalledClaudePrintSmokeWithFakeCodexUpstream -v
```

Live tests against Codex require `claudodex clx:auth-login`.

## Releases

Homebrew is the supported distribution channel for macOS arm64. The release
workflow packages clean `git archive` source with dependencies vendored in a
temporary directory. Do not commit `vendor/`, generated formulas, bottles,
binaries, or credentials. See [the packaging guide](packaging/homebrew/README.md)
for local preparation commands.

Before publication, review the complete pending change, run the checks above,
and test the exact registered Claude binary against the local fake backend.
Changes to launcher behavior also need real terminal tests. The release CI verifies its pinned
Claude build against the compiled version/platform/SHA registry before executing it;
its smoke tests do not call a real model.

The tag workflow uses `macos-15` arm64 to build and test the source formula and
bottle. It uploads source, bottles, and signed update metadata to a public
**non-latest** release before updating only `Formula/claudodex.rb` in
`bassner/homebrew-tap`. Publishing the signed metadata first keeps pinned
release notes available when Homebrew metadata changes during an update.
CI then runs a normal `brew install --formula` from the real tap on `macos-26`
arm64 and requires a bottle receipt; a source fallback cannot pass that gate.
The public bottle `root_url` is the same non-latest release as the source
archive. After the pour, CI reads back and verifies the signed manifest, its
assets, and the tested formula before promoting the release to latest.

Initial maintainer setup requires explicit publication approval:

- Create the public `bassner/homebrew-tap` repository, with its default branch.
- Protect release tags (`v*`) and the `release` and `homebrew-tap` environments
  with trusted release permissions and required reviewers. Never expose secrets
  to pull request or fork workflows.
- Set `CLAUDODEX_RELEASE_SIGNING_KEY` in the protected `release` environment to
  a base64 Ed25519 seed or private key matching the embedded trust root. Generate
  and store it privately; the helper rejects a different key without printing it.
- Set `HOMEBREW_TAP_TOKEN` in `homebrew-tap`, scoped to Contents write on only
  `bassner/homebrew-tap`. The tap job has no release signing key.
- Review `packaging/release-notes/v<VERSION>.md`, then approve and push the tag.

Every channel uses signed metadata; no standalone payload installer or native
rollback channel is shipped. A trust-root change must first ship in a verified
package. Key loss or compromise requires manual recovery through a separately
verified package update.

Keep the signed metadata envelope stable across schema versions: `schema`,
`repository`, `channel`, `key_id`, `version`, `release_notes`, and the `artifacts`
array must retain their names and types. Preserve the manifest, note, and
artifact-count bounds and the artifact name/kind, size, and lowercase SHA256
formats in [manifest.go](internal/selfupdate/manifest.go). Schema 1 clients tolerate
additive JSON fields and new artifact kinds. A newer schema is authenticated
before the client shows a manual `brew upgrade bassner/tap/claudodex` warning;
unsupported metadata is never accepted for an update. These stable fields and
bounds let older clients reach that recovery message.

If a job fails, the prior latest stays available. Rerun the failed workflow:
identical assets are reused, different bytes are rejected, and tap commits use
normal fast-forward pushes. Do not delete release assets to hide a mismatch or
force-revert unrelated tap commits. A tap conflict requires review. Before the
first promotion, also verify a CI-built macOS 15 bottle pours on the maintainer's
current macOS without compilation; a local macOS 26 build alone does not prove
this compatibility.

Compatibility caches retain the current Claudodex version and model mapping.
Alternating between builds or model overrides can rebuild and re-sign a cached
Claude copy. Copies used by live sessions remain available until their leases end.

## Pull Requests

Good pull requests include:

- a short explanation of the compatibility issue being fixed,
- tests or fixtures for the changed request/response shape,
- the Claude Code version involved when relevant,
- the Codex model or endpoint involved when relevant,
- confirmation that `go test ./...` and `go vet ./...` pass.

For Claude Code UI/binary patches, include:

- Claude Code version,
- OS and architecture,
- binary SHA256,
- exact strings or byte ranges being patched,
- behavior when the version or SHA does not match.

## Security And Privacy

Never commit OAuth tokens, Claude credentials, local sidecar state, transcripts,
or logs. If a bug report requires request or response examples, redact account
IDs, access tokens, installation IDs, paths containing private names, and any
project-sensitive content.

Use `CLAUDODEX_PROXY_LOG` and `CLAUDODEX_OAUTH_PROXY_LOG` only for local
diagnostics, and review logs before sharing them.
