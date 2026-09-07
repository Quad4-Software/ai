#!/usr/bin/env python3
# SPDX-License-Identifier: 0BSD
"""Generate MCP client, gateway, and registry configs from mcp-servers.json."""

import argparse
import json
import os
import sys


def load_manifest(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def client_config(servers, repo):
    gateway = next(s for s in servers if s["name"] == "gateway")
    return {
        "mcpServers": {
            "gateway": {
                "command": os.path.join(repo, gateway["command"]),
                "args": [],
                "env": {
                    "GATEWAY_CONFIG": os.path.expanduser("~/.config/mcp/quad4-mcp.json")
                }
            }
        }
    }


def gateway_config(servers, repo):
    mcp = {}
    for s in servers:
        if s["name"] == "gateway":
            continue
        env = {k: "" for k in s.get("env", [])}
        mcp[s["name"]] = {
            "command": os.path.join(repo, s["command"]),
            "args": [],
            "env": env
        }
    return {"mcpServers": mcp}


def server_json(s):
    return {
        "$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
        "name": f"io.github.Quad4-Software.ai/{s['module']}",
        "title": s["name"].replace("-", " ").title(),
        "description": s["description"],
        "version": s["version"],
        "packages": [
            {
                "registryType": "github",
                "identifier": "github.com/Quad4-Software/ai",
                "version": s["version"],
                "transport": {"type": "stdio"}
            }
        ]
    }


def write_json(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)
        f.write("\n")


def main():
    parser = argparse.ArgumentParser(description="Generate MCP configs")
    parser.add_argument(
        "--manifest",
        default="mcp-servers.json",
        help="path to mcp-servers.json"
    )
    parser.add_argument(
        "--repo",
        default=os.getcwd(),
        help="absolute path to the repo root"
    )
    parser.add_argument(
        "--out",
        default="dist",
        help="output directory for generated files"
    )
    parser.add_argument(
        "--print",
        action="store_true",
        help="print the client and gateway JSON to stdout"
    )
    parser.add_argument(
        "--server-json",
        action="store_true",
        help="also generate registry server.json files"
    )
    args = parser.parse_args()

    repo = os.path.abspath(args.repo)
    manifest = load_manifest(args.manifest)
    servers = manifest["servers"]

    client = client_config(servers, repo)
    gateway = gateway_config(servers, repo)

    if args.print:
        print("=== Client mcp.json (one gateway entry) ===")
        print(json.dumps(client, indent=2))
        print("\n=== Gateway quad4-mcp.json (all servers) ===")
        print(json.dumps(gateway, indent=2))
        return

    out = os.path.abspath(args.out)
    write_json(os.path.join(out, "mcp.json"), client)
    write_json(os.path.join(out, "quad4-mcp.json"), gateway)
    print(f"Client config: {out}/mcp.json")
    print(f"Gateway config: {out}/quad4-mcp.json")
    print("Add the gateway entry to your MCP client config and copy")
    print("quad4-mcp.json to ~/.config/mcp/quad4-mcp.json.")

    if args.server_json:
        regdir = os.path.join(out, "server-json")
        for s in servers:
            if s["name"] == "gateway":
                continue
            write_json(os.path.join(regdir, f"{s['module']}.json"), server_json(s))
        print(f"Registry server.json files: {regdir}/")


if __name__ == "__main__":
    main()
