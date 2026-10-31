# Agent Handoff Guide

This file is for coding agents working on Claudodex. It should stay public-safe:
do not add personal machine paths, credentials, tokens, private transcripts, or
environment-specific debugging output.

## Project Mission

Claudodex has one job: make the real Claude Code binary work well with OpenAI
Codex subscription models.

Keep the project narrow:

- This is Claude Code routed to Codex, not Codex routed to Anthropic.
- Do not turn Claudodex into a generic multi-provider proxy.
- Do not replace Claude Code's terminal UI or workflow.
- Prefer compatibility with normal Claude Code behavior over custom UX.

## Non-Negotiable Invariants

- Normal Claude Code flags must pass through unchanged unless Claudodex must
  adapt a model/config flag for compatibility.
- `clx:*` commands are Claudodex-specific commands. Keep them separate from
  Claude Code slash commands.
- Never send Anthropic payment, account, or subscription headers upstream.
- Keep OpenAI/Codex auth isolated from normal Claude Code auth.
- Preserve normal Claude Code settings semantics. User-facing Claude settings
  live under the normal Claude config locations; the Claudodex sidecar is an
  implementation detail.
- Resolve symlinks before editing or reporting config/instruction files. Do not
  edit sidecar paths as if they were canonical user files.
- Do not edit user statusline scripts. Statusline compatibility must use
  Claudodex-owned wrapper/overlay behavior.
- Prefer live Codex metadata over hardcoded model limits whenever the data is
  available.

## Architecture Map

- `cmd/claudodex`: command entrypoint.
- `internal/app`: CLI command routing, `clx:*` commands, and Claudodex model
  override flags.
- `internal/launcher`: launches the real `claude` binary, prepares environment
  variables, sidecar config, OAuth compatibility proxy, statusline compatibility,
  model capability cache, settings sync, and binary UI patches.
- `internal/proxy`: local Anthropic-compatible HTTP API consumed by Claude Code.
- `internal/convert`: request, stream, tool-call, usage, billing, and reasoning
  effort conversion.
- `internal/codex`: Codex backend client, model metadata, usage, SSE, and
  WebSocket handling.
- `internal/modelconfig`: default model family mapping and override handling.
- `internal/auth`: OpenAI OAuth, token storage, refresh, revocation, and
  installation identity.

## External References

When behavior is unclear, verify against the current installed `claude` binary
first. Source-level references are useful, but they can be stale or incomplete.

Useful public references:

- OpenAI Codex source: `https://github.com/openai/codex`
- Claude Code source, if published or mirrored publicly:
  `https://github.com/anthropics/claude-code` and
  `https://github.com/yasasbanukaofficial/claude-code`

Do not vendor external repositories into this repo, copy proprietary code into
Claudodex, or treat a reference checkout as more authoritative than observed
runtime behavior from the installed Claude Code version.

## Model And Context Rules

Default model families are centralized in `internal/modelconfig` and may be
overridden at startup. Keep all routing, `/v1/models`, model argument rewriting,
capability cache, context-window handling, statusline behavior, auto-compact
limits, and settings normalization consistent with those defaults and overrides.

Runtime family mapping comes from the live Codex catalog:

- Opus routes to Sol, Sonnet to Terra, Haiku to Luna, and Fable to Astra.
- Select the newest compatible numeric generation within each family. Never
  cross families or silently replace an unavailable explicit model pin.
- Track explicit pins independently of the comparable model configuration,
  including pins equal to a compiled compatibility default. Direct GPT IDs in
  flags, settings and resumed requests stay exact.
- Derive ordinary reasoning efforts, defaults and fast-mode support from the
  selected model's metadata. Clamp unsupported efforts. `ultracode` workflow
  orchestration remains owned by Claude Code; never select Codex `ultra`.
- Resolve models once before settings and proxies. Every consumer must share
  that snapshot. Refresh metadata on every launch; only temporary failures may
  use a recent account/backend/region/protocol-scoped cache, with a warning.

Do not hardcode context windows such as `200k`, `272k`, or `1m` as source of
truth. Fetch Codex model metadata and derive context-window behavior from it.
UI labels and real context windows are separate concerns.

## Streaming And Tool Calls

Long tool calls and streamed JSON arguments are fragile. Be conservative:

- Keep Claude Code tool search disabled until Claudodex translates Anthropic
  `tool_reference` blocks end to end. Normal complete tool definitions remain
  supported.
