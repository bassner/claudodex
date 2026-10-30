#!/usr/bin/env bash
# Do not promote an older release when retrying an old tag workflow.
set -euo pipefail
version=${1:?release version required}
latest=$(gh release view --repo bassner/claudodex --json tagName --jq .tagName)
python3 - "$version" "$latest" <<'PY'
import re
import sys

def stable(value):
    if not re.fullmatch(r"v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", value):
        raise SystemExit("Cannot compare the current latest release safely")
    return tuple(map(int, value.removeprefix("v").split(".")))

if stable(sys.argv[1]) < stable(sys.argv[2]):
    raise SystemExit("Refusing to publish an older release than the current latest")
PY
