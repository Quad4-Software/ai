---
name: typescript
description: >
  This skill covers TypeScript 6 vs 7 (tsgo, the native Go compiler) and
  how to adopt it safely. Use when upgrading TypeScript, adding a fast
  type-check path, or debugging why typescript-eslint or svelte-check
  break under TypeScript 7.
compatibility: typescript-7
---

## When to use this skill

- You are asked to upgrade a repo to TypeScript 7, or asked whether it is possible.
- You want a fast native type-check alongside the existing JS toolchain.
- typescript-eslint or svelte-check suddenly broke after touching the typescript dep.
- `eslint .` exploded with hundreds of bogus errors in `.svelte-check/` files.

## How to use

1. Check the installed versions: `npm view typescript version`, the repo's
   `typescript` dep, and typescript-eslint's peer range.
2. Do NOT point the main `typescript` dep at 7.x when the repo uses
   typescript-eslint, svelte-check, vue-tsc, ts-morph or ts-jest. They need
   the JS compiler API, which TypeScript 7.0 does not ship.
3. Add the native engine as a sidecar instead (see below).
4. If eslint starts flagging generated `++*.svelte.ts` files, gitignore and
   eslint-ignore `.svelte-check/` and delete the directory.

## Examples

- "Upgrade to TS 7" -> sidecar: `pnpm add -Dw @typescript/native@npm:typescript@7.0.2`
- "Fast typecheck" -> `svelte-check --tsgo --tsconfig ./tsconfig.json`
- "Lint broke with 700 errors" -> `rm -rf .svelte-check` and ignore it.

# TypeScript 7 (tsgo)

TypeScript 7.0 shipped 2026-07-08 as the first stable release of the Go
port of the compiler (project Corsa / typescript-go). It is a faithful
port, not a rewrite: same checking semantics, same diagnostics, just
native code plus shared-memory parallelism.

## What 7.0 actually gives you

- 8x to 12x faster full builds (VS Code codebase: 125.7s -> 10.6s).
- About 6-26% lower aggregate memory on Microsoft's benchmark projects.
- Editor speed: new LSP-based language server, multithreaded. VS Code has
  a dedicated TypeScript Native Preview extension; other editors consume
  it via LSP.
- Parallelism knobs: `--checkers` (checker workers, default 4),
  `--builders` (project-reference build workers), `--singleThreaded`.
- Stricter-by-default surface: `strict` and `esnext` are now the defaults,
  and TypeScript 6.0 deprecations became hard errors (for example
  `moduleResolution: "node"`/`node10`, `baseUrl`, ES5 targets, AMD/UMD emit).
- No new syntax or type features. Feature work resumes on the 7.x line
  after the port.

## The catch: no programmatic API in 7.0

The Go compiler exposes no stable embeddable API until 7.1. Consequences:

- typescript-eslint peers `typescript <6.1.0`; installing `typescript@7`
  alongside it is a peer conflict and crashes type-aware linting.
- Volar-based template checkers cannot run on it: svelte-check (default
  mode), vue-tsc, astro check, Angular template checking.
- ts-morph, ts-jest and anything importing the `typescript` module are out.
- Editor tooling must use the new LSP server, not `tsserver` from 7.

This is why "just bump typescript to 7" breaks real repos.

## Supported sidecar setup

Keep `typescript` on 6.x for lint and template checks. Add the native
compiler under an alias that tooling discovers:

```sh
pnpm add -Dw "@typescript/native@npm:typescript@7.0.2"
```

svelte-check resolves `@typescript/native` (preferred) or
`@typescript/native-preview` and requires the resolved package to be
typescript >= 7. Then:

```json
"check:tsgo": "svelte-check --tsgo --tsconfig ./tsconfig.json --fail-on-warnings"
```

Notes from doing this on weberr-xmpp (Svelte 5, ~200 files):

- `svelte-check --tsgo` writes generated `++*.svelte.ts` wrappers to
  `.svelte-check/`. That directory MUST be in .gitignore and in eslint's
  ignores list or `eslint .` lints the generated files and reports
  hundreds of bogus errors. This is the number one trap.
- tsgo surfaces implicit-any diagnostics the JS engine suppresses for
  Svelte snippet/callback params (for example
  `{#snippet failed(error, reset)}` and `<svelte:boundary onerror>`).
  Fix by annotating params explicitly; both engines then pass.
- knip cannot see the alias being consumed by svelte-check's package
  discovery, so add `@typescript/native` to knip ignoreDependencies.
- Diagnostics can drift slightly vs the JS engine; keep the JS `check`
  script as the authoritative gate and treat check:tsgo as a fast
  second opinion until 7.1 stabilizes the API.
- `typescript@6.0.3` is currently the latest 6.x and satisfies
  typescript-eslint's range.

## Official compatibility package

Microsoft ships `@typescript/typescript6`, which provides a `tsc6` binary
and re-exports the 6.0 JS API. That is the reverse alias (native as the
primary dep, JS API for tooling). Prefer the `@typescript/native` sidecar
when the repo's gate is still the JS toolchain; prefer typescript6 when a
repo has fully moved to native builds and only needs the API for a few
tools.

## What still cannot use TS 7 (as of 7.0.2)

- typescript-eslint (type-aware rules). Tracked in issue 10940.
- Volar-based checkers in default mode (see svelte-check --tsgo above).
- ts-morph (silently wrong output risk), ts-jest, ts-patch transformers,
  tsconfig language-service plugins.
- ESLint async parsers are unsupported anyway, so even a future API will
  need design work in typescript-eslint.

## Links

- Release post: https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/
- RC post with parallelism details: https://devblogs.microsoft.com/typescript/announcing-typescript-7-0-rc/
- Native previews and API progress: https://devblogs.microsoft.com/typescript/announcing-typescript-native-previews/
- Compiler repo and CHANGES.md (behavioral diffs): https://github.com/microsoft/typescript-go
- typescript-eslint tracking issue: https://github.com/typescript-eslint/typescript-eslint/issues/10940
- svelte-check --tsgo / --incremental PR: https://github.com/sveltejs/language-tools/pull/2932
- svelte-check tsgo tracking issue: https://github.com/sveltejs/language-tools/issues/2733
- 7-day release-age note: minimumReleaseAge gates in pnpm-workspace.yaml
  apply to these packages too; 7.0.2 is well past the window but brand new
  7.1.x releases will not be.
