#!/usr/bin/env python3
# SPDX-License-Identifier: 0BSD
"""Smoke-test every built MCP server for initialize, tools/list, and read-only mode."""

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


def tool_names(tools_msg):
    if not tools_msg or "error" in tools_msg:
        return []
    result = tools_msg.get("result", {})
    tlist = result.get("tools", [])
    return [t.get("name") for t in tlist]


def inspect(repo, s, readonly=False):
    path = os.path.join(repo, s["command"])
    if not os.path.exists(path):
        return "missing", 0, f"binary not found: {path}", []
    env = os.environ.copy()
    for k in s.get("env", []):
        if k not in env:
            env[k] = ""
    if "MCP_REPO_ROOT" not in env:
        env["MCP_REPO_ROOT"] = repo
    if not env.get("GATEWAY_CONFIG"):
        env["GATEWAY_CONFIG"] = os.path.join(repo, "dist", "quad4-mcp.json")
    if readonly:
        env["MCP_READ_ONLY"] = "1"
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
        return "fail", 0, "no initialize response", []
    if "error" in init:
        proc.terminate()
        return "fail", 0, f"initialize error: {init['error'].get('message', 'unknown')}", []
    caps = init.get("result", {}).get("capabilities", {})
    tools = send(proc.stdin, proc.stdout, {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {}
    })
    if not tools:
        proc.terminate()
        return "fail", 0, "no tools/list response", []
    if "error" in tools:
        proc.terminate()
        return "fail", 0, f"tools/list error: {tools['error'].get('message', 'unknown')}", []
    result = tools.get("result", {})
    tlist = result.get("tools")
    if not isinstance(tlist, list):
        proc.terminate()
        return "fail", 0, "tools/list result.tools is not a list", []
    count = len(tlist)
    names = tool_names(tools)
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
        return "warn", 0, "zero tools advertised", names
    return "ok", count, "", names


def call_hidden_write(repo, s, name):
    env = os.environ.copy()
    for k in s.get("env", []):
        if k not in env:
            env[k] = ""
    if "MCP_REPO_ROOT" not in env:
        env["MCP_REPO_ROOT"] = repo
    if not env.get("GATEWAY_CONFIG"):
        env["GATEWAY_CONFIG"] = os.path.join(repo, "dist", "quad4-mcp.json")
    env["MCP_READ_ONLY"] = "1"
    proc = subprocess.Popen(
        [os.path.join(repo, s["command"])],
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
    if not init or "error" in init:
        proc.terminate()
        return "fail", "no initialize for write call"
    call = send(proc.stdin, proc.stdout, {
        "jsonrpc": "2.0",
        "id": 3,
        "method": "tools/call",
        "params": {"name": name, "arguments": {}}
    })
    proc.stdin.close()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.terminate()
    if not call:
        return "fail", "no response to blocked tools/call"
    if "error" in call:
        return "ok", call["error"].get("message", "unknown")
    # Some servers may return isError true; that also counts as blocked.
    result = call.get("result", {})
    if result.get("isError") or result.get("content", [{}])[0].get("text", "").startswith("error:"):
        return "ok", "blocked"
    return "fail", f"write tool {name} was allowed in read-only mode"


def main():
    repo = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    manifest = load_manifest(os.path.join(repo, "mcp-servers.json"))
    failed = 0
    for s in manifest["servers"]:
        status, count, note, names = inspect(repo, s, readonly=False)
        label = {"ok": "OK", "warn": "WARN", "fail": "FAIL", "missing": "MISS"}[status]
        if status == "ok":
            print(f"{label}: {s['name']:18} {count} tools")
        else:
            print(f"{label}: {s['name']:18} {note}")
        if status in ("fail", "missing"):
            failed += 1
            continue
        _, ro_count, ro_note, ro_names = inspect(repo, s, readonly=True)
        ro_status = "ok" if ro_count is not None else "fail"
        if ro_status == "fail":
            print(f"    RO: FAIL: {s['name']:18} {ro_note}")
            failed += 1
            continue
        hidden = set(names) - set(ro_names)
        if hidden:
            print(f"    RO: {s['name']:18} {len(hidden)} write tools hidden")
            test = sorted(hidden)[0]
            call_status, call_msg = call_hidden_write(repo, s, test)
            if call_status != "ok":
                print(f"    RO: FAIL: {s['name']:18} {call_msg}")
                failed += 1
            else:
                print(f"    RO: {s['name']:18} blocked {test}")
        else:
            print(f"    RO: {s['name']:18} no write tools to hide")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
