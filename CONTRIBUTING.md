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

## Checks

Run these before opening a pull request:

```sh
gofmt -w $(find . -path ./other_repos -prune -o -name '*.go' -print)
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
