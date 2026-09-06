#!/usr/bin/env bash
set -euo pipefail

# Generate custom themed SVG badges for the repo README.
# Usage: ./badges/gen-badges.sh

OUTDIR="$(cd "$(dirname "$0")" && pwd)"

badge() {
  local file=$1 left=$2 right=$3 color=$4 leftw=$5 rightw=$6
  local total=$((leftw + rightw))
  cat > "$OUTDIR/$file.svg" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$total" height="20" role="img" aria-label="$left: $right">
  <title>$left: $right</title>
  <linearGradient id="g$file" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="c$file">
    <rect width="$total" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#c$file)">
    <rect width="$leftw" height="20" fill="#555"/>
    <rect x="$leftw" width="$rightw" height="20" fill="$color"/>
    <rect width="$total" height="20" fill="url(#g$file)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="$((leftw / 2))" y="14">$left</text>
    <text x="$((leftw + rightw / 2))" y="14">$right</text>
  </g>
</svg>
EOF
}

# label, status, color, left width, right width
badge ci      "CI"      "passing"  "#4c1"   28 58
badge gosec   "gosec"   "passing"  "#4c1"   38 58
badge race    "race"    "passing"  "#37d"   36 58
badge release "release" "v0.1.7"   "#a35"   42 48
badge go      "go"      "1.27"     "#00ADD8" 26 40
badge license "license" "0BSD"     "#999"   38 40

echo "wrote badges to $OUTDIR"
