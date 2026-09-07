#!/usr/bin/env python3
# SPDX-License-Identifier: 0BSD
"""Smoke-test every built MCP server for initialize and tools/list."""

import json
import os
import subprocess
import sys
import time


def load_manifest(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def send(stdin, stdout, req):
    line = json.dumps(req).encode("utf-8") + b"\n"
    stdin.write(line)
    stdin.flush()
    timeout = time.time() + 10
    while time.time() < timeout:
        try:
            out = stdout.readline()
        except Exception:
            break
        if not out:
            continue
        try:
            msg = json.loads(out.decode("utf-8"))
        except json.JSONDecodeError:
            continue
        if msg.get("id") == req["id"]:
            return msg
    return None


def inspect(repo, s):
    path = os.path.join(repo, s["command"])
    if not os.path.exists(path):
        return "missing", 0, f"binary not found: {path}"
    env = os.environ.copy()
    for k in s.get("env", []):
        if k not in env:
            env[k] = ""
    if "MCP_REPO_ROOT" not in env:
        env["MCP_REPO_ROOT"] = repo
    env.setdefault("GATEWAY_CONFIG", os.path.join(repo, "dist", "quad4-mcp.json"))
    proc = subprocess.Popen(
        [path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        env=env,
        cwd=repo
    )
    init = send(proc.stdin, proc.stdout, {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-11-25",
            "capabilities": {},
            "clientInfo": {"name": "mcp-inspector", "version": "0.1.0"}
        }
    })
    if not init:
        proc.terminate()
        return "fail", 0, "no initialize response"
    if "error" in init:
        proc.terminate()
        return "fail", 0, f"initialize error: {init['error'].get('message', 'unknown')}"
    caps = init.get("result", {}).get("capabilities", {})
    tools = send(proc.stdin, proc.stdout, {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {}
    })
    count = 0
    if not tools:
        proc.terminate()
        return "fail", 0, "no tools/list response"
    if "error" in tools:
        proc.terminate()
        return "fail", 0, f"tools/list error: {tools['error'].get('message', 'unknown')}"
    result = tools.get("result", {})
    tlist = result.get("tools")
    if not isinstance(tlist, list):
        proc.terminate()
        return "fail", 0, "tools/list result.tools is not a list"
    count = len(tlist)
    # optional prompts and resources checks
    for cap, method in [("prompts", "prompts/list"), ("resources", "resources/list")]:
        if cap in caps:
            send(proc.stdin, proc.stdout, {
                "jsonrpc": "2.0",
                "id": cap,
                "method": method,
                "params": {}
            })
    proc.stdin.close()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.terminate()
    if count == 0:
        return "warn", 0, "zero tools advertised"
    return "ok", count, ""


def main():
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    manifest = load_manifest(os.path.join(repo, "mcp-servers.json"))
    failed = 0
    for s in manifest["servers"]:
        status, count, note = inspect(repo, s)
        label = {"ok": "OK", "warn": "WARN", "fail": "FAIL", "missing": "MISS"}[status]
        if status == "ok":
            print(f"{label}: {s['name']:18} {count} tools")
        else:
            print(f"{label}: {s['name']:18} {note}")
        if status in ("fail", "missing"):
            failed += 1
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
