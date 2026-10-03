---
name: pnpm
description: >
  This skill covers pnpm, with focus on pnpm 12 (the Rust rewrite,
  stable August 2026) and its supply chain security controls. Use it for
  pnpm 11 to 12 differences, install and CI setup, workspace and lockfile
  behavior, and hardening installs against compromised packages:
  minimumReleaseAge, allowBuilds, trustPolicy, blockExoticSubdeps, and
  named registries.
---

## When to use this skill

- You are upgrading a project or CI pipeline from pnpm 11 to 12.
- You are configuring install-time supply chain protections in
  pnpm-workspace.yaml.
- You are debugging lockfile diffs, peer resolution, or git
  dependencies.
- You are choosing between npm, pnpm, Yarn, and Bun on security
  grounds.

## How to use

1. Read this file for the version map and the seven pnpm 12 differences.
2. Load [references/supply-chain.md](references/supply-chain.md) for the
   hardening settings, lockfile caveats, and the npm/Yarn/Bun
   equivalents.
3. Fall back to https://pnpm.io/ (docs now default to 12.x, selector
   keeps 11.x and 10.x).

## Examples

- "Why did CI fail with ERR_PNPM_UNRECOGNIZED_WORKSPACE_SETTINGS?"
- "Set up pnpm so a brand-new malicious release cannot install."
- "Explain why pnpm 12 rewrote my lockfile on first install."
- "Grep our CI for pnpm 12 breakage before upgrading."

# pnpm

## Version map

- **pnpm 12.x** (latest: 12.9.0, Oct 2026): a rewrite in Rust.
  Stable since 12.0 on Aug 26, 2026. Keeps pnpm 11 commands, flags,
  settings, and lockfile format.
- **npm `latest` now points at pnpm 12.** `npm i -g pnpm` installs 12
  and needs Node.js 22.13+ to install. The standalone installer needs
  no Node.js at all, and once installed pnpm 12 runs as a native
  executable. `pnpm self-update` upgrades a pinned or global 11.10+
  install to 12.
- **pnpm 11.x** is still maintained (11.28 line), with its docs behind
  the version selector.
- **pnpm 10.x** added `minimumReleaseAge` (10.16) and the two-document
  lockfile. Postinstall scripts in dependencies have been off by
  default since v10.

## pnpm 12: the seven differences

1. **Project-aware global bins.** A globally installed `node`, `deno`,
   or `bun` runs the version the project pins via `devEngines.runtime`
   or a runtime dependency. Outside a project you get the global copy.
   Controlled by `globalShims`. pnpm never creates shims on its own.
   `pnpm shim add yarn` creates one on request.
2. **Git deps are identities, not transports.** GitHub, GitLab, and
   Bitbucket specifiers (`github:`, `git+https:`, `git+ssh:`, bare
   owner/repo) all resolve through the host's HTTPS URL. SSH URLs are
   never recorded in the lockfile for those hosts, so a laptop lockfile
   installs on CI without keys. For private repos use
   `git config --global url."git@github.com:".insteadOf
   https://github.com/`. Old lockfiles with SSH entries need one
   `pnpm update <pkg>`. pnpm never rewrites a lockfile unprompted.
3. **Naming a package manager means the tool.** pnpm 12 installs and
   runs npm, Yarn (Classic, Berry, 6/zpm), Bun, Node, and Deno as real
   tools, not the same-named npm packages. `pnx yarn@4 install`,
   `pnpm add -g node@22`, and `pnpm add yarn` (records the project's
   packageManager field) all do what the name says.
4. **Deterministic cyclic lockfiles.** Cycles are cut at a fixed edge
   ordered by package ID. Reordering globs or package.json entries no
   longer changes the lockfile. Peer resolution is 2-3x faster and
   ~25% less memory on cycle-heavy workspaces. Expect a one-time
   lockfile diff on the first re-resolve.
5. **`packageImportMethod: auto` hardlinks first on Linux.** Reflink
   clone is now the fallback (btrfs installs roughly halve
   materialization time). Nothing changes on ext4 or macOS.
6. **`engineStrict` follows the edge.** An installed package whose
   regular `dependencies` reach an engine-incompatible package now
   fails even under `optionalDependencies`. Packages reachable only
   through optional edges still skip.
7. **`--resolution-only` is gone.** It fails outright. Use
   `pnpm peers check`, which reads issues from the lockfile with no
   re-resolution. Grep CI scripts for `--resolution-only` before
   upgrading.

Notable minor releases since 12.0: 12.6 added automatic dependency
deduplication, relocatable `node_modules`, `--save-types`, and
`file:`/`link:` protocols in catalogs. 12.7 added
`pnpm install --allow-build=<pkg>` (writes `allowBuilds`,
`--allow-build='!pkg'` denies), `pnpm publish --publish-wait-timeout`,
and let the global `node` shim follow `.nvmrc`/`.node-version`. 12.8
warns when `pnpm pack`/`pnpm publish` would ship a `.env` file and
accepts every setting as `--config.<name>=<value>`. 12.9 adds a
per-registry `networkConcurrency` setting and runs pnpm in StackBlitz
WebContainers.

## Settings live in pnpm-workspace.yaml

Workspace globs and pnpm settings share `pnpm-workspace.yaml`. pnpm 12
errors on unrecognized keys (suggests the closest real name) when the
project's version pin is satisfied, warns otherwise, and never fails
under `pnpm config`. A misspelled `minimumReleaseAge` silently dropped
the policy in pnpm 11. `.npmrc` in a repo no longer expands
environment variables (since pnpm 11.6, June 2026): a checked-in
`.npmrc` cannot exfiltrate `$HOME` secrets into registry URLs or
tokens.

## Core mechanics

- Content-addressable global store + hardlinks: one copy of each file
  on disk, shared across projects.
- Non-flat `node_modules`: real deps live under `.pnpm`, direct
  dependencies are symlinked. Phantom dependencies fail by
  construction.
- `pnpm install --frozen-lockfile` is the CI default (auto-detected in
  CI), and commit `pnpm-lock.yaml`.
- `pnpm-lock.yaml` is a two-document YAML file since v10: scanners that
  read only the first document report zero dependencies without
  erroring.

## Supply chain

pnpm ships the strongest install-time controls of the JS package
managers: build-script approval, release-age cooldown, trust policy,
exotic-subdep blocking, and registry pinning. Details and the full
setting reference: [references/supply-chain.md](references/supply-chain.md).

## Sources

- Docs and blog: https://pnpm.io/, https://pnpm.io/blog/releases/12.0,
  https://pnpm.io/blog/whats-different-in-pnpm-12
- Supply chain guide: https://pnpm.io/supply-chain-security
- Releases: https://github.com/pnpm/pnpm/releases
