# pnpm supply chain security

Why it matters: compromised npm releases are usually detected and pulled
within hours, but the window between publish and removal is enough to
hit anyone installing latest. Controls below run at install time, so
they work without external scanning. Reference:
https://pnpm.io/supply-chain-security

## Threat model the settings address

- `preinstall`/`install`/`postinstall` scripts execute arbitrary code at
 install time. Most historical compromises (nx packages, the
 Shai-Hulud worm wave, chalk/debug takeovers, axios 1.14.1's phantom
 dep, mastra's easy-day-js) shipped malware there. Shai-Hulud 2.0
 moved to `preinstall` so the payload runs even when the install
 fails.
- Fresh malicious releases sit below the detection latency of Socket,
 Snyk, Aikido et al. Every 2025-2026 campaign was pulled within hours,
 which is exactly what minimumReleaseAge defends against.
- Trust regression: a package previously published by a trusted
 publisher (OIDC provenance) gets re-published without it, which is
 what a hijacked maintainer account looks like. Caveat: the May 2026
 TanStack attack minted tokens via a stolen OIDC token, so malicious
 releases can carry valid provenance. trustPolicy catches the
 downgrade case, not the stolen-credential case.
- Exotic sources: a transitive dep pointing at a git repo or tarball
 URL bypasses registry scanning entirely.
- Dependency confusion: a private name published to the public
 registry, or a same-name version on a second registry. May 2026 saw a
 45-package campaign squatting nine real internal org scopes.
- Phantom deps: a version bump adds one new transitive dependency that
 carries the payload (axios -> plain-crypto-js). Review every new
 transitive name in a lockfile diff.

For the full incident timeline and IoC checklist see
`bug-hunting/references/supply-chain.md`.

## Build script gating

Dependency build scripts do not run by default (since pnpm 10).

```yaml
# pnpm-workspace.yaml
allowBuilds:
  esbuild: true
  sharp: true
```

- `pnpm approve-builds` interactively picks which pending builds to
 allow and writes `allowBuilds` (older spelling
 `onlyBuiltDependencies` still works).
- `ignoredBuiltDependencies` silences the warning for packages whose
 scripts you deliberately skip.
- `strictDepBuilds: true` turns unreviewed build scripts into a hard
 install error instead of a warning.
- `dangerouslyAllowAllBuilds: true` re-enables everything. Do not use
 it. Approve per package so a compromised version of a package that
 never had scripts cannot suddenly run one.

## Release-age cooldown

```yaml
minimumReleaseAge: 1440            # minutes. Default 1440 (1 day) since v11
minimumReleaseAgeExclude:
  - webpack
  - '@myorg/*'
  - 'nx@21.6.5'                    # exempt one exact version
minimumReleaseAgeStrict: true      # fail instead of falling back when
                                   # no in-range version is old enough
minimumReleaseAgeIgnoreMissingTime: true  # skip registries with no time field
```

- Unit is minutes (durations like `"7 days"` also parse). 10080 = one
 week.
- Applies to transitive deps too. Set `minimumReleaseAge: 0` to opt
 out.
- Excluding exact versions is how urgent security patches bypass the
 cooldown without disabling it.

## Trust policy

```yaml
trustPolicy: no-downgrade          # off (default) | no-downgrade
trustPolicyExclude:
  - 'chokidar@4.0.3'
trustPolicyIgnoreAfter: 525600     # skip trust checks for releases
                                   # older than N minutes
```

`no-downgrade` fails the install when a package's trust evidence drops
versus earlier releases: trusted-publisher provenance downgraded to
provenance-only or to nothing. `trustPolicyIgnoreAfter` keeps old
packages (published before provenance existed) from tripping it.

## Other controls

- `blockExoticSubdeps: true` - transitive deps may only come from
 registries, which blocks git and tarball URLs below the root.
- `namedRegistries` - pin scoped or private names to a specific
 registry. Since v11.20 the lockfile records them as
 `name@registryName:version`, so a same-name package on the default
 registry cannot substitute.
- `verifyDepsBeforeRun` - checks the lockfile against package.json
 before `pnpm run`, catching stale installs.
- Integrity: tarball hash mismatches are fatal. `--update-checksums`
 rewrites them. Only run it after verifying the tarball source.
- Lockfile is a two-document YAML file. Audit your SBOM/scanner: tools
 reading only document one see no dependencies and report clean.

## Cross-package-manager equivalents

| Control | npm | pnpm | Yarn Berry | Bun |
| --- | --- | --- | --- | --- |
| Release cooldown | `min-release-age` (days, npm 11.10+, Feb 2026) | `minimumReleaseAge` (min, 10.16+, default 1440 in v11) | `npmMinimalAgeGate` (min, 4.10+) | `install.minimumReleaseAge` (sec, 1.3+) |
| Build scripts | `--ignore-scripts` (all or nothing) | `allowBuilds` per package | `enableScripts` / cache | trustedDependencies |
| Trust regression | none | `trustPolicy` | none | none |

Pick resolver-level cooldown (pnpm/npm/Yarn/Bun setting) or Dependabot
`cooldown`, not both: the resolver has no security exception and will
block Dependabot's own urgent update PRs.

## Recommended baseline

```yaml
# pnpm-workspace.yaml
minimumReleaseAge: 2880            # 2 days, or 10080 for a week
blockExoticSubdeps: true
trustPolicy: no-downgrade
strictDepBuilds: true
allowBuilds:
  # approve per package, keep the list short
```

Plus: commit pnpm-lock.yaml, pin the pnpm version via
`packageManager`/`devEngines.packageManager`, use
`--frozen-lockfile` in CI, and pin GitHub Actions to full SHAs (this
repo already does).
