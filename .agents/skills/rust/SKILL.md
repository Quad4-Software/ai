---
name: rust
description: >
  This skill covers the Rust toolchain and edition system, with the
  full Rust 2024 edition surface: language changes, stdlib, Cargo,
  rustdoc, rustfmt, and the migration workflow. Use when writing Rust,
  bumping an edition, reviewing edition-sensitive code, or pinning
  toolchain versions.
compatibility: rust-1.98
---

## When to use this skill

- You are choosing or bumping `edition` in Cargo.toml, or migrating a
  crate to Rust 2024.
- Code fails to compile after an edition bump and you need the actual
  rule change, not a guess.
- You need current toolchain facts: stable is 1.98.x (September 2026),
  Rust 2024 was released in 1.85.0.

## How to use

1. Read this file for the edition model and the Rust 2024 change list.
2. Read `references/edition-2024.md` for per-change before/after
   migration examples.
3. Migrate with `cargo fix --edition` on a clean tree, then build and
   review the lint output.

## Examples

- "Bump this crate to edition 2024 and fix the fallout."
- "Why does this `impl Iterator` now complain about a captured lifetime?"
- "Is `let` chaining usable on stable yet?"

# Rust editions and the 2024 edition

Editions are opt-in per package via `edition = "..."` in Cargo.toml.
They let rustc make small breaking changes while old crates keep
compiling. Editions so far: 2015, 2018, 2021, 2024. Edition is per
package, not per workspace; dependencies on other editions link fine.
`rust-version` (MSRV) is orthogonal: it states the minimum toolchain,
not the language dialect.

Current state (September 2026): stable is rustc 1.98.1. Rust 2024
shipped in 1.85.0 (February 2025). `cargo new` defaults to 2024.
There is no Rust 2027 yet; a 2027 edition is not announced.

## Rust 2024 language changes

Breaking changes and behavior shifts, grouped by how often they bite
in real code.

High-impact:

- **RPIT/impl-Trait lifetime capture**: `impl Trait` return types now
  capture ALL in-scope lifetimes by default. Previously lifetimes had
  to be named in the bound (`+ 'a`). Code that relied on non-capture
  now needs `+ use<..>` precise capturing, and the `Captures`/outlives
  tricks are obsolete. Most common edition-2024 breakage.
- **Tail expression temporary scope**: temporaries in the tail
  expression of a block, function, or closure may drop BEFORE local
  variables, not after. Code relying on the old order (a MutexGuard in
  the tail expr while a local is read by a Drop impl) can deadlock or
  UB-observe changes; the `tail_expr_drop_order` lint flags it.
- **if-let temporary scope**: temporaries in an `if let` scrutinee drop
  before the else branch runs, instead of after the whole expression.
  `if let Ok(x) = lock.read() else { .. }` no longer holds the lock in
  the else arm.
- **Match ergonomics restrictions**: when the default binding mode is
  not `move` (scrutinee matched by reference), bindings may not declare
  `mut`, `ref`, or `ref mut`, and `&`/`&mut` patterns can only appear
  before a binding-mode shift. Fix by matching explicitly or binding by
  value. Hard error in 2024.
- **`static mut` references denied**: taking `&`/`&mut` to a
  `static mut` is a deny-by-default `static_mut_refs` error in 2024.
  Migrate to `UnsafeCell`/`SyncUnsafeCell` or `addr_of_mut!`.

Safety surface tightening:

- **`unsafe extern` blocks**: `extern "C" { .. }` must be written
  `unsafe extern "C" { .. }`, since declaring an item is itself unsafe.
- **Unsafe attributes**: `no_mangle`, `export_name`, `link_section`
  must be written `#[unsafe(no_mangle)]` etc.
- **`unsafe_op_in_unsafe_fn` warns by default**: unsafe ops inside an
  `unsafe fn` body need explicit `unsafe { }` blocks.

New capability:

- **`let` chains**: `if let A = x && let B = y { }` and the same in
  `while`. Edition-gated to 2024+ because `let` in condition position
  needed a grammar carve-out.
- **`gen` keyword reserved**: `gen` identifiers need `r#gen`. Reserves
  space for gen blocks (still unstable) and the future generators
  feature. `gen` blocks do not exist on stable yet.
- **Reserved syntax**: unprefixed `#"foo"#` guarded strings and runs of
  2+ `#` are reserved for future use.

Type inference / macros:

- **Never type fallback**: never-to-any coercions fall back to `!`
  instead of `()`. Was edition-2024-only; since Rust 1.100 it applies
  on all editions and the migration lint is gone.
- **`expr` macro fragment** now also matches `const { }` blocks and
  `_` underscore expressions; `expr_2021` preserves the old match set.
- **`missing_fragment_specifier` is a hard error**: bare `$x` in
  `macro_rules!` must have a specifier.

## Standard library changes

- Prelude gains `Future` and `IntoFuture`. Custom traits named `Future`
  or a method `.into_future()` can become ambiguous.
- `Box<[T]>` implements `IntoIterator` in all editions, but
  `boxed.into_iter()` only resolves to it in 2024+. Earlier editions
  keep autoref'ing to a slice iterator. A real silent-behavior change.
- Newly `unsafe fn`: `std::env::set_var`, `std::env::remove_var`, and
  `CommandExt::before_exec`. Treat process env mutation as unsafe
  because of shared-state races with threads.

## Cargo changes

- `edition = "2024"` implies `resolver = "3"`: the version-aware
  feature resolver prefers the latest compatible dep versions.
- Removed duplicate Cargo.toml keys: `[project]`, `default_features`,
  `crate_type`, `proc_macro`, `dev_dependencies`, `build_dependencies`.
  Use the hyphenated/`[package]` forms.
- Since Rust 1.99, `default-features` on an inherited dep in a 2024
  package overrides the `[workspace.dependencies]` value instead of
  erroring.

## Rustdoc and rustfmt

- Doctests are merged into a single binary per crate (much faster).
  Doctests that relied on separate binaries may need attention.
- `include_str!`'d doctests resolve nested `include!`/`include_str!`
  relative to the Markdown file, not the Rust source.
- `style_edition = "2024"` in rustfmt.toml applies the new formatting
  style independently of the code edition. Raw identifiers sort
  ignoring `r#`, and `Version` sorting changed; check diffs before
  mass-reformatting.

## Migration workflow

1. On stable: `cargo fix --edition` applies the `rust-2024-compatibility`
   lint group and auto-fixes what it can (needs a clean git tree).
2. Set `edition = "2024"`, run `cargo build` and `cargo test`.
3. Review remaining warnings manually; the dangerous ones are the
   silent drop-order changes (tail expr, if-let scrutinee) and
   `Box<[T]>::into_iter`.
4. CI: pin with `rust-toolchain.toml` `channel = "1.85"` or later; use
   `channel = "stable"` for rolling.

## Adjacent stabilized features worth knowing

Not edition-gated but landed around the same window and often
conflated with 2024: async closures `async |x| {}` (1.85), `#[diagnostic]`
attributes for custom error messages (1.78), `use<..>` precise
capturing syntax itself (usable on all editions since 1.82).