- Schema-backed tool calls must remain pending until arguments are complete.
- Do not emit partial tool blocks just because partial argument deltas arrived.
- Preserve retryability across transient stream resets, unexpected EOFs, broken
  pipes, and implicit resume paths.
- Preserve session resume in both directions between normal Claude Code and
  Claudodex. Provider-specific thinking signatures or replay data must remain
  optional and must not prevent either client from resuming the other's
  transcript. Verify both directions with the real Claude binary before
  changing thinking traces or transcript signatures.
- Use trace fields such as `tool_arg_delta_events` and `tool_arg_delta_bytes`
  when diagnosing turns that look idle during long tool-argument generation.

## Settings And Sidecar Rules

Claudodex uses a Claude Code compatibility sidecar, but the sidecar is not the
source of truth for user-facing settings.

- Treat sidecar config as implementation-owned compatibility state.
- Keep Claudodex fake Claude OAuth state isolated from real Claude auth.
- Keep normal Claude Code settings shared where possible.
- Reconcile non-auth settings before launch, while Claude is running, and at
  shutdown when that path is involved.
- Restore the user's original shell and proxy environment for Claude Code tool
  subprocesses so tools do not inherit Claudodex's internal proxy settings.

## OAuth And Proxy Pitfalls

Claude Code may still make first-party HTTPS calls to Anthropic API hosts.
Claudodex intercepts the required compatibility routes locally and forwards them
to the Codex-backed proxy where appropriate.

Important pitfalls:

- The local HTTPS compatibility certificate must cover long-running and resumed
  sessions. Do not shorten its lifetime casually.
- Client errors shown by Claude Code are not always Codex upstream errors.
  Diagnose whether an error came from Claude Code, the local OAuth proxy, the
  local Anthropic-compatible proxy, or the Codex backend.
- Keep internal proxy routes loopback-only unless there is a deliberate design
  and security review for exposing them elsewhere.

## Binary UI Patch Rules

Claude Code binary patching is versioned compatibility code. Message launches
select a supported local original and apply the current compatibility patch.

- Patch only verified OS, architecture, Claude Code version, and binary SHA
  combinations.
- Add one patch implementation per supported Claude Code version.
- Select an exact verified original source and prepare its current patch before
  auth, settings, or proxy setup. Use that source for all version decisions.
- If installed Claude is unsupported, use a verified locally retained source
  without changing system Claude. If none works, fail with an exact supported
  version and install command. Only `CLAUDODEX_DISABLE_CLAUDE_PATCH=1` explicitly
  permits an unpatched message launch. Help and administrative commands pass
  through before this strict check.
- Verify source and signed patched-cache hashes on reuse. Keep private source
  snapshots and live process leases; never distribute cached Claude binaries.
- UI patches may improve branding, model picker labels, and header text, but
  critical request routing must stay outside binary patching.
- When updating a patch, document the exact Claude Code version and binary hash
  in code/tests.

## Testing Expectations

For normal Go changes:

```sh
git ls-files '*.go' | xargs gofmt -w
go test -count=1 ./...
go vet ./...
go build -o ./claudodex ./cmd/claudodex
```

For launcher, UI, settings sync, model picker, statusline, context window,
`/fast`, background agents, subprocess environment, or binary-patch behavior,
also test the real `claude` binary in a terminal multiplexer such as `tmux`.
Do not rely on static inspection alone for Claude Code runtime behavior.

Installed-Claude smoke tests are available behind the existing opt-in test
environment variable. Live tests against Codex require a local OpenAI login via
Claudodex.

## Diagnostics

Useful local diagnostics:

- `claudodex clx:doctor`
- `claudodex clx:usage`
- `CLAUDODEX_PROXY_LOG`
- `CLAUDODEX_OAUTH_PROXY_LOG`
- Claude Code transcript JSONL files

Never commit diagnostic logs, transcripts, OAuth tokens, local sidecar state, or
generated binaries. Redact account IDs, tokens, installation IDs, private paths,
and project-sensitive content before sharing bug evidence.

## Repo Hygiene

- Keep public-facing user documentation in `README.md`.
- Keep contributor process documentation in `CONTRIBUTING.md`.
- Keep agent operational guidance in this file.
- Do not commit build artifacts, auth files, logs, transcripts, or local Claude
  runtime state.

When in doubt, preserve Claude Code behavior first, keep Claudodex focused on
Codex subscription compatibility, and verify the exact runtime path being
changed.
