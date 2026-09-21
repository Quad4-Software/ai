# Linux threat hunting

## Telemetry: auditd

The reliable exec telemetry source. Use the Neo23x0/auditd ruleset
(broad telemetry, detect in SIEM/Sigma):

```
-a always,exit -F arch=b64 -S execve -S execveat -k process_creation
-a always,exit -F arch=b64 -F euid=33 -S execve -k detect_execve_www
```

Watches on /etc/passwd, /etc/shadow, /etc/sudoers, /etc/audit/,
cron dirs, systemd dirs, SSH config, shell profiles, ld.so.preload.
Syscall rules for ptrace, memfd_create, bpf, namespace syscalls,
io_uring, userfaultfd, kexec, module loads, and a rootcmd rule
(euid=0, auid>=1000). Query with `ausearch -k`, `aureport`.

## eBPF runtime detection

- Falco 0.44.x (CNCF graduated): modern eBPF probe default, rule
 engine, strong for shell-spawned-by-webserver, miners, escapes.
- Tetragon 1.7.x (Cilium): TracingPolicy CRDs, in-kernel filtering,
 works standalone without Kubernetes, runtime enforcement possible.
- Tracee 0.24.x (Aqua): eBPF forensics + detections, can capture
 dropped binaries and memory regions.
- Sysdig OSS 0.41.x: capture/replay and csysdig for interactive scap
 exploration.

## Logs and native telemetry

- auth.log/secure + journald (`journalctl -u ssh --since ...`).
- wtmp/btmp/utmp via last/lastb - correlate against auth.log Accepted
 lines since interactive wipes happen.
- bash_history is unreliable (HISTFILE tricks) - bonus evidence only.
- psacct/acct process accounting gives per-user command history that
 survives shell tricks.
- systemd transient units (`systemd-run`) show in journal and
 /run/systemd/transient.

## Persistence checklist

- systemd units and timers incl. user units and transient units
- cron: /etc/crontab, /etc/cron.d/, cron.daily/hourly,
 /var/spool/cron, at jobs, anacron
- /etc/ld.so.preload and ld.so.conf.d (LD_PRELOAD rootkits)
- shell rc: ~/.bashrc, ~/.bash_profile, /etc/profile.d/, ~/.zshrc,
 ~/.bash_logout
- SSH: authorized_keys, ~/.ssh/config ProxyCommand/LocalCommand,
 sshd_config AuthorizedKeysCommand
- PAM modules (pam_exec.so), /etc/security/*
- kernel modules: lsmod vs /lib/modules, check /proc/sys/kernel/tainted
- eBPF rootkits (bpfdoor, TripleCross, Boopkit): `bpftool prog show`
 for unexpected programs, `ss -0` for raw sockets, audit `-S bpf`
- init.d, rc.local, udev rules, XDG autostart, MOTD scripts,
 ld.so.cache poisoning, binary tampering (debsums, rpm -Va)
- container layer: docker autostart containers, kubelet static pods

## Scanners and memory

- chkrootkit 0.59 (Jan 2026) and rkhunter 1.4.6 (2018, stale) -
 secondary signals only. YARA-X scanning plus Falco/Tracee runtime
 rules give better coverage today.
- Volatility 3 on Linux needs dwarf2json symbol packs. Plugins
 linux.pslist, linux.bash, linux.lsof, linux.elfs, linux.sockstat,
 linux.check_modules. Capture with LiME or AVML.
- Sysmon for Linux 1.5 exists (SysinternalsEBPF) but is niche -
 Falco/Tetragon/auditd are the primary options.
- GTFObins is the Linux LOLBAS: hunt SUID/capability/sudo abuse of
 find, awk, vim, nmap, perl, python.
- Hunt FOR post-compromise enum scripts (linPEAS etc.) in execve
 logs. Do not run them as a hunting step.
