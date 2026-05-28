# Claudodex

Claudodex is a focused Claude Code launcher/proxy for Codex subscription
models. It runs the installed `claude` binary, points it at a local
Anthropic-compatible proxy, and sends model requests to the Codex/ChatGPT
backend.

```text
human -> claudodex -> installed claude -> local proxy -> Codex Responses
```

It is intentionally not a multi-provider proxy and does not use Codex CLI as the
interactive runtime.

## Current Status

The Go implementation has the core path in place:

- first-run OpenAI OAuth token storage under `~/.claudodex`
- launcher pass-through for normal Claude Code flags
- required Claude privacy env vars set to `1`
- Claude settings shared with normal Claude Code while Claudodex auth stays
  isolated
- Opus/Sonnet/Haiku mapped by default to
  `gpt-5.5`/`gpt-5.4`/`gpt-5.4-mini`
- Claude-facing model IDs use Claude Code's `[1m]` suffix and
  `CLAUDE_CODE_AUTO_COMPACT_WINDOW` is capped from live Codex metadata, so
  auto-compact can trigger at the real Codex context limit without patching the
  Claude binary
- Claude-facing `/v1/models`, model capability cache, statusline context, and
  auto-compact limits all use the same live Codex model metadata
- shared Claude settings keep Claude-safe model aliases (`opus`, `sonnet`,
  `haiku`) even when Claudodex runs the corresponding Codex runtime models
- unknown models falling back to the configured Opus target
- Claude `max` effort mapped to Codex `xhigh`
- `POST /v1/messages` backed by Codex Responses HTTP SSE
- `/api/oauth/usage` backed by Codex `/wham/usage`
- fake-upstream tests for request conversion, streaming, usage, errors, and
  token refresh retry
- opt-in installed-`claude` smoke test for `claudodex -p` against a fake Codex
  upstream
- live installed-`claude` smoke coverage against a real Codex login when local
  credentials are available

The remaining documented caveat is Claude Code's own built-in interactive
`/usage` privacy gate. Claudodex's Codex-backed usage endpoint and `clx:usage`
are implemented and live-tested.

## Install From Source

```sh
go build ./cmd/claudodex
install -m 0755 claudodex /usr/local/bin/claudodex
```

`claude` must already be installed and callable on `PATH`.

## First Run

```sh
claudodex clx:auth-login
claudodex clx:doctor
claudodex
```

`clx:auth-login` opens OpenAI OAuth and stores tokens in Claudodex's own auth
file. It does not copy from or write to `~/.codex`. The browser flow uses the
registered Codex callback ports `1455`/`1457`, matching the working
Claudish/opencode OAuth shape.

## Commands

```sh
claudodex clx:auth-login
claudodex clx:auth-status
claudodex clx:auth-logout
claudodex clx:doctor
claudodex clx:usage
claudodex clx:reset-installation-id
claudodex [normal claude args...]
```

All other arguments are passed to `claude`. `claudodex --help` and
`claudodex --version` intentionally execute `claude` directly.

## Model Overrides

The default Codex targets are centralized in Claudodex's model catalog:

- Opus -> `gpt-5.5`
- Sonnet -> `gpt-5.4`
- Haiku -> `gpt-5.4-mini`

Override them at startup with Claudodex-only flags. These flags are consumed by
Claudodex and are not passed to `claude`:

```sh
claudodex --claudodex-opus-model gpt-opus-next
claudodex --claudodex-sonnet-model gpt-sonnet-next --model sonnet
claudodex --claudodex-haiku-model gpt-haiku-next
claudodex --claudodex-models opus=gpt-opus-next,sonnet=gpt-sonnet-next,haiku=gpt-haiku-next
```

Overrides affect request routing, Claude model arg rewriting, `/v1/models`,
model capability cache, statusline context, auto-compact limits, effort
handling, and shared settings normalization. The selected target models must be
present in live Codex `/codex/models` metadata with context-window information.

## Claude Code Settings

Claudodex uses Claude Code's normal settings surface. Files under
`~/.claude/`, such as `settings.json`, agents, plugins, MCP config, and project
state, are symlinked into Claudodex's Claude config sidecar so changes are
visible to normal `claude` and `claudodex`.

The global `~/.claude.json` file is mirrored instead of symlinked because it
also contains account and API-key state. Claudodex reconciles non-auth settings
both ways before launch, while Claude is running, and during shutdown, using
Claude-compatible lock files and a three-way merge. This matches Claude Code's
own multi-instance behavior: it writes global config under `<config>.lock` and
polls the file every second for changes from other running instances. Multiple
`claudodex` instances share the same sidecar and reconcile it instead of
resetting it.
When Claude Code's host-managed model picker reports that a new default model
was saved, Claudodex mirrors that transcript event back into shared settings as
the Claude-safe aliases `opus`, `sonnet`, or `haiku`.
Anthropic account/API-key fields stay in normal Claude Code; Claudodex keeps
its fake Claude OAuth state only in its sidecar. On macOS, Claudodex also sets
Claude secure storage to that sidecar and places a private `security` shim in
the Claude child `PATH` so Claude Code's own secure-storage layer uses the
sidecar `.credentials.json` fallback instead of prompting for a Keychain write
for the fake local OAuth credentials.

Claudodex also uses an internal HTTPS proxy so Claude Code's hardcoded
`api.anthropic.com` bootstrap calls are handled locally. Shell/tool subprocesses
restore the user's original proxy and shell environment before running, so Bash
commands do not inherit Claudodex's internal proxy.

## Diagnostics

`clx:doctor` checks:

- `claude` is installed
- `codex` is installed when available, as a warning only
- Claudodex auth is present
- Codex `/wham/usage` can be fetched and mapped when logged in

`clx:usage` prints the same Codex-backed usage mapping directly from
Claudodex. This remains available even when Claude Code suppresses
nonessential first-party UI calls because `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
is set.

## Verification

```sh
gofmt -w $(find . -path ./other_repos -prune -o -name '*.go' -print)
go test -count=1 ./...
go vet ./...
go build ./cmd/claudodex
CLAUDODEX_RUN_INSTALLED_CLAUDE_SMOKE=1 go test -count=1 ./internal/launcher -run TestInstalledClaudePrintSmokeWithFakeCodexUpstream -v
./claudodex -p "Reply with exactly: ok" --model claude-sonnet-4-6 --dangerously-skip-permissions --max-turns 2
./claudodex clx:usage
```

The installed-Claude smoke uses local fake Claudodex auth and a fake Codex
`/codex/responses` upstream. It does not require live Codex credentials.
The two `./claudodex` commands require `clx:auth-login` to have completed.

## Known Acceptance Gap

The proxy `/api/oauth/usage` route is implemented and tested against Codex
usage fixtures. Installed Claude Code 2.1.152 executes `/usage`, but its
first-party API helper returns `essential-traffic-only` before network when
`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1` is present. Claudodex keeps that
required privacy flag set, so the built-in interactive `/usage` panel remains
blocked by Claude Code itself. Use `clx:usage` until there is a supported way to
mark `/api/oauth/usage` as essential traffic or bypass that specific gate.
