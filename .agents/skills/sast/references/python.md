# Python linting and SAST

## Ruff (0.16.x, MIT)

The default Python linter and formatter, Rust-written, fully offline.
`ruff check [--fix] [--watch]`, `ruff format` (deliberately near-
identical to Black). Config in `[tool.ruff]` or `ruff.toml`.

~800 rules across prefixes: E/W pycodestyle, F pyflakes, I isort, N
naming, UP pyupgrade, **S flake8-bandit security**, B bugbear, SIM,
PTH, PL pylint subset, TRY, DTZ, ASYNC, D docstrings, FA, C90, RUF.

Security S rules worth enabling explicitly: S101 assert, S102 exec,
S105-S107 hardcoded passwords, S301 pickle, S311 random for crypto,
S323/S324 weak TLS and hashing, S501 requests verify=False, S506
unsafe yaml.load, S602-S612 subprocess shell and SQL injection.

```toml
[tool.ruff.lint]
select = ["E", "F", "I", "UP", "B", "SIM", "S", "PTH", "RUF"]
```

Flags: `--fix`, `--fix-only`, `--unsafe-fixes`, `--add-noqa`,
`--statistics`, `--output-format=sarif`. Caveats: no type checking,
not every flake8 plugin is ported, v0.16 had a small breaking set.

## Bandit (1.9.4, Apache-2.0)

Still maintained but mostly redundant once ruff `S` rules are on.
`bandit -r . -f json`, `.bandit` config, `# nosec B603` suppressions.
Fully offline.

## Pylint (4.0.x, GPL-2.0)

Slower, deeper - refactoring and duplication checks ruff does not
cover. Complements ruff rather than competing.

## Type checkers as bug catchers

All fully offline once installed.

- **mypy 2.3.0** (MIT) - 2.x line since May 2026.
- **pyright ~1.1.412** (MIT) - npm-distributed, Node dependency.
- **basedpyright 1.39.x** (MIT) - community fork, stricter
 recommended/all modes, Pylance-like LSP outside VS Code.
- **ty** (Astral, MIT) - beta, Rust, 10-100x faster, unstable API.
- **Pyrefly 1.0** (Meta, MIT, stable May 2026) - Rust, LSP built in,
 runs Instagram-scale codebases.

Pick basedpyright or pyrefly for strictness today, watch ty.
