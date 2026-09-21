# Hunt recipes

## Unsigned executables from user-writable dirs (Windows)

Sysmon EID 1 with Image under AppData/Temp and unsigned:

```powershell
Get-WinEvent -FilterHashtable @{LogName='Microsoft-Windows-Sysmon/Operational'; Id=1} |
  Where-Object { $_.Message -match 'Image: C:\\Users\\[^\\]+\\AppData\\(Local\\Temp|Roaming)' -and
                 $_.Message -match 'SignatureStatus: Unsigned|Unavailable' }
```

Hayabusa/Chainsaw cover this via Sigma (proc_creation_win_susp_*
rules + signature-status pivot). Velociraptor: Windows.Sys.ProcessCreation
joined on authenticode.

## Run-key persistence sweep (Windows)

```powershell
Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run',
                 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run',
                 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run'
```

Flag image paths outside Program Files/Windows, unsigned binaries,
script interpreters, and HKCU entries (no UAC needed). Correlate
with Sysmon EID 13 for creation time and responsible process. Fleet
version: Velociraptor Windows.Sysinternals.Autoruns.

## Shell spawned by a web server (Linux)

auditd: `-a always,exit -F arch=b64 -S execve -F euid=33 -k
detect_execve_www` then `ausearch -k detect_execve_www -i` filtered
for sh/bash/dash/python/nc/socat/perl. Tetragon equivalent:
TracingPolicy on execve where parent comm is nginx/apache2/httpd/
php-fpm and child is a shell. osquery with process_events:

```sql
SELECT * FROM process_events
WHERE parent IN (SELECT pid FROM processes
                 WHERE name IN ('nginx','apache2','httpd'))
  AND path LIKE '%/bin/%';
```

## Recently-registered domains

Join Sysmon EID 22 (DnsQuery) or Zeek dns.log against an NRD feed
(whoisds, openintel). In Zeek, load the intel framework with the NRD
list and alert on dns.log/http.log hits. RITA does threat-intel
checking as part of import.

## Beaconing (network)

`rita import /opt/zeek/logs/current dataset-name`, then
`rita show-beacons` and filter score >= 90. Manual version: group
conn.log by (orig_h, resp_h, resp_p), compute inter-arrival stdev.
Low jitter + many connections + uniform small payloads is a C2
candidate - but modern C2 adds jitter to defeat naive periodicity,
so also check byte-count regularity and long-connection tables.

## Linux persistence one-liner audit

```bash
ls -la /etc/cron.* /var/spool/cron /etc/systemd/system ~/.config/systemd/user 2>/dev/null
cat /etc/ld.so.preload 2>/dev/null          # normally absent or empty
find /etc/profile.d ~/.bashrc ~/.profile -newermt "-30 days"
grep -r . /root/.ssh/authorized_keys ~/.ssh/authorized_keys
bpftool prog show                          # unexpected eBPF programs
ss -0                                      # raw sockets (bpfdoor-style)
cat /proc/sys/kernel/tainted               # out-of-tree/tainted modules
```

Diff against golden image with `rpm -Va` or `debsums` for binary
tampering.
