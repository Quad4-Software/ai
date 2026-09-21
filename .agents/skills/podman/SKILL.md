---
name: podman
description: >
  This skill covers Podman, the daemonless, rootless-first container
  engine, with focus on Podman 6.x (June 2026). Use it for v6 breaking
  changes (slirp4netns, cgroups v1, BoltDB removed), Quadlet systemd
  units, rootless networking (pasta, netavark), podman machine, the
  Docker-compatible CLI and API socket, registries.conf, and migrating
  from Docker. For Docker Engine and Compose see the docker skill.
---

## When to use this skill

- You are upgrading to Podman 6 or fixing what it broke.
- You are writing Quadlet .container/.volume/.network/.kube units.
- You are running rootless containers and hitting networking or
  permission edges.
- You are migrating Docker workflows, or pointing docker-compatible
  tooling at podman's socket.

## How to use

1. Read this file for the v6 map and the mental model.
2. Load [references/quadlet.md](references/quadlet.md) for unit files,
   and [references/rootless.md](references/rootless.md) for
   networking, storage, and permissions under rootless.
3. Fall back to https://docs.podman.io/ and
   https://github.com/containers/podman/blob/main/RELEASE_NOTES.md.

## Examples

- "Why does podman 6 fail after upgrading from 5.6?"
- "Write a Quadlet for a web service with a named volume."
- "Explain pasta vs slirp4netns after the v6 removal."
- "Point docker-compose at podman."

# Podman

## Version map

- **Podman 6.1.0** (Aug 2026) is current stable. v6.0 shipped June 24,
  2026. The **5.8.x** line still gets fixes (5.8.4, June 2026) for
  conservative upgrades.
- Bundled components move together: netavark + aardvark-dns v2, conmon,
  containers/common, Buildah for builds.

## v6.0 breaking changes

- **slirp4netns removed.** Pasta is the only rootless network
  backend (default since 5.0). `--network-cmd-path` is gone.
- **cgroups v1 removed.** Needs cgroup v2 + systemd on the host.
- **BoltDB backend removed.** SQLite only. Migrate on 5.8 first
  (`podman system migrate`) and reboot, before jumping to 6.
- **Netavark drops iptables.** nftables only (default since Fedora 41).
  Hand-written iptables rules for podman networks must be re-expressed.
- **Config rework.** containers.conf parsing unified. Remote clients
  (Windows/Mac podman-remote) get a client/server split, so settings
  must live on the side that acts.
- **Quadlet changes** and new paths: `/usr/share/containers/systemd/
  users` and `.../users/${UID}` let distros ship Quadlets. Manpages
  split per unit type.
- New: multiple static IPs per container, `podman machine
  --import-native-ca` (VM trusts host CAs each boot), `podman exec
  --no-session`, `.volume` UID=/GID=/Options= keys, anonymous volumes
  in `.container` Mount=, `podman quadlet list --filter status=`.

## Mental model vs Docker

- **Daemonless.** `podman run` forks conmon + the container directly.
  No dockerd. The daemon-equivalent is `podman system service`, an
  opt-in Docker-compatible REST API socket.
- **Rootless by default.** Each user gets their own storage and socket
  at `/run/user/$UID/podman/podman.sock`. Set
  `DOCKER_HOST=unix:///run/user/$UID/podman/podman.sock` (or
  `podman system service` socket activation) for docker-compatible
  tooling.
- **Fork/exec model:** `podman generate systemd` is deprecated. Quadlet
  is the supported path to systemd integration.
- **Pods are native.** `podman pod create` groups containers sharing
  namespaces, matching k8s pod semantics.
- **`podman kube play`** runs k8s YAML locally (Quadlet `.kube` units
  wrap it). `podman generate kube` was deprecated in v5. Write Quadlet
  or author YAML instead.

## Registries and short names

- Short names (`nginx` vs `docker.io/library/nginx`) prompt or fail
  depending on `unqualified-search-registries` in
  `/etc/containers/registries.conf` (rootless:
  `~/.config/containers/registries.conf`). Pin fully qualified names
  in scripts. A silent search list is a typosquat vector.
- `registries.conf` controls mirrors, `blocked`, `insecure` (HTTP),
  and per-registry `prefix` rewrites. `policy.json` gates signatures
  (sigstore/cosign verification supported).
- `podman login` writes `auth.json` under `$XDG_RUNTIME_DIR/containers`.

## Builds and compose

- `podman build` delegates to Buildah. Dockerfile/Containerfile syntax
  identical to docker build.
- `podman compose` shells out to a compose provider: `podman-compose`
  (Python, third-party) or `docker-compose` against the podman socket.
  Not built in.

## podman machine

On macOS/Windows (and optionally Linux for isolation), `podman machine
init` + `start` runs a Fedora CoreOS VM hosting the engine. v6 adds
`--import-native-ca` on init/set to import host CAs each boot. The
client talks to the VM over the API socket.

## Sources

- Docs: https://docs.podman.io/
- Releases: https://github.com/containers/podman/releases
- v6 change list: Fedora Wiki Changes/Podman6, LWN.net June 2026
- containers/* config references: https://github.com/containers/common
