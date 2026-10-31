# Effortless Claudodex

Make the normal path: install with Homebrew, run `claudodex`, and reach the real
Claude Code UI with a working Codex subscription and supported models.

Keep the existing focus: Claude Code routed to Codex subscription models. Keep
the four roles: Haiku → Luna, Sonnet → Terra, Opus → Sol, Fable → Astra. Preserve
normal Claude settings, flags, tools, permissions, and explicit user choices.

## Work items

1. **Reuse an existing local Codex login.** Prefer a direct Claudodex login. If
   none exists, reuse an eligible local Codex subscription login before opening
   a browser. Keep Codex responsible for refreshing its own tokens. Do not copy
   rotating refresh tokens into a second credential store. Logout must not log
   the user out of Codex or immediately sign them back into Claudodex.
2. **Discover model routes and capabilities together.** Read the live catalog
   on each launch. Resolve available successors within Luna, Terra, Sol and
   Astra. Use the same resolved configuration for routing, labels, context
   windows, reasoning efforts and capability caches. Preserve explicit pins,
   even when a pin equals a compiled default.
3. **Recover from temporary catalog outages.** Keep a recent catalog scoped to
   the account, region, backend and discovery protocol. Refresh on every launch.
   Use the cached copy only for temporary transport/server failures, with a
   clear warning. Never conceal invalid credentials or a model removal.
4. **Preserve complete replay data across process restarts.** Carry a bounded,
   versioned opaque replay envelope through Claude transcripts. Preserve ordered
   reasoning and tool state across restart, resume and compaction. Validate
   foreign input and account/model boundaries before replay. Normal Claude Code
   must be able to resume a Claudodex session, and Claudodex must be able to
   resume a normal Claude Code session. Keep transcripts compatible in both
   directions: provider-specific replay data must be optional and safely
   ignored when switching providers. Test both directions with the real binary
   before changing thinking traces or transcript signatures.
5. **Add public compatibility checks.** Add push/PR Go and deterministic
   protocol checks and scheduled startup checks for pinned/latest Claude Code.
   Cover supported platforms. Keep native and live release gates. Do not rely
   only on maintenance running on a contributor's Mac.
6. **Make startup diagnosis actionable.** Extend `clx:doctor` to explain the
   authentication source, resolved models and capabilities, selected verified
   Claude source, catalog age, installation method and update state. Give one
   precise recovery action when startup cannot continue.

Items 1–3 are authorized for this implementation. Items 4–6 remain planned.

## Implementation of 1–3

### Authentication contract

- Add one authentication provider used by every launcher, proxy, status,
  doctor, usage and `clx:serve` credential path, including forced refresh after
  HTTP 401. It exposes current/fresh credentials, rejected-token refresh,
  user/account identity and destination constraints. `Store` and `Refresher`
  remain direct-Claudodex-login components; borrowed sources never enter them.
- Prefer an existing direct Claudodex login. If none exists and reuse has not
  been disabled, resolve native Codex from `CODEX_HOME`, otherwise `.codex`
  under the user home. Keep Claudodex's own home override independent.
- Use native `codex app-server` for credential authority for supported file and
  keyring logins. The installed CLI 0.159.2 was verified to support
  `config/read`, subscription `account/read`, and the legacy `getAuthStatus`
  method with `{includeToken:true,refreshToken:false}`. That method returns
  the current access token only, not a refresh token. It is not included in
  the generated v2 schemas, so probe this capability instead of assuming that
  generated schemas describe every native method. If unsupported, retain the
  normal direct browser-login path; never scrape secure stores or stale files.
- Persist only a private source reference and expected user/account identity
  in a separate `codex-login.json`, never in direct `auth.json`. Keep borrowed
  access credentials in memory only. Never extract, copy or persist Codex's
  refresh token or copy credential files/keyring data.
- Resolve and capability-check the native binary before adopting a source.
  Bound process time, total output and message size; close pipes and terminate
  owned processes on errors or cancellation. Request no model thread,
  generation or tool execution. Native logs/state and native credential refresh
  writes are allowed; Claudodex creates no lock/temp files under `CODEX_HOME`.
