# Interface discovery

## AutoInterface discovery

- AutoInterface finds peers over UDP without any IP infrastructure
- Default ports are 29716 and 42671
- Use `discovery_scope` to limit or expand reach to `link`, `admin`, `site`,
  `organisation` or `global`
- When a discovery scope is set, the defaults become discovery_port 48555 and
  data_port 49555
- Set `group_id` to isolate multiple networks on the same LAN

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