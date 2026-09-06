#!/usr/bin/env bash
set -euo pipefail

CHECKSUMS="dist/checksums.txt"
CHANGELOG="dist/CHANGELOG.md"
OUT="notes.md"

if [[ ! -f "$CHECKSUMS" ]]; then
  echo "error: $CHECKSUMS not found" >&2
  exit 1
fi

{
  if [[ -f "$CHANGELOG" ]]; then
    cat "$CHANGELOG"
    echo ""
  fi
  echo "## Artifact checksums"
  echo ""
  echo "| Artifact | SHA-256 |"
  echo "| --- | --- |"
  while read -r hash name; do
    [[ -z "$hash" ]] && continue
    echo "| $name | \`$hash\` |"
  done < "$CHECKSUMS"
} > "$OUT"

echo "wrote $OUT"
