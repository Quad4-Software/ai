# Blackhole

Blackholes let a node drop announces and traffic from identified spammers.
Distributed blackhole lists let communities share abuse signals.

## CLI

- `rnpath -B <hash>`: blackhole an identity
- `rnpath -U <hash>`: unblackhole an identity
- `rnpath -b`: list blackholed identities
- `rnstatus -b`: watch blocked IPs and interfaces

## Operation

- Blackholes are for abuse and spam, not for routine filtering
- Combine with `null_ident` options to reject unidentified or null-identity peers
- Monitor and manage through `rnstatus` and `rnpath`
- Distributed lists can be published and consumed by transport nodes

## Tool reference

Tool-building rules are in [references/tools.md](references/tools.md). This section is optional if the related MCP server is installed.