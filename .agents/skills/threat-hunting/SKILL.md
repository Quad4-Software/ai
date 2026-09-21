---
name: threat-hunting
description: >
  This skill covers threat hunting on Linux and Windows endpoints with
  free and open-source tooling as of September 2026: hunting
  methodology (hypothesis-driven, Pyramid of Pain, ATT&CK v19, Sqrrl
  maturity, PEAK, TaHiTI), Sysmon and Windows event IDs, Sysinternals,
  KAPE, Chainsaw, Hayabusa, Velociraptor, osquery, Volatility 3,
  Sigma/pySigma, YARA/YARA-X, auditd, Falco, Tetragon, Tracee, Linux
  persistence locations, eBPF rootkits, and network hunting with Zeek,
  Suricata, RITA, Arkime. Use it when hunting for persistence, C2,
  lateral movement, or triaging a suspect host.
---

## When to use this skill

- You are hunting for persistence, C2, or lateral movement on a host
  or fleet.
- You are picking telemetry tooling (Sysmon, auditd, Falco, osquery)
  or triage tooling (KAPE, Chainsaw, Hayabusa, Velociraptor).
- You need the right event IDs, log paths, or persistence locations.
- You are writing hunt hypotheses or mapping findings to ATT&CK.

## How to use

1. Read this file for methodology and tool selection.
2. Load [references/windows.md](references/windows.md) for Windows
   telemetry, artifacts, and persistence keys.
   Load [references/linux.md](references/linux.md) for auditd, eBPF
   tools, logs, and Linux persistence.
   Load [references/network.md](references/network.md) for Sigma,
   YARA, fleet tools, and network hunting.
   Load [references/recipes.md](references/recipes.md) for concrete
   hunts with commands.

## Examples

- "Hunt for unsigned binaries executing from user-writable dirs."
- "What persistence locations should I check on this Linux box?"
- "Set up Sysmon and start sweeping EVTX with Sigma rules."

# Threat hunting

## Methodology

- Hypothesis-driven hunting is the core loop: form a testable
  statement ("an attacker persists via WMI subscriptions on file
  servers"), scope data, hunt, document, and feed results back into
  automated detection. A hunt that finds nothing still produces a
  detection or a coverage-gap report.
- IOC sweeping (hashes, IPs, domains) is cheap and perishable. TTP
  hunting mapped to ATT&CK techniques is durable. Use IOCs as entry
  points, then pivot to the technique. The Pyramid of Pain ranks
  indicators: hashes, IPs, domains, artifacts, tools, TTPs.
- **ATT&CK v19.2 (Aug 2026)** is current. Note v19 split Defense
  Evasion into two tactics: Stealth and Defense Impairment. Re-base
  coverage matrices on v19.
- Maturity model (Sqrrl HMM0-HMM4): HM1 is IOC sweeps, HM2 runs others'
  procedures, HM3 writes repeatable procedures, HM4 automates
  successful hunts into detections. The goal of a hunt program is
  always the last step.
- Frameworks: PEAK (Prepare, Execute, Act, Knowledge) for hunt
  lifecycle and deliverables. TaHiTI for threat-intel-integrated
  hunts with a documentation template.

## Tool selection

- **Windows telemetry**: Sysmon 15.22.x (built-in optional Windows 11
  feature since Feb 2026, also classic installer. Runs as PPL) with
  sysmon-modular or SwiftOnSecurity config. Windows event log for
  4688/4624/4648/4698/7045/1102/4104.
- **Windows triage**: KAPE + EZ Tools for artifact collection,
  Chainsaw (fast) or Hayabusa (fullest Sigma coverage incl. v2
  correlation) for EVTX hunting, Velociraptor for live/forensic
  fleet hunts, osquery for scheduled SQL hunts.
- **Linux telemetry**: auditd with the Neo23x0 ruleset for execve and
  file watches. Falco, Tetragon, or Tracee for eBPF runtime
  detection. Osquery for state. journald/auth.log for auth,
  psacct for process accounting (bash_history is unreliable).
- **Memory**: Volatility 3 (2.28.x) - Windows symbols auto-fetch,
  Linux needs dwarf2json packs. Capture with LiME or AVML.
- **Detection content**: Sigma (sigmac is dead, use pySigma +
  sigma-cli), YARA-X (classic YARA 4.5.x is maintenance mode), LOLBAS
  for Windows binary abuse, GTFObins for Linux.
- **Network**: Zeek 9.0 LTS logs, Suricata 8.0.7+ (8.0.7 fixes a 9.4
  UAF - patch immediately), RITA 5.x for beaconing on Zeek logs,
  Arkime or Malcolm for full capture.
- **Dated tools**: rkhunter is stale (2018), chkrootkit gets rare
  releases, LOKI is semi-maintained (THOR Lite is the active free
  scanner). Treat them as secondary signals only.

## Golden rules

- Hunt for attacker tooling execution (linPEAS, PsExec) rather than
  running it yourself.
- Missing telemetry is a signal: attackers blind ETW/auditd. A host
  that stopped logging is a finding.
- Baseline first, then hunt deviations. Unsigned binaries, processes
  in user-writable paths, LOLBin parent/child anomalies, new Run-key
  entries, and unexpected eBPF programs catch most commodity and many
  targeted intrusions.
- Everything in references/recipes.md assumes you already collected
  telemetry. If a hunt returns nothing, ask whether the data source
  exists before concluding the host is clean.
