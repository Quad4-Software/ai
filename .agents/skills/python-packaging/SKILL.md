---
name: python-packaging
description: >
  This skill covers Python packaging and environment management with uv
  (Astral, Rust, 0.12.x) and Poetry (2.5.x). Use it when choosing between
  uv and Poetry, writing pyproject.toml, managing virtualenvs and Python
  versions, locking and exporting dependencies, or hardening installs
  (hashes, cooldowns, build isolation).
---

## When to use this skill

- You are starting a Python project and need the tool decision.
- You are converting pip-tools, virtualenv, pyenv, or conda workflows.
- You are writing pyproject.toml, configuring indexes, or exporting
  requirements.txt.
- You are pinning Python itself, or debugging a uv/Poetry lock.

## How to use

1. Read this file for current versions and the decision rule.
2. Load [references/uv.md](references/uv.md) or
   [references/poetry.md](references/poetry.md) for commands, config,
   and hardening.
3. Fall back to https://docs.astral.sh/uv/ and
   https://python-poetry.org/docs/ for anything not covered.

## Examples

- "Convert this requirements.in + pip-compile setup to uv."
- "Should this monorepo use uv workspaces or Poetry?"
- "Pin uv in CI and explain why."

# uv and Poetry

## Version map

- **uv 0.12.x** (latest 0.12.17, Sept 2026). Still pre-1.0: minor
  releases can carry marked breaking changes. Pin the tool itself in CI
  (`uvx uv@0.12.17`, installer checksum, or distro pin). uv 0.12.14
  shipped a regression that broke `uv pip install --system` in the
  official python images and was fixed same day in 0.12.15.
- **Poetry 2.5.x** (latest 2.5.1, Sept 2026). 2.0 (Jan 2025) made
  `[project]` PEP 621 metadata the primary table. `[tool.poetry]` is
  optional legacy. poetry-core 2.x is the build backend.

## What uv 0.12 changed

- `uv init` creates a **packaged project by default**: `src/` layout,
  `[build-system]` with `uv_build`, a `[project.scripts]` entry. The
  old bare `main.py` layout is behind `uv init --no-package`. Existing
  projects unaffected.
- Legacy sdist formats (`.tar.bz2`, `.tar.xz`) are rejected per
  PEP 625, including in existing lockfiles. `.zip` sdists still work.
- `uv_build` upper bounds need `<0.13` to admit 0.12.

## Decision rule

- **uv** when you want one tool for everything: Python installs
  (`uv python`), project envs (`uv sync`), scripts (`uv run`/`uvx`),
  pip-compatible ops (`uv pip`), workspaces, and a build backend
  (`uv_build`). Fast (Rust), single binary, no Python needed to
  bootstrap.
- **Poetry** when the project already locks with it, or you want its
  mature publish workflow (`poetry build`, `poetry publish`) and plugin
  ecosystem (`poetry-plugin-export` for requirements.txt).
- They coexist: Poetry manages `pyproject.toml` + `poetry.lock`, uv
  manages `uv.lock`. Do not run both lockfiles in one repo.
- pip + pip-tools remains the zero-dependency baseline, and `uv pip` is a
  drop-in faster path for the same workflow.

## Supply chain posture (both)

- Commit the lockfile (`uv.lock` / `poetry.lock`). Install with
  `--locked`/`--frozen` in CI.
- `--require-hashes` (uv pip) and `uv export --format
  requirements-txt --hashes` give hash-pinned output. `poetry export`
  includes hashes by default (`--without-hashes` opts out).
- `uv --exclude-newer <ISO date>` is the cooldown equivalent of pnpm's
  minimumReleaseAge.
- uv 0.12 hardening: `--require-hashes` inside a requirements.txt now
  enables hash-checking (previously warned and skipped), MD5-only
  hashes are rejected in hash-checking mode, and declared artifact
  `size` must match the download.
- Index hygiene: pin `[[tool.uv.index]]` or Poetry sources explicitly
  and set `default = true` deliberately to keep dependency-confusion
  order predictable.
- Neither runs install-time arbitrary code the way npm postinstall
  does. Sdists still build, and build isolation is on by default in uv
  (`--no-build-isolation` only when you trust the source).

## Sources

- uv: https://docs.astral.sh/uv/, https://github.com/astral-sh/uv
- Poetry: https://python-poetry.org/docs/,
  https://python-poetry.org/blog/announcing-poetry-2.5.0/
