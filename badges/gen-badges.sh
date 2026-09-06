#!/usr/bin/env bash
set -euo pipefail

OWNER="Quad4-Software"
REPO="ai"
BRANCH="master"

cat <<EOF
<a href="https://github.com/${OWNER}/${REPO}/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/ci.yml?branch=${BRANCH}&logo=github&label=ci" alt="ci"></a>
<a href="https://github.com/${OWNER}/${REPO}/actions/workflows/gosec.yml"><img src="https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/gosec.yml?branch=${BRANCH}&logo=github&label=gosec" alt="gosec"></a>
<a href="https://github.com/${OWNER}/${REPO}/actions/workflows/race.yml"><img src="https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/race.yml?branch=${BRANCH}&logo=github&label=race" alt="race"></a>
<a href="https://github.com/${OWNER}/${REPO}/releases/latest"><img src="https://img.shields.io/github/v/release/${OWNER}/${REPO}?logo=github&label=release" alt="release"></a>
<a href="https://github.com/${OWNER}/${REPO}/blob/${BRANCH}/agents-mcp/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/${OWNER}/${REPO}?filename=agents-mcp/go.mod&logo=go&label=go%20version" alt="go version"></a>
<a href="https://github.com/${OWNER}/${REPO}/blob/${BRANCH}/LICENSE"><img src="https://img.shields.io/github/license/${OWNER}/${REPO}?logo=opensourceinitiative&label=license" alt="license"></a>
EOF