- Pin official subscription endpoints and pass canonical `CODEX_HOME`
  explicitly. Remove API-key/base-URL overrides plus
  `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE`, and
  `CODEX_APP_SERVER_LOGIN_CLIENT_ID`. Validate effective provider, endpoint and
  login settings before requesting credentials. Never print raw native config,
  tokens or error payloads. Native account type and auth method must be ChatGPT;
  reject API-key, workload/external-token and unsupported modes.
- Derive account and user IDs, expiry and residency from the returned access
  token's claims. Require usable IDs and expiry and an expected OpenAI issuer.
  Match native account email with token email if both are available. Free
  accounts are eligible when native ChatGPT auth and catalog access are present;
  do not hardcode a changing paid-plan allowlist.
- Residency must be known. The installed access token has account/user IDs and
  expiry but lacks `chatgpt_account_is_fedramp`. Accept an explicit residency
  claim when present. Otherwise, only when native effective storage is `file`,
  read its ID-token metadata, bind both user/account IDs to the native access
  token, and use Codex's verified ID-token interpretation of the FedRAMP flag.
  Do not extract refresh tokens. Retry changing/truncated metadata briefly.
  Reject conflicting/unknown residency. Keyring-only logins without an exported
  residency signal retain ordinary browser OAuth; do not guess or read a stale
  fallback file. A future native residency export can remove this limit.
- Use bounded short-lived native reads rather than a long-lived native auth
  cache that can conceal account changes. Coalesce concurrent reads into one
  in-flight result. A shared in-memory result may be at most two seconds old,
  including before reusing a WebSocket; never use that cache for HTTP 401 or
  refresh. Pin running proxies to their startup user and
  account. A changed source/account fails clearly, even between launches,
  until `clx:auth-login --codex` explicitly adopts the current source.
- If startup metadata rejects an access token with HTTP 401, refresh once
  through that same provider using the rejected token and retry discovery.
  Require unchanged user/account identity before the retry; do not refresh for
  HTTP 403 or hide either rejection behind cached metadata.
- Measured on installed Codex 0.159.2: five separate native initialize/config/
  account/access-token reads, including process cleanup, took 1378.9, 554.2,
  576.7, 864.6 and 515.5 ms (median 576.7 ms, maximum 1378.9 ms). Target a
  1.5-second cold-read budget on this machine; the two-second memory age and
  in-flight coalescing avoid putting this cost on every concurrent request.
- Never force refresh at startup if the token is fresh. Serialize all borrowed
  native reads and refreshes with an in-process guard and a cross-process lock
  under Claudodex state, since the native authority can proactively refresh
  while reading. Re-read the authoritative native token under the lock and use one
  already rotated by Codex or another session. On HTTP 401, request refresh
  only if the rejected access token is still current. Let native
  `account/read {refreshToken:true}` refresh, then retrieve the new access
  token with `getAuthStatus`. Check the same user/account before and after;
  an unchanged rejected token or expired result is not refresh success.
  Native Codex owns its own source locking and rotation.
- Native Codex uses guarded reload, but a separate Codex process can refresh
  concurrently with Claudodex. This residual first-party coordination race is
  a known limit; never introduce an independent refresh-token writer.
- Borrowed credentials may be sent only to the official Codex backend with
  matching residency behavior. Carry the allowed origin with credentials and
  enforce it at the Codex client boundary, including redirects, metadata,
  usage, workspace discovery, HTTP and WebSocket requests. Reject custom
  backend/token endpoints for borrowed sources. Custom backend tests use
  explicit synthetic direct Claudodex credentials. Keep servers loopback-only.
- Every logout, direct or borrowed, disables ambient reuse and removes the
  source reference. Revoke only direct Claudodex credentials; never revoke,
  delete or alter Codex's login. `clx:auth-login --codex` explicitly clears
  suppression and adopts an eligible Codex source. When a direct login is
  active, require logout first. Ordinary `clx:auth-login` keeps its existing
  browser/direct-login behavior and clears suppression on success.
