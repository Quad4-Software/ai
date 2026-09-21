---
name: project-scaffold
description: >
  This skill covers scaffolding a new open-source project the
  old-school way: minimal README (name, one-liner, install, usage,
  license), 0BSD LICENSE, master branch, minimal .gitignore, and a
  minimal language skeleton for Go, Python, Node/TS, Rust, or C.
  Explicitly omits Code of Conduct files, badge walls, funding configs,
  issue/PR templates, editor configs, AI configs, and README
  project-layout sections. Use it when creating a new repo or project.
metadata:
  sources:
    - https://suckless.org/philosophy/
    - https://landley.net/toybox/license.html
    - https://go.dev/doc/modules/layout
    - https://doc.rust-lang.org/cargo/guide/project-layout.html
    - https://keepachangelog.com/
---

## When to use this skill

- Creating a new repository or standalone project.
- Writing a minimal README, LICENSE, or language skeleton.
- Auditing a fresh scaffold for boilerplate cruft.

## Philosophy

The suckless standard: a short usage section and a descriptive man
page, with the complete details in the source. The canonical README
example is dwm's: name, one-line description, requirements, install,
running, configuration. Under 50 lines. No badges, no TOC, no feature
marketing, no layout map.

A README answers three questions: what it is, how to install it, how
to use it. One clause of "why" in the opening line is free and covers
the most commonly missing category in README research (Prana et al.
2019, 4,226 sections studied). Everything else is earned by scale.

## The README template

```markdown
# name

One or two sentences saying what it is and what it does. Concrete
facts only.

## Install

    command to build or install

## Usage

    name --flag <arg>

Short example with real input and real output.

License: 0BSD.
```

Optional only when earned: Requirements (non-obvious build deps, dwm
pattern), Configuration (one paragraph naming the mechanism), a
pointer to the man page instead of duplicated flag docs, Contributing
(one line, only if contributions are actually wanted).

Never emit: badge walls, TOC, feature bullets, "why X is awesome",
comparison tables, emojis, demo GIFs, project-layout sections,
sponsors, "built with" footers, duplicate license text.

## License: 0BSD

House default is 0BSD (ISC minus the attribution clause, OSI
approved, SPDX id `0BSD`). Toybox's license page describes it as a
public-domain-equivalent that avoids CC0/unlicense legal FUD. Offer
ISC or MIT only when asked or when the ecosystem demands it.

`LICENSE` file, no extension:

```text
Copyright (c) YEAR <author>

Permission to use, copy, modify, and/or distribute this software for any purpose
with or without fee is hereby granted.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND
FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.
```

Every generated source file gets `// SPDX-License-Identifier: 0BSD`
(or the language's comment equivalent) on line 1 instead of a full
license header.

## Repo files

Always: README.md, LICENSE, .gitignore (project-specific lines only,
no template boilerplate), the language skeleton, and a Makefile when
the toolchain lacks a task runner.

Only when earned: a man page (any CLI), SECURITY.md (a real reporting
policy exists), CONTRIBUTING.md (rules-only, contributions actually
wanted), CHANGELOG.md (once tags exist, keep-a-changelog format),
NEWS/AUTHORS (GNU flavor).

Never: CODE_OF_CONDUCT.md, .github/FUNDING.yml, ISSUE_TEMPLATE,
PULL_REQUEST_TEMPLATE, CODEOWNERS, editor configs, AI tool configs
(copilot-instructions, CLAUDE.md, .cursor/), lint starter spam,
Dockerfile or CI (add when needed, not at scaffold time).

## Git conventions

- Default branch `master`. `git init` may still default to it
  depending on config. Force with `git checkout -b master` or set
  `init.defaultBranch master`. On GitHub, flip the default branch in
  repo settings since the web defaults to main.
- Linear history or short-lived branches with real merge commits.
  No git-flow, no commitlint, no conventional-commit tooling.
- Releases: annotated tags `git tag -a v1.2.3` from master, semver.
  Tags are immutable once pushed. A bad release gets a new tag.

## Language skeletons

Go (flat is idiomatic, go.dev/doc/modules/layout):

```
name/
  go.mod          (module example.com/name, go 1.27)
  main.go
  internal/pkg/pkg.go    (house rule: thin main, logic in internal/)
  Makefile README.md LICENSE .gitignore
```

Python (src layout per PyPA, prevents importing uninstalled code):

```
name/
  pyproject.toml  (hatchling backend, license = {text = "0BSD"})
  src/name/__init__.py  __main__.py
  README.md LICENSE .gitignore
```

A script-scale project can be a single name.py plus README/LICENSE.

Node/TS:

```
name/
  package.json    (name, version, type:module, license:"0BSD", minimal scripts)
  tsconfig.json   (NodeNext, strict, outDir dist)
  src/index.ts
  README.md LICENSE .gitignore
```

Rust (`cargo new` already emits the minimum):

```
name/
  Cargo.toml      (edition 2024, license = "0BSD")
  src/main.rs
  README.md LICENSE .gitignore
```

C, suckless style:

```
name/
  name.c  config.def.h  config.mk  Makefile  name.1  README  LICENSE
```

config.mk carries PREFIX, MANPREFIX, CC, CFLAGS. Makefile is POSIX
make with all/clean/install/uninstall and DESTDIR support. No
autoconf/cmake ceremony.
