# Windows threat hunting

## Telemetry: Sysmon

- Sysmon 15.22.0 (Sept 2026). Since Feb 2026 it also ships as a
 built-in optional Windows 11 feature. The classic Sysinternals
 installer still exists. Runs as PPL. Schema 4.90 adds Event ID 29
 FileExecutableDetected.
- Event IDs: 1 ProcessCreate, 2 FileCreateTime, 3 NetworkConnect, 5
 ProcessTerminate, 6 DriverLoad, 7 ImageLoad, 8 CreateRemoteThread,
 9 RawAccessRead, 10 ProcessAccess, 11 FileCreate, 12/13/14 registry
 create/set/rename, 15 FileCreateStreamHash, 17/18 pipe create/
 connect, 19/20/21 WMI filter/consumer/binding, 22 DnsQuery, 23
 FileDelete archived, 24 ClipboardChange, 25 ProcessTampering, 26
 FileDeleteDetected, 27/28 FileBlock*, 29 FileExecutableDetected.
- Configs: olafhartong/sysmon-modular (actively maintained, Go
 toolkit v1.0 Sept 2026, per-version config families) or
 SwiftOnSecurity/sysmon-config. Both need per-environment tuning.

## Windows event IDs

Security log: 4688 process creation (enable command-line auditing),
4624/4625 logons (types 2,3,9,10), 4648 explicit creds (lateral
movement), 4672 special privileges, 4698/4702 scheduled task
created/updated, 4697 service installed, 1102 audit log cleared
(high signal), 4768/4769 Kerberos, 4728/4732/4756 group changes,
4103/4104 PowerShell module/script-block, 4105/4106 execution
start/stop. System log: 7045 service installed, 7036 state changes.

## Triage and EVTX hunting

- KAPE (Kroll, free gated): targets+modules triage collector, sync
 KapeFiles repo. Pair with EZ Tools: MFTECmd, PEcmd, AmcacheParser,
 RECmd, EvtxECmd, LECmd, bstrings.
- Chainsaw 2.16.x (Rust): fast Sigma hunting over EVTX, MFT parsing,
 gap analysis.
- Hayabusa 4.1.x (Rust): fullest Sigma support incl. v2 correlation
 rules and neq modifiers, ATT&CK v19 mapping, timeline output.
- Velociraptor 0.77.x: VQL hunts across a fleet, artifact exchange,
 Windows.KapeFiles.Targets, Windows.Forensics.*, offline collector
 for agentless triage.
- osquery 5.23.x: processes, services, autoruns, scheduled_tasks,
 registry, powershell_events, windows_events tables.

## Sysinternals

Autoruns (`autorunsc64 -a * -h -s -c` for CSV sweeps), Process
Explorer (VT integration, signer column), TCPView, Sigcheck
(`-i -h -e -u -vr` for unsigned/unchecked), Procmon. PsExec is dual
use - hunt for PSEXESVC service and ADMIN$ writes.

## Persistence checklist

- Run/RunOnce keys (HKLM/HKCU CurrentVersion\Run, Policies\Explorer)
- Winlogon Shell and Userinit
- IFEO Debugger values, SilentProcessExit, AppCertDlls
- Services (7045/4697), TaskCache, BITS jobs
- WMI subscriptions: __EventFilter + __EventConsumer + binding
 (Sysmon 19-21)
- LSA Authentication/Security Packages, Netsh helper DLLs, print
 monitors, COM hijacks under HKCU CLSID
- Autoruns or Velociraptor Windows.Sysinternals.Autoruns covers all.

## Forensic artifacts

Prefetch (execution evidence, check enabled per SKU), ShimCache and
Amcache (execution), MFT + $UsnJrnl (file timeline), SRUM
(per-process network bytes, 30-day, good for exfil), LNK files, jump
lists, UserAssist, BAM/DAM, RecycleBin $I, RDP bitmap cache.

## Logging depth

Enable PowerShell Script Block Logging (4104), Module Logging
(4103), Transcription via GPO. ETW providers (Kernel, Threat-
Intelligence, DNS, PowerShell) back most EDR telemetry - attackers
patch EtwEventWrite to blind it, so hunt for missing telemetry.
PSReadLine history is a bonus artifact, not detection.

## Memory

Volatility 3 (2.28.x): pslist/psscan, cmdline, netscan, malfind,
hivelist, mftparser. Windows ISF symbols auto-fetch from MS PDBs.
