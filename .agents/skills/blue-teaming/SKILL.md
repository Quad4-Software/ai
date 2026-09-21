---
name: blue-teaming
description: >
  This skill covers blue teaming: the whole defensive security
  function as of September 2026. Use it for SOC stack selection
  (Wazuh, Security Onion, Elastic, Splunk free limits), alert triage,
  detection engineering as code (Sigma, Atomic Red Team, Caldera),
  incident response (NIST 800-61r3, PICERL), DFIR triage tooling
  (KAPE, EZ Tools, Hayabusa, Volatility, MemProcFS), deception
  (Canarytokens, Cowrie, T-Pot), tabletop exercises, purple teaming,
  and SOC metrics (MTTD/MTTR). Threat hunting itself lives in the
  threat-hunting skill.
metadata:
  sources:
    - https://attack.mitre.org/
    - https://github.com/SigmaHQ/sigma
    - https://www.nist.gov/pubs/sp/800/61/r3/final
---

## When to use this skill

- Building or assessing a SOC or monitoring stack.
- Writing, testing, or tuning detections.
- Standing up incident response, DFIR triage, or deception.
- Running tabletops or purple-team exercises, tracking SOC metrics.

Hunting for adversaries proactively is the threat-hunting skill. This
one covers the operational core: alerting, triage, IR, detection
engineering, deception.

## Free SOC stack

| Layer | Free/OSS picks |
|---|---|
| SIEM/XDR | Wazuh (GPLv2, no paywall, own agent), Security Onion 3.x (full distro: Elastic + Suricata + Zeek + SOC UI, v2.4 EOL Oct 2026), Elastic Basic, OpenSearch |
| EDR/endpoint | Wazuh agent (FIM, SCA, active response), Velociraptor, osquery+Fleet, LimaCharlie free tier, Sysmon+Winlogbeat as poor man's EDR |
| Network | Suricata, Zeek, Arkime, Malcolm |
| Case mgmt | TheHive + Cortex, DFIR-IRIS |
| SOAR | Shuffle (AGPLv3), Tines Community Edition (free cloud), n8n for simple playbooks |
| Threat intel | MISP, OpenCTI, free feeds (AbuseIPDB, URLhaus, MalwareBazaar, ThreatFox, Feodo, CISA KEV) |

Splunk Free is a trap for real use: 500 MB/day ingest and search locks
out after license violations.

## Alert triage

Queue, validate (TP/FP/BTP), enrich (GreyNoise, VT, asset context),
scope (hosts, timeline), escalate or close with a disposition code,
then tune or document. Never close on the alert name alone. Track
dispositions: every false-positive pattern is a tuning opportunity.

## Detection engineering as code

1. Prioritize ATT&CK techniques against your threat model and gap-map
   coverage in ATT&CK Navigator.
2. Write portable first: Sigma for logs (SigmaHQ 3,000+ rules,
   pySigma/sigma-cli converts to your SIEM), YARA for files/memory,
   Suricata rules for network.
3. Rules live in Git with PR review and metadata (author, ATT&CK
   tags, FP notes, logsource).
4. Test with Atomic Red Team or MITRE Caldera before trusting a rule.
   SigmaHQ release tiers: start with `core`, expand to `core+` or
   `emerging-threats` as tuning matures.
5. Deploy via CI converting Sigma to target queries.
6. Measure FP rate, MTTD per rule, coverage percent. Purple-team
   periodically.

ATT&CK v19 (Apr 2026) split Defense Evasion into Stealth (TA0005) and
Defense Impairment (TA0112). Detection Strategies objects replaced
free-text detection fields in v18. D3FEND is the defensive
countermeasure ontology, Engage covers deception.

## Incident response

NIST SP 800-61r3 (final April 2025) supersedes r2 and folds IR into
CSF 2.0 functions. SANS PICERL remains the practical loop:
Preparation, Identification, Containment, Eradication, Recovery,
Lessons learned.

DFIR quick-triage on Windows: KAPE targets + EZ Tools modules
(MFTECmd, EvtxECmd, RECmd, AmcacheParser, PECmd, LECmd), Hayabusa for
fast Sigma-over-EVTX, MemProcFS to mount memory as a filesystem,
Volatility 3 for deep memory analysis, Plaso/Timesketch for timelines,
CyberChef for decode/deobfuscation. See threat-hunting for the full
artifact list.

## Deception

- Canarytokens (free hosted): fake AWS keys, docs, URLs, DNS names
  that alert on touch. Cheapest high-signal detection that exists.
- Cowrie for SSH/Telnet honeypots, T-Pot for the ~30-honeypot bundle,
  Endlessh tarpits, Dionaea/Conpot for malware/ICS.
- Plant honey creds and canary files in AD and repos with alerting.

## Exercises and metrics

- Tabletops: CISA CTEP kits are free, Backdoors & Breaches for team
  training. Quarterly walk-throughs, annually minimum.
- Purple team: pick technique, emulate with Atomic/Caldera, verify
  detection, tune, retest. Track in Vectr.io or Navigator layers.
- Metrics that matter: MTTD, MTTR, MTTC, dwell time, alert fidelity,
  ATT&CK coverage. Avoid vanity counts, trend over snapshot.
