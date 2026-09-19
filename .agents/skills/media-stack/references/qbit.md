# qBittorrent reference

- **Latest stable:** v5.2.3 (July 2026). v5.3.0beta1 exists and flips the
  default to libtorrent 2.1.x. Port 8080 for the WebUI and WebAPI.
- Official builds ship two libtorrent variants: the standard build uses
  libtorrent 1.2.x, the `lt20` build uses libtorrent 2.0.x (v5.2.3 ships
  lt 1.2.20 / lt 2.0.13). libtorrent 2.x changes disk I/O behavior and
  hashing. Stick to one variant per data set.
- `qbittorrent-nox` is the headless build. Docker images (LinuxServer.io,
  hotio) wrap it.

## WebUI and auth

- **No default credentials since v4.6.1.** `admin/adminadmin` is gone.
  nox builds print a random 9-char temporary password to stdout on every
  start until a real password is set:

  ```
  The WebUI administrator username is: admin
  The WebUI administrator password was not set. A temporary password is
  provided for this session: XXXXXXXXX
  ```

  Recover it with `docker logs <container>` or `journalctl -u
  qbittorrent`. Then log in and set a real password, or set
  `WebUI\Password_PBKDF2` in `qBittorrent.conf` directly. GUI builds with
  WebUI enabled but no password refuse to start the WebUI.
- **Two API auth models:**
  1. **SID cookie.** `POST /api/v2/auth/login` with form body
     `username=<u>&password=<p>` returns `200 Ok.` and `Set-Cookie: SID=...`.
     Send `Cookie: SID=...` on subsequent calls.
  2. **API key (v5.2.0+).** Generate one in WebUI settings, send
     `Authorization: Bearer qbt_...`. Skips the cookie dance entirely and is
     the right choice for *arr and script access.
- **Host/CSRF validation** breaks naive reverse proxies. The WebUI checks
  the `Host` header against a server-domains list and matches
  `Referer`/`Origin` to the target origin. Mismatches log "Invalid Host
  header" or "Referer header & Target origin mismatch" and return a blank
  "Unauthorized" page. Fixes: add the public domain to the
  server-domains list, enable `X-Forwarded-Host` support, or blank the
  `Referer`/`Origin` in the proxy (`proxy_set_header Referer "";`).
  `X-Forwarded-For` is honored for real client IPs.

## WebAPI v2 endpoints

All paths are relative to `/api/v2`. Method list is versioned in the
repo's `WebAPI_Changelog.md` and the wiki.

- **App:** `GET /app/version`, `GET /app/webapiVersion`,
  `GET /app/buildInfo`, `GET /app/preferences`,
  `POST /app/setPreferences` (`json={...}`), `GET /app/defaultSavePath`.
- **Torrents:**
  - `POST /torrents/add` accepts `urls` (newline-separated magnets/URLs)
    or multipart `torrents` file parts. Optional fields: `savepath`,
    `category`, `tags` (comma-separated), `paused`/`stopped`,
    `skip_checking`, `sequentialDownload`, `firstLastPiecePrio`,
    `autoTMM`, `contentLayout`, `ratioLimit`, `seedingTimeLimit`,
    `upLimit`, `dlLimit`, `rename`, `downloadPath`/`useDownloadPath`,
    `addToTopOfQueue`, `stopCondition`. Newer builds return
    `success_count`/`pending_count`/`failure_count`, HTTP 202 when
    pending, 409 when all fail.
  - `GET /torrents/info` (filter by `hashes`, `category`, `state`,
    `sort`), `GET /torrents/properties`, `/torrents/trackers`,
    `/torrents/files`, `/torrents/pieceStates`.
  - Control: `/torrents/pause`, `/resume`, `/delete` (`deleteFiles=true`
    removes data), `/recheck`, `/reannounce`, `/recheck`,
    `/setCategory`, `/addTags`, `/removeTags`, `/setLocation`,
    `/rename`, `/setShareLimits` (ratio + seeding time), `/topPrio`,
    `/bottomPrio`, `/setDownloadLimit`, `/setUploadLimit`.
    All take `hashes` as pipe-separated values or `all`.
- **Categories:** `GET /torrents/categories` (name -> `{savePath,
  downloadPath}`), `POST /torrents/createCategory` (`category`,
  `savePath`), `/editCategory`, `/removeCategories`. With Automatic
  Torrent Management on, assigning a category relocates the torrent to
  that category's save path.
- **Tags:** `GET /torrents/tags`, `/torrents/createTags`,
  `/torrents/deleteTags`.
- **Sync:** `GET /sync/maindata?rid=<n>` returns a full state snapshot
  on rid=0 and diffs after that. This is the polling endpoint every UI
  uses.
- **RSS:** `/rss/addFeed`, `/rss/addFolder`, `/rss/removeItem`,
  `/rss/refreshItem`, `/rss/items(withData)`, `/rss/setRule`,
  `/rss/renameRule`, `/rss/removeRule`, `/rss/rules`,
  `/rss/matchingArticles`. Rules drive the auto-downloader: regex
  filters, per-feed category and save-path assignment, add-paused.
- **Search plugins:** `/search/start`, `/search/status`,
  `/search/results`, `/search/plugins`.
- **Transfer:** `/transfer/info`, `/transfer/speedLimitsMode`,
  `/transfer/setDownloadLimit`, `/transfer/setUploadLimit`.

### Example: add a torrent under a category

```bash
curl -s -X POST "http://qbittorrent:8080/api/v2/torrents/add" \
  -H "Authorization: Bearer $QBT_KEY" \
  --data-urlencode "urls=magnet:?xt=urn:btih:..." \
  --data-urlencode "category=radarr" \
  --data-urlencode "savepath=/data/torrents/movies"
```

## Integration with the *arr apps

- Radarr/Sonarr/Lidarr talk to qBittorrent through this same API. In the
  *arr's download-client settings, give the host, port 8080, username and
  password, and a category. The v5.2+ API key is for direct API
  scripting. The *arr's client still logs in with username/password.
- The *arr assigns the category so the download lands in
  `/data/torrents/{movies,tv,music}` (see the shared `/data` layout in the
  parent skill). qBittorrent's per-category save path and auto-TMM keep
  that mapping enforced inside the client.
- **Remote Path Mappings** live on the *arr side and translate the path
  qBittorrent reports into the *arr container's view. Required when the
  client runs on another host or with different mounts.
- **Completed Download Handling -> Remove** in the *arr deletes the
  torrent and its data only after the client reports seeding complete and
  stopped. The client's own `ratioLimit`/`seedingTimeLimit` fields
  decide when "complete" happens, so per-indexer seed rules in the *arr
  should be stricter than the client's global limit.
- `POST /torrents/recheck` and `/torrents/reannounce` are the right tools
  when a *arr import reports a file missing but the data looks present.

## Operational notes

- `qBittorrent.conf` lives under the config dir (`~/.config/qBittorrent`
  or `/config/qBittorrent` in containers). Back it up with the
  `BT_backup` fastresume dir to survive a host rebuild without
  re-checking every torrent.
- The WebUI's default port 8080 collides with lots of things. Change it
  in settings or remap the container port.
- Expose the WebUI only on the LAN or behind auth. It has no rate
  limiting, and CVE history in the WebUI is long. Do not publish it.
- `qbit_manage` automates category/tag hygiene, cross-seed detection, and
  orphan cleanup. It checks *arr hardlinks before deleting, so it is safe
  to run against a shared `/data` layout.
