# uv reference

uv is Astral's Rust toolchain: one binary covering pip, pip-tools,
virtualenv, pyenv, and the project workflow. Docs:
https://docs.astral.sh/uv/

## Install and self-management

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh   # standalone, no Python needed
uv self update                                    # standalone builds only
```

Distro/pip/cargo installs exist but self-update only works for the
standalone build. Pin uv in CI: pre-1.0 means minor bumps can carry
marked breaking changes.

## The three faces

| Face | Commands | Use |
| --- | --- | --- |
| Project | `uv init`, `uv add`, `uv remove`, `uv sync`, `uv lock`, `uv run`, `uv tree`, `uv export` | pyproject.toml + uv.lock workflow |
| pip interface | `uv pip install/sync/compile/freeze/list` | drop-in pip + pip-tools replacement |
| Tools/Python | `uvx` (uv tool run), `uv tool install`, `uv python install/list/pin` | CLI tools and interpreters |

## Project workflow

```bash
uv init myapp                 # packaged project: src/myapp, uv_build, [project.scripts]
uv init --no-package myapp    # old flat layout, no build system
uv add requests 'httpx>=0.28' # adds dep, locks, syncs
uv add --dev pytest           # dev group
uv sync --locked              # install exactly uv.lock (CI mode)
uv run script.py              # run inside the project env, syncing first
uv run --with rich script.py  # ephemeral extra dep
uv lock --upgrade-package foo # targeted lock refresh
uv export --format requirements-txt --hashes
```

- Lockfile is `uv.lock` (TOML), cross-platform and committed.
- `.python-version` pins the interpreter. `uv python install 3.13`
 fetches managed builds from python-build-standalone.
- `requires-python` in `[project]` gates resolution.
- Workspaces: `[tool.uv.workspace] members = ["packages/*"]` plus
 `[tool.uv.sources]` entries like `foo = { workspace = true }`.

## uv_build

Astral's own PEP 517 backend, default for `uv init` since 0.12:

```toml
[build-system]
requires = ["uv_build>=0.12,<0.13"]
build-backend = "uv_build"
```

Suitable for pure-Python packages. Native/complex builds still use
hatchling or setuptools. Pin the upper bound to the next minor.

## Indexes and sources

```toml
[[tool.uv.index]]
name = "internal"
url = "https://pypi.internal/simple"
default = true        # replaces PyPI as the default index

[tool.uv.sources]
internalpkg = { index = "internal" }   # per-package index pin
```

- `--default-index`/`--extra-index-url`/`--find-links` mirror pip flags.
- `--index-strategy` controls first-hit vs union across indexes, and
 `unsafe-best-match` opts into pip-like behavior. Keep the default
 for dependency-confusion resistance.
- Git/URL/path sources live in `[tool.uv.sources]`.

## Security-relevant flags

- `--require-hashes` on install/sync (and since 0.12, honored inside
 requirements.txt): every requirement must pin `==` and carry a
 secure hash. MD5-only is rejected since 0.12.
- `--exclude-newer 2026-09-01` (or RFC 3339): ignore releases after
 the timestamp. Per-package: `--exclude-newer-package pkg=date`.
- `--exclude-newer-package` accepts a cutoff per name, the escape
 hatch for urgent patches.
- Build isolation is default. Use `--no-build-isolation` only for trusted
 sdists. `uv sync --no-build` / `--no-binary` restrict artifact kinds.
- `UV_INDEX_URL`, `UV_DEFAULT_INDEX`, `UV_TOKEN`, `UV_NATIVE_TLS`,
 `UV_HTTP_TIMEOUT` are the env knobs. `uv auth` (preview) manages
 index credentials outside files.

## Gotchas

- `uv pip` needs a target: `--system`, `--python`, or a discovered
 `.venv` (VIRTUAL_ENV honored).
- `uv sync` removes packages not in the lockfile by default.
 `--inexact` preserves extras.
- Constraints (`-c constraints.txt`) bound versions without adding
 deps, same as pip.
- On this repo's Go-only convention: uv is for Python side tooling
 (scripts, gen-configs), not a Go module concern.
