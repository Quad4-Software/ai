# Electron inside Windows sandboxes and Job Objects

Why wrapping an Electron app in Windows containment breaks it, and
what to do.

## How the Chromium sandbox launches children

The broker (main) process spawns each target (renderer, GPU,
network, utility) suspended, then applies three confinements:

1. A restricted token - renderers lock down to USER_LOCKDOWN after
 warm-up, GPU/network get LPAC tokens.
2. A fresh Job Object - renderer jobs set JOB_OBJECT_UILIMIT_*
 (clipboard, handles, global atoms, display settings) plus
 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE and sometimes
 JOB_OBJECT_LIMIT_ACTIVE_PROCESS=1 or PROCESS_MEMORY.
3. Untrusted/Low integrity level plus an alternate winstation and
 desktop - alternate desktop creation is a hard CHECK at startup,
 which is why Session 0 / services fail outright.

## The Job Object nesting rule

Windows allows a jobbed process to join a second job (nesting) only
if **neither job sets UI restrictions**, and only on Windows 8+ /
Server 2012+. Chromium's sandbox jobs always set UI restrictions. So
if the Electron main process already lives inside an outer job with
UILIMITs - a CI runner, an EDR containment job, a host sandbox,
Codex-style sandboxes - AssignProcessToJobObject fails, the suspended
child is killed, and you see:

- `render-process-gone` with `reason: 'launch-failed'` (often
 exitCode 18)
- GPU process crash loops ending in `GPU process isn't usable.
 Goodbye.` (fatal, whole app exits)
- Or a blank window.

Compounding outer limits: JOB_OBJECT_LIMIT_ACTIVE_PROCESS caps the
whole tree's process count, JOB_OBJECT_LIMIT_JOB_MEMORY applies the
most restrictive commit cap across the chain, and
KILL_ON_JOB_CLOSE on the outer job means closing its handle kills
the app.

## The V8 sandbox angle

The V8 sandbox reserves a ~1 TB virtual address cage (32 GB guards
both sides) and commits sandboxed-heap and ExternalPointerTable
memory inside it. The reservation itself does not count against job
commit limits, but every committed page does. An outer job with a
tight JOB_OBJECT_LIMIT_JOB_MEMORY can starve renderer commit and
crash it at startup.

## LPAC / AppContainer filesystem rules

GPU and network-service sandboxing uses Less Privileged AppContainer
tokens. LPAC processes can only touch objects ACL'd to
S-1-15-2-2 (ALL RESTRICTED APPLICATION PACKAGES) or the package SID.
If the install dir lacks that ACE, or carries unresolvable zombie
AppContainer SIDs (real bug: electron#51761, caused by another
tool's AppContainer sandbox leaving package SIDs under
%LOCALAPPDATA%), GPU init fails and the app dies. Fix:
`icacls <dir> /reset` then grant `*S-1-15-2-2:(OI)(CI)(RX)`.

Electron cannot run inside a real AppContainer at all (electron
#14548). MSIX packages must declare runFullTrust.

## Documented instances

- electron#49167: renderers launch-fail only when run elevated on
 machines with AppLocker (crbug 740132 blocks the sandboxed child).
- electron#51761 / openai-codex#27236: zombie AppContainer SIDs ->
 GPU crash loop.
- amd/gaia#1800: crashes inside Windows Sandbox (no GPU). Fix:
 `app.disableHardwareAcceleration()` or `--disable-gpu`.
- Microsoft winapp-cli guide: sparse MSIX-identity dev packaging
 needs `--no-sandbox` (dev-time only).
- Session 0 / service hosts: alternate-desktop CHECK kills the app.

## Workarounds, escalating

1. Fix the outer job: remove UI restrictions, allow breakaway
 (JOB_OBJECT_LIMIT_BREAKAWAY_OK), launch the app with
 CREATE_BREAKAWAY_FROM_JOB, relax process/memory limits.
2. Grant the LPAC SID on the install dir for GPU/network sandbox.
3. `app.disableHardwareAcceleration()` or `--disable-gpu` for
 GPU-less environments (Windows Sandbox, VMs).
4. `--disable-gpu-sandbox`, `--in-process-gpu` for GPU-only failures.
5. `--no-sandbox` (CLI or `app.commandLine.appendSwitch('no-
 sandbox')`) removes all renderer isolation - renderers run at the
 parent's integrity level. Never ship this where content is
 untrusted.
6. Last resorts: `--single-process`, `--no-zygote` on Linux.
