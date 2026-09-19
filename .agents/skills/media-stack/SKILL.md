---
name: media-stack
description: >
  This skill covers the self-hosted media automation stack: Radarr, Sonarr,
  Lidarr, Prowlarr, Seerr (Overseerr/Jellyseerr successor), qBittorrent,
  Emby, and Jellyfin. Use it for stack architecture, the shared /data
  filesystem layout, hardlink and permission rules, the request-to-import
  pipeline, API auth patterns, remote path mappings, and upgrade caveats.
---

## When to use this skill

- You are deploying or debugging Radarr, Sonarr, Lidarr, Prowlarr, Seerr,
  qBittorrent, Emby, or Jellyfin together.
- You need the filesystem layout, UID/GID, or hardlink rules that make
  instant imports work.
- You are wiring apps to each other: Seerr to Radarr/Sonarr, the *arr apps
  to a download client, or the *arr apps to Emby/Jellyfin.
- You are calling the *arr, Seerr, qBittorrent, or media-server APIs.
- You are diagnosing "bad remote path mapping", permission, or stuck-queue
  errors.

## How to use

1. Read this file for the pipeline model, the shared `/data` layout, and the
   failure-mode map. Most stack bugs are layout or permission bugs.
2. Load a topic file from `references/` for per-app API and version detail:
   [references/arr.md](references/arr.md) for Radarr/Sonarr/Lidarr/Prowlarr,
   [references/seerr.md](references/seerr.md) for Seerr,
   [references/qbit.md](references/qbit.md) for qBittorrent,
   [references/emby-jellyfin.md](references/emby-jellyfin.md) for the media
   servers.
3. Fall back to the Servarr wiki at https://wiki.servarr.com and the TRaSH
   guides at https://trash-guides.info for settings not covered here.

## Examples

- "Why does Radarr copy files instead of hardlinking, and how do I fix it?"
- "Write a curl call that adds a movie to Radarr with a quality profile."
- "Explain the path a Seerr request takes until it lands in Jellyfin."
- "Diagnose 'Downloading into Root Folder' health check errors."
- "Set up qBittorrent categories so Sonarr and Radarr never collide."

# The media automation pipeline

A request moves through four layers. Each layer only talks to its
neighbors, through REST APIs.

```
user -> Seerr -> Radarr/Sonarr/Lidarr -> indexer -> download client
                                              |
                                              v
Emby/Jellyfin <- library scan <- import (hardlink/move) <- completed download
```

1. **Request layer.** Seerr takes user requests, enforces quotas and
   approvals, and forwards approved items to Radarr or Sonarr over their
   APIs. Lidarr has no request UI of its own.
2. **Library managers.** The *arr apps hold the wanted list. They search
   indexers, send releases to the download client, watch the queue, and
   import finished files into the media library.
3. **Indexers.** Prowlarr centralizes indexer credentials and syncs them
   into each *arr app. The *arr apps can also hold indexers directly.
4. **Download client.** qBittorrent receives the torrent (or an NZB client
   receives the nzb), writes files under its category save path, and
   reports state back to the *arr API.
5. **Import.** The *arr app detects the completed download, hardlinks or
   moves it into the root folder, renames it per the naming scheme, then
   tells Emby or Jellyfin to rescan via a Connect notification.
6. **Playback.** Emby or Jellyfin serves the library. It never talks to the
   download client or indexers.

Nothing in this chain polls the filesystem for new media. The *arr app
drives every transition through the download client's API. A "stuck"
download almost always means the client's reported path does not match a
path the *arr container can see.

## Component map

| App | Role | Port | API | Latest stable (Sept 2026) |
|---|---|---|---|---|
| Radarr | Movies | 7878 | v3 | v6.4.4.10685 |
| Sonarr | TV | 8989 | v3 | v4.0.19.2979 |
| Lidarr | Music | 8686 | **v1** | v3.1.0.4875 |
| Prowlarr | Indexer sync | 9696 | v1 | v2.5.2.5491 |
| Seerr | Requests | 5055 | v1 | v3.4.1 |
| qBittorrent | Download client | 8080 | WebAPI v2 | v5.2.3 |
| Emby | Media server | 8096 / 8920 | REST | 4.9.5.0 |
| Jellyfin | Media server | 8096 | REST | 10.11.11 |

Version detail and per-app API surface: see the reference files.

## The shared /data layout

The layout is the single most consequential design decision in the stack.
The convention (documented by the TRaSH guides and the Servarr docker
guide) is one top-level directory mounted identically into every container
that touches media:

```
data/
|-- torrents/           # download client writes here
|   |-- movies/         # qBittorrent category "radarr" -> savepath torrents/movies
|   |-- tv/             # category "sonarr"
|   `-- music/          # category "lidarr"
|-- usenet/             # if an NZB client is in play
|   |-- incomplete/
|   `-- complete/
`-- media/              # *arr root folders and Emby/Jellyfin libraries
    |-- movies/
    |-- tv/
    `-- music/
```

Every container mounts the same host path at the same container path,
`/data`. qBittorrent only needs `/data/torrents` (or even one subtree per
category), but mounting the same absolute host path into every container
keeps the paths identical on both sides of every API conversation.

### Why this layout matters

- **Hardlinks.** When the download directory and the library are the same
  filesystem, the *arr import creates a hardlink: a second directory entry
  pointing at the same inode. The torrent keeps seeding from
  `torrents/movies/...` while the library file exists at
  `media/movies/...` at zero extra space and zero I/O. Hardlinks cannot
  span filesystems, and inside a container every separate `-v` mount is a
  separate filesystem even if the host paths sit on one disk.