- A permanently invalid direct login may fall through to an eligible Codex
  source on the same launch only if reuse has not been disabled. Transient
  direct-auth or borrowed-refresh failures must not switch sources. An expired
  token still blocks startup during a refresh outage; catalog fallback cannot
  fix that. Status and usage must use this same provider.

### Catalog contract

- Keep `modelconfig.Config` as the comparable four-target value used by binary
  patch identity. Track explicit pins separately, including per-role flags and
  `--claudodex-models`. A caller-supplied model override is also explicit.
- Add one catalog service for fetching, validation, resolution and caching.
  Extend parsed backend metadata with catalog priority, minimum client version,
  default reasoning effort and relevant speed capabilities.
- Automatic candidates must be advertised for API use, visible, compatible
  with the discovery protocol and members of the exact requested model family.
  Select the newest numeric generation within that family, then backend
  priority, then a stable slug tie-break. Never cross families. Reject duplicate
  identities, invalid model IDs and unusable selected capabilities.
- A pin selects that exact available model. Missing or unsupported pins get an
  actionable error and are never silently remapped. Retain compiled targets
  only as compatibility defaults for helpers without an actual catalog.
- Direct model slugs from `--model`, saved Claude settings or resumed requests
  also mean that exact catalog model, even when they equal an old compiled
  default. Preserve them through settings normalization and argument rewriting.
  Validate their own context/effort metadata and never fall back to Opus for an
  unknown direct GPT model. Role aliases follow automatic role resolution.
  Automatic IDs use `^gpt-[0-9]+(\.[0-9]+)*-(luna|terra|sol|astra)$`; suffixed
  variants are explicit-only. Both pins and automatic choices require API
  access and visible entries. Lower backend priority numbers win ties.
- Missing any of the four automatic families blocks startup with an actionable
  override message. Do not silently assign another family or keep a hidden
  compiled default. This preserves the existing four-role contract.
- Derive request efforts from each selected model's advertised effort set.
  Clamp unsupported requests to the highest supported effort no greater than
  the requested effort, or the lowest supported effort if every level is above
  it. Order: `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max`. Ignore
  unknown levels for automatic selection; never select Codex `ultra` workflow
  orchestration. Reject a model with no supported ordinary effort. Use the
  model's supported default for omitted effort or `auto`, with a conservative
  supported fallback if its default is unknown. Apply the same clamp after
  budget-token mapping. Preserve
  `ultracode` orchestration in Claude Code. Runtime catalog paths must not rely
  on a hardcoded `max`/`xhigh` model-name list.
- Send the priority service tier only when the requested model advertises fast
  mode. Inherited/subagent fast settings on unsupported targets must use the
  standard tier with a trace field. `speed` alone does not identify an explicit
  user choice, so do not reject requests solely because it is `fast`. Reject
  only when an independent explicit user request can be proven. Verify actual
  inherited/subagent `speed` behavior with the real Claude binary in tmux.
  Derive UI effort defaults from metadata while preserving explicit choices.
- Preserve native CLI, environment, top-level and per-model effort preferences.
  Claude's native effort controls cannot directly express `none` or `minimal`.
  For an automatic default, select a native effort only when conversion maps it
  exactly to the catalog default. If no native choice can express it, report an
  actionable error asking for an explicit effort instead of silently changing
  the default. Direct API request conversion still supports these levels.
- Keep the shared auto-compaction threshold safe for every selectable model's
  ordinary and long-context form, including additional direct models and their
  effective percentages or explicit compact limits. This conservative common
  threshold can compact early when switching models; keep each model's true
  context window in capabilities and statuslines.
- Share one resolved result across the launcher, local proxies, `clx:serve`,
  model arguments, statusline and capability caches. Resolve before creating
  the sidecar or starting proxies.
