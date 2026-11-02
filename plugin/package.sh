#!/usr/bin/env bash
# Builds the Claude Code plugin zip and its marketplace.json for a release.
# Usage: plugin/package.sh <version> <windows exe> <out dir> <base url of the release assets>
# Testers install from the marketplace.json over https, so neither git nor Go is needed.
set -euo pipefail

version="$1"
exe="$2"
out="$3"
base_url="$4"

here="$(cd "$(dirname "$0")" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

mkdir -p "$out" "$work/simconnect/server"
out="$(cd "$out" && pwd)"
exe_abs="$(cd "$(dirname "$exe")" && pwd)/$(basename "$exe")"

cp -r "$here/.claude-plugin" "$here/.mcp.json" "$here/skills" "$here/output-styles" "$work/simconnect/"
cp "$here/../LICENSE" "$work/simconnect/"
cp "$exe_abs" "$work/simconnect/server/simconnect-mcp.exe"
sed -i "s/0\.0\.0-dev/$version/" "$work/simconnect/.claude-plugin/plugin.json"

zip_name="simconnect-plugin_${version}.zip"
rm -f "$out/$zip_name"
if command -v zip >/dev/null 2>&1; then
  (cd "$work" && zip -qr "$out/$zip_name" simconnect)
else
  (cd "$work" && /c/Windows/System32/tar.exe -a -cf "$zip_name" simconnect && mv "$zip_name" "$out/")
fi

sha="$(sha256sum "$out/$zip_name" | cut -d' ' -f1)"

cat > "$out/marketplace.json" <<EOF
{
  "name": "simconnect",
  "owner": { "name": "mrlm-net" },
  "plugins": [
    {
      "name": "simconnect",
      "description": "Talk to Microsoft Flight Simulator from Claude: read the live sim, check it is connected, and write up bug reports.",
      "version": "$version",
      "source": {
        "source": "archive",
        "url": "$base_url/$zip_name",
        "sha256": "$sha"
      }
    }
  ]
}
EOF

echo "$out/$zip_name ($sha)"
