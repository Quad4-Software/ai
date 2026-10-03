# Interface discovery

## AutoInterface discovery

- AutoInterface finds peers over UDP without any IP infrastructure
- Default ports are 29716 and 42671
- Use `discovery_scope` to limit or expand reach to `link`, `admin`, `site`,
  `organisation` or `global`
- The manual's scoped discovery example uses `discovery_port 48555` and
  `data_port 49555`
- Set `group_id` to isolate multiple networks on the same LAN

## Discovery announces and auto-connect

- Discoverable interfaces announce discovery data (implementation, version,
  port, transport flag) that other nodes can use to connect
- Since 1.5.5, auto-connect to discovered interfaces also works on Windows
  and macOS. The `[reticulum]` options `autoconnect_discovered_interfaces`,
  `autoconnect_unverified_implementations`, `autoconnect_interface_mode`,
  `autoconnect_interface_gravity` and `autoconnect_announces_to_internal`
  control the behaviour. `interface_discovery_sources` pins trusted
  discovery sources by identity hash
- 1.5.6 requires a discovered interface to announce itself as
  transport-enabled before it is used for auto-connect, and enables a
  static transport identity when discoverable interfaces are configured
  on a non-transport instance

## LXMF contacts

- Add `discovery_lxmf_address` to any interface to publish an LXMF contact
  address in its discovery data
- This lets operators coordinate interconnections and local network growth
- You can view discovered interface data with `rnstatus -D`
- User-facing clients should support reading and displaying this field

## CLI

- `rnstatus -d`: list discovered interfaces
- `rnstatus -D`: list discovered interfaces with full details
- Use discovery output to inspect `discovery_lxmf_address` fields

## Tool reference

Tool-building rules are in [references/tools.md](tools.md). This section is optional if the related MCP server is installed.