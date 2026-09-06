#!/usr/bin/env bash
set -euo pipefail

OWNER="Quad4-Software"
REPO="ai"
BRANCH="master"

cat <<EOF
[![ci](https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/ci.yml?branch=${BRANCH}&logo=github&label=ci)](https://github.com/${OWNER}/${REPO}/actions/workflows/ci.yml)
[![gosec](https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/gosec.yml?branch=${BRANCH}&logo=github&label=gosec)](https://github.com/${OWNER}/${REPO}/actions/workflows/gosec.yml)
[![race](https://img.shields.io/github/actions/workflow/status/${OWNER}/${REPO}/race.yml?branch=${BRANCH}&logo=github&label=race)](https://github.com/${OWNER}/${REPO}/actions/workflows/race.yml)
[![release](https://img.shields.io/github/v/release/${OWNER}/${REPO}?logo=github&label=release)](https://github.com/${OWNER}/${REPO}/releases/latest)
[![go version](https://img.shields.io/github/go-mod/go-version/${OWNER}/${REPO}?filename=agents-mcp/go.mod&logo=go&label=go%20version)](https://github.com/${OWNER}/${REPO}/blob/${BRANCH}/agents-mcp/go.mod)
[![license](https://img.shields.io/github/license/${OWNER}/${REPO}?logo=opensourceinitiative&label=license)](https://github.com/${OWNER}/${REPO}/blob/${BRANCH}/LICENSE)
EOF
