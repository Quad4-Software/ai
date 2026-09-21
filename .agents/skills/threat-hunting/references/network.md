# Detection content and network hunting

## Detection content

- **Sigma**: rules at github.com/SigmaHQ/sigma, spec v2.1.0.
 sigmac is archived - use pySigma 1.5.x + sigma-cli with backends
 (Splunk, KQL, ES) and pipelines (pysigma-pipeline-sysmon).
 Hayabusa consumes Sigma v2 correlation rules natively.
- **YARA**: classic YARA 4.5.x is maintenance mode. YARA-X 1.20.x
 (Rust, `yr` CLI) is the active line, mostly rule-compatible. YARA
 Forge ruleset (Nextron-curated) is the recommended rulebase.
 LOKI is semi-maintained. THOR Lite is the active free scanner.
- **LOLBAS** (Windows binaries) and **GTFObins** (Linux binaries) are
 the reference for living-off-the-land abuse: hunt unexpected
 parents, network connections, or paths for certutil, mshta,
 regsvr32, rundll32, bitsadmin, msbuild, wmic, find, awk, vim, nmap.

## Fleet management

- Velociraptor 0.77.x: server + agent, VQL hunts, artifact exchange.
- osquery 5.23.x + Fleet 4.91.x (free tier: scheduled and distributed
 queries, packs) or Kolide (SaaS).
- Defender XDR advanced hunting (KQL) is the reference schema even if
 not free. Many Sigma rules convert to KQL.

## Network hunting

- **Zeek 9.0.0 LTS** (Aug 2026): conn.log, dns.log, http.log, ssl.log
 feed most network hunts. Zeek 8.2.x is EOL.
- **Suricata 8.0.7+**: patch immediately, 8.0.7 fixes CVE-2026-94084
 (9.4 UAF). Suricata 7 is EOL at 7.0.17.
- **RITA 5.1.x** (Active Countermeasures, rewritten on ClickHouse):
 imports Zeek logs, beacon scoring, long connections, DNS tunneling,
 threat-intel checks, prevalence/first-seen.
- **Arkime 6.7.x**: full packet capture + session search.
- **Malcolm 26.08.x** (cisagov): all-in-one bundle of Zeek + Suricata
 + Arkime + OpenSearch. Easiest deployment for a full network
 hunting stack.
- Newly-registered-domain hunting: join Sysmon EID 22 or Zeek dns.log
 against an NRD feed (whoisds, openintel). RITA automates part of
 the threat-intel pivot.

## Version table (Sept 2026)

| Tool | Version |
| --- | --- |
| Sysmon | 15.22.0 |
| KAPE | 1.3.x (KapeFiles repo active) |
| Chainsaw | 2.16.4 |
| Hayabusa | 4.1.0 |
| Velociraptor | 0.77.2 |
| osquery | 5.23.1 |
| Volatility 3 | 2.28.2 |
| Falco | 0.44.1 |
| Tetragon | 1.7.1 |
| Tracee | 0.24.1 |
| Sysdig | 0.41.4 |
| Zeek | 9.0.0 LTS |
| Suricata | 8.0.7 |
| pySigma | 1.5.0 |
| YARA-X | 1.20.0 |
| Fleet | 4.91.1 |
| Arkime | 6.7.0 |
| Malcolm | 26.08.0 |
| RITA | 5.1.2 |
| MITRE ATT&CK | v19.2 |
