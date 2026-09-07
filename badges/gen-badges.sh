#!/usr/bin/env bash
set -euo pipefail

# Generate custom themed SVG badges for the repo README.
# Usage: ./badges/gen-badges.sh

OUTDIR="$(cd "$(dirname "$0")" && pwd)"

badge() {
  local file=$1 left=$2 right=$3 color=$4 leftw=$5 rightw=$6
  local total=$((leftw + rightw))
  cat > "$OUTDIR/$file.svg" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$total" height="22" role="img" aria-label="$left: $right">
  <title>$left: $right</title>
  <defs>
    <linearGradient id="g$file" x2="0" y2="100%">
      <stop offset="0" stop-color="#fff" stop-opacity=".15"/>
      <stop offset="1" stop-color="#000" stop-opacity=".1"/>
    </linearGradient>
    <filter id="s$file" x="0" y="0" width="100%" height="100%">
      <feDropShadow dx="0" dy="1" stdDeviation="0.5" flood-color="#000" flood-opacity=".2"/>
    </filter>
  </defs>
  <clipPath id="c$file">
    <rect width="$total" height="22" rx="6" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#c$file)">
    <rect width="$leftw" height="22" fill="#2d2d2d"/>
    <rect x="$leftw" width="$rightw" height="22" fill="$color"/>
    <rect width="$total" height="22" fill="url(#g$file)"/>
  </g>
  <rect x="$leftw" y="0" width="1" height="22" fill="#000" fill-opacity=".15"/>
  <g fill="#fff" text-anchor="middle" font-family="system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif" font-size="11" font-weight="600" filter="url(#s$file)">
    <text x="$((leftw / 2))" y="16">$left</text>
    <text x="$((leftw + rightw / 2))" y="16">$right</text>
  </g>
</svg>
EOF
}

# label, status, color, left width, right width
badge ci          "CI"         "passing"  "#4c1"    28 58
badge gosec       "gosec"      "passing"  "#4c1"    38 58
badge race        "race"       "passing"  "#37d"    36 58
badge mcp-inspector "inspector"  "passing"  "#9c4"    54 58
badge release     "release"    "v0.2.0"   "#a35"    42 48
badge go          "go"         "1.27"     "#00ADD8" 26 40
badge license     "license"    "0BSD"     "#999"    38 40

echo "wrote badges to $OUTDIR"
