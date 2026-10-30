# Homebrew packaging

The tap target is `bassner/homebrew-tap`, formula `Formula/claudodex.rb`.
`cmd/clx-release` generates the formula from the uploaded source archive and
Homebrew's bottle JSON. There is no prebuilt-binary cask or `bin.install` path.
Do not change quarantine attributes or Gatekeeper settings.

Prepare from a clean, reviewed checkout. The source command uses `git archive`
and runs `go mod vendor` only in a temporary directory. Output is deterministic
for the same tracked source and dependency versions; it never includes dirty or
untracked files. Build and test the extracted archive with `GOPROXY=off`,
`GOTOOLCHAIN=local`, `GOWORK=off`, and `-mod=vendor` before uploading it.

```sh
go build -o /tmp/clx-release ./cmd/clx-release
/tmp/clx-release source --version 0.3.40 --out /tmp/clx-assets
/tmp/clx-release formula --version 0.3.40 \
  --source /tmp/clx-assets/claudodex-0.3.40-source.tar.gz \
  --out /tmp/claudodex.rb
```

The generated formula pins the uploaded source URL and SHA256, compiles offline,
and records `main.version` and `main.installMethod=homebrew` in Go build info.
Version checks before runtime handoff read those build settings without running
the candidate executable. Do not add Go's `-trimpath` flag or `std_go_args`:
[Go issue 52372](https://go.dev/issue/52372) explains why that removes recorded
linker flags. The formula sets `GOFLAGS` explicitly so user defaults cannot
remove this metadata. Claude itself is not packaged.

In a disposable tap/build environment, install that formula with
`brew install --build-bottle bassner/tap/claudodex`, run `brew test`, then:

```sh
brew bottle --json --no-rebuild \
  --root-url https://github.com/bassner/claudodex/releases/download/v0.3.40 \
  bassner/tap/claudodex
/tmp/clx-release bottles --version 0.3.40 \
  --bottle-json ./claudodex--0.3.40.arm64_sequoia.bottle.json \
  --asset-dir . --out /tmp/clx-assets
/tmp/clx-release formula --version 0.3.40 \
  --source /tmp/clx-assets/claudodex-0.3.40-source.tar.gz \
  --bottle-json /tmp/clx-assets/claudodex-bottles.json \
  --out /tmp/claudodex-with-bottles.rb
```

`bottles` checks the actual file checksum and copies it to the public filename
from Homebrew JSON. Homebrew's local filename can contain a double hyphen while
its download filename contains one. The command also writes minimal public
metadata; runner paths and detailed build records are not published. Do not
guess a bottle filename or tag.

After source and bottle tests, sign the notes and asset identities. Supply the
key through `CLAUDODEX_RELEASE_SIGNING_KEY` in the protected CI step; do not put
it in a command, argument, file in this repo, or log.

```sh
/tmp/clx-release manifest --version 0.3.40 \
  --notes packaging/release-notes/v0.3.40.md \
  --source /tmp/clx-assets/claudodex-0.3.40-source.tar.gz \
  --bottle-json /tmp/clx-assets/claudodex-bottles.json \
  --asset-dir /tmp/clx-assets --out /tmp/clx-signed
/tmp/clx-release verify-manifest --version 0.3.40 \
  --asset-dir /tmp/clx-assets --out /tmp/clx-signed
```

The helper checks the signing key against `selfupdate.TrustedKeys()` and verifies
its own output. Inline release notes, source/bottle SHA256, and byte sizes are
signed. The updater displays these signed notes, independent of later edits to
the GitHub release description.

When changing metadata, follow the
[release compatibility rules](../../CONTRIBUTING.md#releases)
so installed clients can still discover an update.

The workflow is the publication path. It uploads source, bottles, and signed
metadata without replacement before publishing one tap formula file with a
scoped token. Pinned signed notes are then available if a user's Homebrew
metadata changes before the update is announced as latest. CI uses a normal
`brew install --formula` from the real tap and checks its bottle receipt. Only
after that pour does it read back and verify the signed manifest, asset hashes,
and tested formula, then mark latest. Verification checks the expected version
and every signed asset's size and SHA256. Publication, tap creation, and secret
configuration require maintainer approval.
