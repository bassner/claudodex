<p align="center">
  <img src="assets/claudodex-logo.svg" alt="Claudodex" width="620">
</p>

<p align="center">
  <strong>Claude Code, routed through your OpenAI Codex subscription.</strong>
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-0f766e"></a>
  <img alt="Language: Go" src="https://img.shields.io/badge/language-Go-2563eb">
  <img alt="Status: experimental" src="https://img.shields.io/badge/status-experimental-b45309">
</p>

Claudodex is a focused launcher and compatibility proxy for running the real
`claude` binary against OpenAI Codex subscription models. It is not a
multi-provider proxy, not a Codex CLI wrapper, and not a replacement terminal
UI. You still use Claude Code. Claudodex swaps the model backend.

```text
you -> claudodex -> installed claude -> local Anthropic-compatible proxy -> Codex
```

## Quickstart

You need macOS on Apple Silicon, Homebrew, a working `claude` command, and an
OpenAI account with Codex subscription access.

```sh
brew install bassner/tap/claudodex
claudodex
```

Claudodex reuses an eligible local Codex login when available. Otherwise an
interactive launch opens the normal OpenAI sign-in flow. To sign in separately,
run `claudodex clx:auth-login`.

Homebrew uses a prebuilt bottle when available; otherwise it installs Go and
builds from pinned source. Claude Code is installed separately. If your Claude
version is unsupported, see [Compatibility](#compatibility).

## Why Claudodex?

There are already useful projects in this space, including Claudish,
ccproxy-api, claude-code-proxy variants, and Codex proxy experiments. Many of
them try to support a long list of providers, models, transports, and runtime
styles. That flexibility is valuable, but it can also make the Claude Code plus
Codex subscription path harder to install, harder to reason about, or easier to
miss in small compatibility details.

Claudodex deliberately takes the opposite shape: one launcher, one target use
case, and as little setup as possible. The goal is that a person with a Codex
subscription and an installed `claude` binary can log in once, run `claudodex`,
and get a Claude Code experience whose backend is Codex.

Why not call it `claudex`? Because that name is already used in multiple places.
`Claudodex` is clunkier, but it is specific and searchable.

## What It Does

- Runs your installed `claude` binary with normal Claude Code flags.
- Keeps subagent conversations separate using Claude Code's native worker IDs.
  Agent model overrides use exact Codex IDs; omitted models and `inherit` retain
  Claude Code's normal inheritance rules.
- Uses a direct OpenAI login or safely reuses an eligible local Codex login.
- Discovers the newest available numeric model generation within each family:
  Opus -> Sol, Sonnet -> Terra, Haiku -> Luna, Fable -> Astra. Explicit model
  choices stay pinned. All four families must be available or explicitly mapped.
- Derives reasoning efforts from the selected model's metadata. Unsupported
  efforts are clamped to an advertised level. Claude Code's `ultracode` keeps
  its own workflow orchestration and requests the supported equivalent of `max`.
  Models without fast-mode support use the standard service tier.
- Disables Claude Code's deferred tool-search protocol so complete tool
  definitions are sent through the Codex-compatible request path.
- Fetches live Codex model metadata on every launch for routing, context windows,
  capabilities and auto-compact limits. A temporary network or server failure can
  use this account's cached catalog for up to 24 hours, with a warning. Invalid
  credentials, removed models and malformed responses do not use this fallback.
- Uses the token usage reported by completed OpenAI responses for Claude Code's
  context accounting. Local preflight estimates exclude base64 payload text and
  count image tokens from the decoded pixel dimensions instead.
- Shows OpenAI reasoning summaries through Claude Code's native thinking UI,
  with each titled summary section kept as a separate thinking entry.
- Preserves encrypted OpenAI reasoning items for lossless stateless tool-call
  continuations while keeping Responses API storage disabled.
- Keeps Claude Code settings shared with normal Claude Code while keeping
  Claudodex auth isolated.
- Sets Claude privacy flags by default:
  `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`, `DISABLE_TELEMETRY=1`,
  `DO_NOT_TRACK=1`, and `DISABLE_GROWTHBOOK=1`.
- Provides Codex-backed usage through `claudodex clx:usage`.

## Compatibility

Claudodex is experimental. Each compatibility patch targets a specific Claude
Code version and binary checksum for an operating system and CPU architecture.
Current verified patches cover macOS on Apple Silicon, including Claude Code
2.1.318, 2.1.319, 2.1.322 and 2.1.323. If the installed Claude version has no supported patch,
whether older or newer, Claudodex uses a verified local copy from its cache or
Claude's retained versions without changing system Claude. If none is available,
launch stops with recovery steps:

```sh
claude install 2.1.323
claudodex clx:doctor
```

Claudodex does not download Claude or create patches automatically.
`CLAUDODEX_DISABLE_CLAUDE_PATCH=1 claudodex` explicitly permits unpatched Claude;
that combination and other platforms are unverified. Help, version, and Claude's
administrative subcommands remain available for repair.

## Updates

Top-level launches check signed GitHub release metadata before starting Claude.
Automatic prompts require standard input, output, and error to be terminals.
For Homebrew installations, an interactive launch shows the installed and
available versions, signed release notes, and release URL, then asks:

```text
Update Claudodex now? [y/N]
```

The default is No. Declining, empty input, EOF, or cancellation keeps the current
version; the next launch checks again. After Yes, Claudodex refreshes Homebrew
metadata and runs `brew upgrade --formula --yes bassner/tap/claudodex`. Homebrew
manages required dependencies. A source fallback may install Go and take longer.
The accepted version takes effect in the same invocation, with the original
Claude arguments. Running sessions keep their pinned statusline helper.

Automatic checks in piped and one-shot runs only print an update notice when a
newer version is available. They do not prompt or replace the running program.
Tool invocations inside a Claudodex session skip the update check. Source and
unknown installations also notify only; rebuild them manually. No scheduler is
installed.

```sh
claudodex clx:update        # Check now and ask on a terminal
claudodex clx:update --yes  # Explicit script consent for a Homebrew update
```

Set `CLAUDODEX_NO_AUTO_UPDATE=1` to skip launch checks. An offline or failed check
keeps the current runtime; `clx:update` reports check errors. If signed update
information requires a newer client, follow the displayed manual Homebrew
upgrade command.

Update metadata uses an embedded Ed25519 public key. Install the initial package
from the trusted tap. A lost or compromised signing key requires a separately
verified package update; automatic key recovery is not supported.

## Install From Source

For a manual installation, use Go 1.25 or newer:

```sh
git clone https://github.com/bassner/claudodex.git
cd claudodex
go build -o claudodex ./cmd/claudodex
mkdir -p ~/.local/bin
install -m 0755 claudodex ~/.local/bin/claudodex
```

Ensure `~/.local/bin` is on your `PATH`. Source builds require manual updates:
pull the latest changes, rebuild, and repeat the install step. When moving to
Homebrew, check `command -v claudodex` and remove any older source copy that
shadows the Homebrew installation.

## Usage

All normal Claude Code arguments are passed through:

```sh
claudodex -p "Reply with exactly: ok"
claudodex --model opus
claudodex --continue
```

## Commands

```sh
claudodex clx:auth-login
claudodex clx:auth-login --codex
claudodex clx:auth-status
claudodex clx:auth-logout
claudodex clx:version
claudodex clx:doctor
claudodex clx:usage
claudodex clx:update
claudodex clx:update --yes
claudodex clx:reset-installation-id
claudodex clx:serve --host 127.0.0.1 --port 0
claudodex [normal claude args...]
```

`claudodex --help` and `claudodex --version` intentionally execute `claude`
directly. Use `claudodex clx:version` for the Claudodex version.

## Login Reuse

A direct Claudodex login takes precedence. Without one, Claudodex asks the
installed Codex CLI for its current subscription access token. Codex keeps
ownership of token refresh. Claudodex stores only a private reference to that
login; it never copies its refresh token or credential file. `CODEX_HOME`, when
set, selects the native Codex home independently of Claudodex's own state.

Reuse requires a supported native credential-export method, a ChatGPT login and
known account residency. File-backed Codex logins can supply residency through
their matching ID-token metadata. Keyring logins without an exported residency
signal, API-key logins and unsupported native versions use ordinary browser
sign-in instead. Borrowed credentials can only reach the official Codex backend.

`clx:auth-logout` logs out of Claudodex and disables automatic login reuse. It
leaves Codex logged in. Run `clx:auth-login --codex` to enable reuse again or adopt
a changed Codex account; log out of a direct Claudodex login first. Running
sessions reject account changes and ask you to restart.

## Model Overrides

Role aliases follow live model discovery. These flags pin exact targets:

```sh
claudodex --claudodex-opus-model gpt-opus-next
claudodex --claudodex-sonnet-model gpt-sonnet-next --model sonnet
claudodex --claudodex-haiku-model gpt-haiku-next
claudodex --claudodex-fable-model gpt-astra-next
claudodex --claudodex-models opus=gpt-opus-next,sonnet=gpt-sonnet-next,haiku=gpt-haiku-next,fable=gpt-astra-next
```

Overrides affect routing, model argument rewriting, `/v1/models`, capability
cache, statusline context, auto-compact limits, effort handling, and settings
normalization. The selected Codex models must appear in live Codex model
metadata as visible API models with usable context and ordinary reasoning
capabilities. Missing pins fail with a recovery message. A direct GPT ID in
`--model`, saved settings or a resumed request also selects that exact model;
it does not follow automatic family upgrades.

Saved effort preferences and explicit flags take precedence over catalog
defaults. If Claude's native effort controls cannot express a model's default,
startup asks you to choose an explicit `--effort` value.

Claude uses one shared auto-compaction threshold for a running session.
Claudodex keeps it safe across selectable models and saved or resumed direct
choices. Mixed-model sessions can therefore compact before the largest model's
window is full. Each model still reports its own real context window.

## Settings And Privacy

Claudodex uses a Claude Code compatibility sidecar under `~/.claudodex`, but
normal Claude Code settings remain the source of truth where possible:

- If another Claude wrapper saved an unrecognized provider model, Claudodex uses
  its Opus route for that launch and leaves the shared model setting unchanged.
- Statusline commands receive the selected Codex model name through a
  Claudodex-owned wrapper; the original statusline script is not changed.
- `~/.claude/` files such as settings, agents, plugins, MCP config, and project
  state are symlinked into the sidecar.
- `~/.claude.json` is mirrored instead of symlinked because it also contains
  account and API-key state.
- Non-auth settings are reconciled before launch, while Claude is running, and
  during shutdown.
- Fake Claude OAuth state for the local compatibility layer stays isolated in
  Claudodex's sidecar.

Claudodex also restores the user's original proxy and shell environment for
Claude Code tool subprocesses, so Bash commands do not inherit Claudodex's
internal API proxy.

## Diagnostics

Interactive sessions write metadata traces to
`~/.claudodex/logs/proxy-trace-YYYY-MM-DD.jsonl`. On startup, Claudodex removes
managed daily trace files whose last write was more than seven days ago.
Set `CLAUDODEX_PROXY_TRACE=0` to disable tracing, or set it to a file path to
write a custom trace. Custom files have no automatic retention. Automation
does not enable tracing by default.

The interactive startup screen shows the version and progress while preparing
Claude Code, login, model metadata, settings, and local proxies. It disappears
when Claude Code starts. Warnings are retained in stderr history, and startup
errors remain visible. Print mode and
administrative commands keep their normal output.

On Claude Code 2.1.314, 2.1.316, 2.1.317, 2.1.318 and 2.1.319, auto-compaction
uses the active model's live Codex window, including its explicit `[1m]` route.
Switching to a model with a smaller window updates compaction accordingly.
Claude Code still reserves room
for its summary and output. Older supported Claude versions, or model limits
outside the native compaction schema, use a conservative shared ceiling with a
startup warning.

`clx:doctor` checks whether `claude` is installed, whether Claudodex auth is
present, and whether Codex usage is reachable. `clx:usage` prints Codex-backed
usage windows directly.

Claude Code's built-in `/usage` command is still subject to Claude Code's own
essential-traffic privacy gate. Claudodex keeps that privacy flag enabled, so
`clx:usage` is the reliable usage command for now.

Codex generation requests wait up to 45 seconds for upstream response headers
and retry that wait once. This does not limit an SSE response body after its
headers arrive. Set `CLAUDODEX_CODEX_RESPONSE_HEADER_TIMEOUT` to a duration
such as `90s`, a number of seconds, or `off` to override that behavior.

## Development

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

Live checks require a completed `clx:auth-login`:

```sh
./claudodex -p "Reply with exactly: ok"
./claudodex clx:usage
```

## Contributing

Contributions are welcome, especially compatibility fixes for new Claude Code
versions and Codex response shapes. Please read [CONTRIBUTING.md](CONTRIBUTING.md)
before opening a pull request.

## License

MIT. See [LICENSE](LICENSE).
