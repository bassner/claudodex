#!/usr/bin/env bash
# GH_TOKEN must be scoped only to bassner/homebrew-tap contents. No force push.
set -euo pipefail
formula=${1:?formula file required}
version=${2:?release version required}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || exit 1
[[ -f "$formula" ]] || exit 1
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
git -c credential.helper='!gh auth git-credential' clone --quiet \
  https://github.com/bassner/homebrew-tap.git "$tmp/tap"
git -C "$tmp/tap" config credential.helper '!gh auth git-credential'
git -C "$tmp/tap" config user.name "Claudodex Maintainers"
git -C "$tmp/tap" config user.email "41898282+github-actions[bot]@users.noreply.github.com"
python3 - "$tmp/tap/Formula/claudodex.rb" "$version" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
if path.exists():
    versions = re.findall(r'^  version "([0-9]+\.[0-9]+\.[0-9]+)"$', path.read_text(), re.MULTILINE)
    if len(versions) != 1:
        raise SystemExit("Cannot compare the existing tap formula version safely")
    if tuple(map(int, versions[0].split("."))) > tuple(map(int, sys.argv[2].split("."))):
        raise SystemExit("Refusing to downgrade the existing tap formula")
PY
mkdir -p "$tmp/tap/Formula"
cp "$formula" "$tmp/tap/Formula/claudodex.rb"
git -C "$tmp/tap" add -- Formula/claudodex.rb
if git -C "$tmp/tap" diff --cached --quiet; then
  exit 0
fi
git -C "$tmp/tap" commit --quiet -m "Update Claudodex to $version" -- Formula/claudodex.rb
for attempt in 1 2 3; do
  # A concurrent tap commit is retained. A conflict on this formula needs review.
  git -C "$tmp/tap" fetch --quiet origin
  git -C "$tmp/tap" rebase '@{upstream}'
  if git -C "$tmp/tap" push origin HEAD; then
    exit 0
  fi
done
echo "Tap moved during publication; retry without reverting other commits." >&2
exit 1
