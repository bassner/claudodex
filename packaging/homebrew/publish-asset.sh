#!/usr/bin/env bash
# A retry may reuse an identical public asset, but must not replace its bytes.
set -euo pipefail
tag=${1:?release tag required}
shift
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || exit 1
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
for path in "$@"; do
  name=$(basename "$path")
  [[ "$name" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$ ]] || exit 1
  gh release view "$tag" --repo bassner/claudodex --json assets > "$tmp/assets.json"
  if jq -e --arg name "$name" '.assets[] | select(.name == $name)' "$tmp/assets.json" > /dev/null; then
    gh release download "$tag" --repo bassner/claudodex --pattern "$name" --dir "$tmp"
    cmp "$path" "$tmp/$name" || { echo "Existing release asset differs: $name" >&2; exit 1; }
    rm "$tmp/$name"
  else
    gh release upload "$tag" "$path" --repo bassner/claudodex
  fi
done