- **Atomic moves.** With one filesystem, a copy-free `rename()` replaces
  copy-then-delete. With split mounts (`/movies`, `/tv`, `/downloads` as
  separate volumes), every import is a full copy plus delete, and
  hardlinks are impossible.
- **Path identity.** The download client reports the completed file's
  path to the *arr API. If both see `/data/torrents/movies/file.mkv`, the
  path resolves with no translation. If they differ, the *arr needs a
  remote path mapping (below), and every mapping is a place for drift.

The failure signature of a broken layout is a health check reading
"Downloading into Root Folder", "Bad Remote Path Mapping", or imports that
hang at 100% CPU doing copies.

## Permissions

All containers must read and write the same files. Two working patterns:

- **Single user.** Run every container as the same `PUID`/`PGID` (usually
  1000/1000 on LinuxServer.io images). Simplest, fine for one host.
- **Shared group.** Each app runs as its own user but shares a group, and
  every image runs with umask `002` (hotio images expose `UMASK`.
  LinuxServer.io images default to `022` and need `UMASK=002` set
  explicitly). The downloader creates group-writable files so the *arr
  can move or hardlink them.

Mismatched ownership between the download client and the *arr container is
the most common import failure after bad path mapping. The error surfaces
in the *arr queue as a failed manual import, not in the download client.

## Remote path mappings

Settings -> Download Clients -> Remote Path Mappings translates the path a
download client reports into the path the *arr container sees. Two fields:
the client **Host** (must match the hostname configured in the download
client entry, e.g. `qbittorrent`) and the remote/local path pair.

Example: qBittorrent on a seedbox reports `/home/user/done/movie.mkv` and
the Radarr container mounts the seedbox share at `/remote/done`. The
mapping is host `seedbox-host`, remote `/home/user/done`, local
`/remote/done`.

If the download client and the *arr share identical mounts (the `/data`
layout), no mapping is needed. Mappings are only required across hosts or
when paths differ.

## Completed download handling and seeding

The *arr decides when a torrent leaves the client:

- **Remove Completed** deletes the torrent and its data once the client
  reports seeding complete and stopped. Keep it off until per-indexer seed
  rules are correct, or the *arr can delete files before they import.
- **Seed ratio/time goals** come from the indexer settings in the *arr
  (minimum seed time per indexer) and from qBittorrent's per-category or
  per-torrent ratio limits. The *arr's indexer-level rules should be
  stricter than the client's global limit, or the client finishes seeding
  first and the *arr removes the job while private trackers still expect
  seed time.
- **Categories** keep everything separable. Each *arr assigns a category
  (`radarr`, `sonarr`, `lidarr`), which in qBittorrent maps to a save path
  and optional auto-TMM relocation. `qbit_manage` and similar cleanup
  tools key off categories to avoid deleting files a hardlinked library
  still references.

## Wiring the apps together

- **Seerr -> Radarr/Sonarr:** Settings -> Services in Seerr. Give the
  *arr hostname, port, API key, and URL base. The test button returns the
  *arr's quality profiles and root folders, which Seerr stores as
  selectable defaults. Seerr supports a separate 4K *arr instance per
  type. Radarr/Sonarr v3/v4 only.
- **Prowlarr -> *arr:** Settings -> Apps in Prowlarr. Add each *arr with
  its API key. Prowlarr pushes indexers and keeps them in sync. App
  profiles control which indexer categories sync to which app.
- ***arr -> qBittorrent:** Settings -> Download Clients. Host, port 8080,
  the WebUI username/password, and the category. Test it from the *arr
  container's network, not the host's.
- ***arr -> Emby/Jellyfin:** Settings -> Connect. Add the media server
  with an API key. The *arr sends grab/import/upgrade/rename/delete
  events so the server rescans only the affected library. Remote path
  mappings do not apply here. The media server needs its own mount of
  `/data/media` (read-only is enough).

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| Import stalls, "bad remote path mapping" health check | Client and *arr see different paths | Add a remote path mapping, or unify mounts under `/data` |
| Import copies gigabytes slowly | Download and media dirs on separate mounts | Merge into one `/data` filesystem so imports are atomic moves/hardlinks |
| Permission denied on import | PUID/PGID mismatch or umask 022 | Shared group + `UMASK=002`, or one user everywhere |
| Torrent removed before seeding | Client seed limits beat *arr indexer rules | Raise client limits or per-indexer seed time in the *arr |
| Seerr test fails on hostname | Seerr resolving a name the *arr container can't | Use the Docker network service name, not `localhost` |
| Media server shows old files after rename | Connect notification missing or rescan off | Add the Emby/Jellyfin connection in the *arr. Check its API key |
| qBittorrent login rejected after update | Random temp password in nox builds | Read the temp password from `docker logs` and set a real one |

## Upgrading

- All the *arr apps release on `master`/`main` (stable) and `develop`
  (pre-release) channels. In Docker, never use the in-app updater. Pull a
  new image tag. Lidarr's in-container update breaks audio fingerprinting.
- The *arr apps keep their DB in `config.xml`-adjacent SQLite files under
  `/config`. Back up `/config` before major upgrades. There is no
  downgrade path after a schema migration.
- Sonarr v3 is EOL. v4 removed Release Profiles and replaced Preferred
  Words with Custom Formats. Lidarr v3.0.1 moved to .NET 8, dropped Basic
  Auth, and dropped linux-x86.
- Jellyfin 10.9+ rewrote the DB on EF Core. Upgrades are one-way. The
  installer drops a `library.db.bak` next to the DB. Skip-major-version
  jumps are not supported on some paths. 10.11 re-added a legacy
  migration route for pre-10.10 installs.
