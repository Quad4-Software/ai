# Poetry reference

Poetry 2.5.x (latest 2.5.1, Sept 2026). Dependency manager, virtualenv
manager, and PEP 517 build frontend in one. Docs:
https://python-poetry.org/docs/

## Install

```bash
pipx install poetry       # recommended: isolated env, easy upgrades
poetry self update        # inside the pipx install
```

## pyproject.toml layout

Since 2.0 the standard `[project]` table is primary. `[tool.poetry]`
is optional and only needed for Poetry-only fields (`packages`,
extra source config, legacy `dev-dependencies` style groups).

```toml
[project]
name = "myapp"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = ["requests>=2.32"]

[dependency-groups]          # PEP 735, preferred over [tool.poetry.group]
dev = ["pytest>=8"]

[build-system]
requires = ["poetry-core>=2.0"]
build-backend = "poetry.core.masonry.api"
```

Caveat: per PEP 517 a missing `[build-system]` should fall back to
setuptools. Poetry 2.4/2.5 still warns and uses poetry-core. A future
minor will flip the default to setuptools, so always declare the
`[build-system]` table.

## Daily commands

```bash
poetry new myapp            # scaffold (src layout)
poetry init                 # interactive pyproject.toml
poetry add requests         # resolve + lock + install
poetry add --group dev pytest
poetry install              # install from poetry.lock
poetry install --sync       # remove anything not in the lock
poetry update               # re-resolve within version bounds
poetry run pytest           # run inside the env
poetry env use 3.13         # pick interpreter
poetry env list / info      # env management
poetry build / publish      # sdist+wheel, upload to PyPI or --repository
poetry check --lock         # validate pyproject and lock freshness
```

- `poetry.lock` is committed. `poetry install` errors when pyproject
 moved ahead of the lock. `poetry lock` refreshes it.
- Virtualenvs: `poetry config virtualenvs.in-project true` puts `.venv`
 in the repo (recommended for tooling that assumes it).

## Config

`poetry config --list` shows all settings. Notable: `virtualenvs.in-project`,
`virtualenvs.create`, `repositories.<name>`, `installer.max-workers`,
and (new in 2.5) `installer.builtin-uninstall` (opt-in fast uninstall
that stops shelling out to `pip uninstall`).

## Plugins

- `poetry-plugin-export` - `poetry export -f requirements.txt`. Hashes
 included by default, `--without-hashes` opts out, `--output` for a
 file. Still needed for Docker/CI pip installs.
- `poetry-plugin-up` - `poetry up` bumps deps to latest allowed.

## Dependency sources

```toml
[[tool.poetry.source]]
name = "internal"
url = "https://pypi.internal/simple"
priority = "primary"      # primary | supplemental | explicit
```

`priority` controls lookup order. `explicit` means the source is only
used for packages that name it (dependency-confusion resistant):

```toml
[tool.poetry.dependencies]
internalpkg = { version = "^1.0", source = "internal" }
```

## uv interop

Poetry locks resolve against `poetry.lock`. uv resolves `uv.lock`. For
a Poetry-managed project, `uv pip install -r <(poetry export)` is the
bridge. `poetry build` still uses poetry-core. uv can consume the
pyproject fine since Poetry 2.0 made `[project]` standard metadata.
