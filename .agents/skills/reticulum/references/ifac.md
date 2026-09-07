# Interface Access Codes

IFAC authenticate or isolate virtual networks by requiring an access code or
signature on every packet over an interface. Enable IFAC on any interface that
untrusted peers can reach.

## When to use

- Any interface over the Internet, public WiFi, or shared radio should use IFAC
- Without IFAC, anyone with access to the carrier can inject packets
- IFAC can also create separate virtual networks on the same physical medium

## Configuration

The `ifac_*` options vary by interface type. Typical options are:

- `ifac_size`: size of the IFAC field in bits; use a strong size for your threat
  model
- `ifac_key`: key or passphrase used to derive the IFAC
- `ifac_netname`: a network name for an authenticated IFAC

## Packet layout

A packet with IFAC has the IFAC bytes appended after the 2-byte header:

```
[HEADER 2 bytes] [IFAC N bytes] [ADDRESSES 16/32 bytes] [CONTEXT 1 byte] [DATA]
```

The IFAC flag in the first header byte is set to 1 when an IFAC is present.

## Security notes

- Always enable IFAC on public or untrusted carriers
- Use a strong passphrase or authentication
- Never leak the IFAC key or passphrase in tool output or commit logs
- IFAC does not replace end-to-end encryption; it adds a network-access control
  layer

## Tool reference

Tool-building rules are in [references/tools.md](references/tools.md). This section is optional if the related MCP server is installed.