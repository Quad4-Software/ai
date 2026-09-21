# Quadlet reference

Quadlet generates systemd units from declarative container files.
Supported since Podman 4.4, fully integrated since 4.8. `podman
generate systemd` is deprecated. Quadlet is the path.

## File types and locations

| Extension | Generates |
| --- | --- |
| `.container` | a service running one container |
| `.volume` | a podman volume |
| `.network` | a podman network |
| `.kube` | a service running `podman kube play` on a YAML file |
| `.pod` | a pod grouping containers |
| `.image` | a pulled image (pin by digest) |
| `.build` | an image built at unit time |
| `.artifact` | OCI artifact |

Paths, in load order (later overrides earlier):

- System: `/etc/containers/systemd/`, `/usr/share/containers/systemd/`
 (distro-shipped, new in v6: `users/` and `users/${UID}` subdirs)
- Rootless user: `~/.config/containers/systemd/`,
 `/etc/containers/systemd/users/${UID}/`

After editing: `systemctl --user daemon-reload` (or system daemon).
Quadlets are generated at daemon-reload, not install time.
`podman quadlet list` shows generated units (`--noheading`,
`--filter status=`, Pod field since v6).

## Minimal .container

```ini
[Unit]
Description=My web app

[Container]
Image=docker.io/library/nginx:1.29
PublishPort=127.0.0.1:8080:80
Volume=web-data.volume:/usr/share/nginx/html:ro
Network=web.network

[Service]
Restart=always

[Install]
WantedBy=default.target
```

## Keys that matter

[Container] section (subset):

- `Image=` (fully qualified or `name.image` ref), `ContainerName=`
- `PublishPort=host:container` (bind 127.0.0.1 by default on
 internet-facing hosts)
- `Volume=src:dst[:opts]` - `name.volume` refs a Quadlet volume, and
 `./rel` paths resolve relative to the unit file
- `Network=name.network` for a Quadlet network, or `host`/`none`/
 `pasta:...` mode strings. `Network=host` gives host networking
- `Environment=`, `EnvironmentFile=`, `Secret=name[,type=mount]`
- `User=`, `Group=`, `UserNS=`, `PodmanArgs=` (escape hatch)
- `HealthCmd=`/`HealthInterval=` for healthchecks
- `DropCapability=ALL` + `AddCapability=`, `NoNewPrivileges=true`,
 `ReadOnly=true`, `SecurityLabelDisable`, `SeccompProfile=`
- `Notify=healthy` for sd_notify-style readiness
- `AutoUpdate=registry|local` for `podman auto-update`
- v6: `Mount=` with no source creates an anonymous volume

[Volume]: `UID=`, `GID=`, `Options=`, `Copy=` (v6 additions on the
first three).

[Service]: standard systemd. `Restart=`, `TimeoutStartSec=`,
`Delegate=yes` when the container manages its own cgroups.

[Install]: `WantedBy=default.target` for rootless (user units have no
multi-user.target), `WantedBy=multi-user.target` for system units.

## Rootless specifics

- Units live in the user manager. `loginctl enable-linger <user>` keeps
 them running without login.
- Rootless Quadlets cannot bind ports <1024 (same sysctl limit as
 rootless generally).
- `GlobalArgs=` in the file prepends podman flags for every unit.

## Debugging

- `systemctl --user status web-app.service`, `journalctl --user -u`
- Generated unit text: `/usr/lib/systemd/system-generators/
 podman-system-generator` runs at daemon-reload. Inspect output via
 `podman quadlet print` (or the generated dir under
 `/run/user/$UID/systemd/generator/`).
- A unit that never appears usually means a parse error. Check
 `systemctl --user daemon-reload` output and the generator logs.