- Preserve verified-Claude selection and initial patch preparation before
  authentication, as the project requires. Disable retention during this first
  phase so it cannot delete a prior final-model patch. If discovery changes
  targets, prepare/reselect using the same verified original with the final
  configuration. Run retention only with the final config, while both leases
  are active, so the default and final patches remain reusable without another
  copy/sign on each launch. Try other exact verified retained sources if a
  longer final label does not fit; if none works, fail with a clear model-pin/
  compatibility message. Test long labels and repeated launches. Hold leases
  until the child exits. Keep existing patch identity stable; do not rewrite
  historical patch implementations in this task. Never make routing depend on
  binary patch substitutions or change system Claude.
- Native acceptance found that Claude 2.1.314/2.1.316's configured-model
  recognizer ignores additional catalog models and custom picker options. Add
  a bounded UI overlay for those exact registered hashes, after their existing
  patch, to recognize a launcher-owned catalog allowlist. Keep the four role
  targets and exact request routing unchanged. Unknown GPT IDs still fail
  catalog validation. Verify older explicit GPT choices with native Claude.

### Cache and error contract

- Always attempt a fresh metadata request, retaining the existing bounded
  startup timeout. Cache parsed metadata only; never authentication data.
- Store a versioned private JSON file under Claudodex state using an atomic
  write. Key it by a hash of account identity, region, normalized backend and
  discovery protocol, using the actual `client_version` sent on the wire and
  both user/account IDs. Do not share a cache when identity is unknown.
- Accept a cache at most 24 hours old. Reject future timestamps, invalid scope,
  corrupt/oversized files, unsafe paths and unusable selected model metadata.
- Allow fallback for transport failures, the metadata request's own timeout,
  HTTP 408/429 and HTTP 5xx. Honor parent cancellation without fallback.
  Reject 401/403, other permanent HTTP errors, malformed catalogs and confirmed
  missing models. Invalidate or replace old data after authoritative rejection
  so a later outage cannot bring back a confirmed unavailable model.
- A successful response is authoritative. An optional cache-write failure must
  not prevent an otherwise valid launch. Warn on fallback and report its age;
  do not silently present cached metadata as freshly discovered.

## Validation and delivery

- Review the overall architecture before implementation and review the complete
  implementation independently afterward. Resolve all findings in the same
  review conversation. Keep unrelated checkout changes untouched.
- Add focused tests for auth precedence, source references, account changes,
  refresh ownership, logout suppression, destination constraints, explicit pins,
  future model generations, effort metadata, cache isolation/age/corruption,
  transient fallback and permanent rejection.
- Run formatting, `go test -count=1 ./...`, `go vet ./...`, and a Go build.
- Run installed-Claude smoke tests and real terminal checks. Exercise initial
  login reuse without copying credentials, all four resolved model labels,
  supported efforts and fallback startup with a controlled temporary catalog
  failure. Use synthetic upstream responses for deterministic native tests.
- Update README and agent model guidance to describe the final behavior and
  supported authentication sources. Record precise limitations and test evidence.
- Keep releases and Homebrew installation unchanged until a separate release
  action is requested. Do not publish or replace the installed tool here.

## Status

- [x] Record all six work items and the implementation contracts.
- [x] Obtain independent architecture approval (including native-read timing
  and inherited fast-mode conditions).
- [x] Implement local Codex login reuse.
- [x] Implement catalog-driven routing and capabilities.
- [x] Implement scoped catalog caching and transient fallback.
- [x] Complete automated, installed-Claude and terminal verification.
- [x] Complete independent implementation review and update this status.

Validation completed on Claude Code 2.1.316: 2,151 Go test cases and 2,058
affected-package race test cases passed. Twelve new native startup/effort
launches and a tmux Agent flow passed with synthetic upstream generation.
Thirty-six existing native regression cases passed, including the corrected
catalog-default expectation in the unpatched smoke test. Formatting, vet and
build passed. Native Codex login reuse and live metadata were verified without
model generation or copying refresh tokens.

The overall plan was approved by an independent Claude Opus review. Its final
implementation review hit a quota limit; the required fresh Astra fallback
review approved items 1–3 after all findings were fixed. Items 4–6 remain planned.
Items 1–3 are prepared for the separately authorized v0.5.0 release. Publication
and installation are verified by the release workflow and local acceptance run.
