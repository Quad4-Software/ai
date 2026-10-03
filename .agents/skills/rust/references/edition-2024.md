# Rust 2024 migration reference

Per-change before/after notes for edition 2024 (stabilized in Rust
1.85.0). Ordered by how likely they are to break real code. Sources:
the Rust Edition Guide and RFC 3501.

## RPIT / impl Trait lifetime capture

In editions <= 2021, `impl Trait` in return position captured only
lifetimes that appeared in the bounds. In 2024, ALL in-scope generic
parameters, including lifetimes, are captured by default.

```rust
// 2021: 'a is NOT captured; f borrows nothing from its arg.
fn f<'a>(x: &'a str) -> impl Iterator<Item = char> { .. }

// 2024: same signature captures 'a. If you relied on the return value
// outliving the borrow, this now errors. Opt out with precise capture:
fn f<'a>(x: &'a str) -> impl Iterator<Item = char> + use<> { .. }
// or name only what you want:
fn f<'a, T>(x: &'a T) -> impl Iterator + use<T> { .. }
```

`use<..>` bounds work on all editions since Rust 1.82, so libraries
can use them without bumping the edition. The `Captures` trick and
`impl Trait + '_` outlives workarounds are obsolete in 2024.

Migration symptom: `cannot return value referencing function
parameter` or unexpected borrow errors at call sites.

## Tail expression temporary scope

Temporaries created while evaluating a block's tail expression used to
outlive the block's locals. In 2024 they can drop BEFORE the locals.
This is a semantics change, not just a lint.

```rust
fn f() -> i32 {
    let c = RefCell::new(0);
    c.borrow_mut().max(5)  // temp borrow_mut guard
    // 2021: guard drops after locals. 2024: guard may drop first.
    // If a local's Drop impl borrows c, this flips lock ordering.
}
```

Watch for MutexGuard/RefCell/lock guards in tail position, and for
locals whose Drop impl touches the same state. The
`tail_expr_drop_order` lint (allow-by-default, or deny in the
compatibility group during migration) flags affected sites. Fix by
binding the temp: `let r = c.borrow_mut().max(5); r` is unchanged.

## if-let temporary scope

```rust
if let Ok(v) = map.lock().unwrap().get(&k) {
    // lock guard lives for the whole if in both editions
} else {
    // 2021: scrutinee temps still held here (deadlock if you relock)
    // 2024: scrutinee temps dropped before else
}
```

The change only affects the else branch. Code that deadlocked in 2021
starts working; code that silently relied on the extended lifetime must
bind the scrutinee first.

## let chains

```rust
if let Some(a) = x.get(0) && let Some(b) = x.get(1) && a < b {
    ..
}
while let Some(v) = it.next() && !v.is_empty() { .. }
```

2024+ only. `let` in a condition position requires the edition carve
out; on <= 2021 write nested `if let` or `match`.

## Match ergonomics reservations

When a pattern matches through a reference (default binding mode is
`ref`/`ref mut`, not `move`), 2024 forbids `mut`, `ref`, `ref mut` on
bindings, and `&`/`&mut` patterns can only appear before the binding
mode shifts.

```rust
let mut v = vec![(1, 2)];
match &mut v[..] {
    // 2021 compiles; 2024 errors: binding modifiers not allowed
    // [(ref mut x, mut y)] => ..
    [(x, y)] => { *x += *y; }          // x is &mut i32 already
}
match &v[..] {
    // 2024 errors: & pattern after binding mode left move
    // &[ref z] => ..
    [z] => ..                          // z: &i32 via binding mode
}
```

Fix: drop the modifiers (the binding mode already gives the right
reference), or match by value/deref to keep the mode `move`.

## static mut references

```rust
static mut BUF: [u8; 16] = [0; 16];

// 2024 deny-by-default error:
let p = unsafe { &mut BUF };

// Use a cell wrapper:
static BUF: SyncUnsafeCell<[u8; 16]> = SyncUnsafeCell::new([0; 16]);
let p = unsafe { &mut *BUF.get() };

// or take a raw pointer without forming a reference:
let p = unsafe { addr_of_mut!(BUF) };
```

`static mut` itself still compiles (deprecation is a separate lint);
what is denied is forming a reference to it.

## unsafe extern blocks

```rust
// 2021
extern "C" { fn libc_fn(x: i32) -> i32; }
// 2024
unsafe extern "C" { fn libc_fn(x: i32) -> i32; }
```

Applies to every `extern "ABI" { }` block. `cargo fix --edition`
rewrites these automatically.

## Unsafe attributes

```rust
#[no_mangle]                 // 2021
#[unsafe(no_mangle)]         // 2024
#[unsafe(export_name = "f")] // 2024
#[unsafe(link_section = ".text")]
```

Affected: `no_mangle`, `export_name`, `link_section`. Auto-fixed by
`cargo fix --edition`.

## unsafe_op_in_unsafe_fn

Now warns by default in 2024 (previously allow):

```rust
unsafe fn f(p: *mut i32) {
    *p = 1;                  // warning in 2024
    unsafe { *p = 1; }       // correct
}
```

## gen keyword

`gen` is a reserved keyword in 2024. Existing `gen` bindings, fields,
or fn names need raw identifiers: `r#gen`. The point is to free the
name for gen blocks / generators, which are NOT stable as of Rust
1.99. Do not write `gen { .. }` expecting iterators on stable.

## Reserved syntax

`#"foo"#` without a prefix and sequences of 2+ `#` are reserved. Only
matters for macro matchers and code generators that emit tokens.

## Macro fragment specifiers

- `expr` fragments now also match `const { }` blocks and `_`
  placeholder expressions. If a macro arm must NOT match those, switch
  it to `expr_2021`.
- `$x` with no `:specifier` is a hard error (was
  `missing_fragment_specifier` lint): `macro_rules! m { ($x) => {} }`
  fails; write `$x:expr` or the intended specifier.

## Prelude additions

`std::future::Future` and `std::future::IntoFuture` are in the 2024
prelude. Breakage mode: an in-scope trait or type named `Future`, or a
foreign trait method `.into_future()`, becomes ambiguous. Fix by
disambiguating with fully qualified syntax (`<T as Trait>::method(x)`)
or renaming the local item.

## IntoIterator for Box<[T]>

`Box<[T]>` implements `IntoIterator` on all editions, but the method
call is edition-dependent:

```rust
let b: Box<[i32]> = vec![1, 2].into_boxed_slice();
// <= 2021: iterates &i32 over the slice (autoref)
// 2024: iterates i32 by value (moves out of the box)
for x in b.into_iter() { .. }
```

Silent behavior change: items go from `&T` to `T`. The compiler warns
via `boxed_slice_into_iter` / compatibility lints during migration.

## Newly unsafe functions

```rust
std::env::set_var("K", "v");            // unsafe fn in 2024
std::env::remove_var("K");              // unsafe fn in 2024
cmd.before_exec(|| { .. });             // unsafe fn in 2024
```

Wrap calls in `unsafe { }` and uphold the contract: do not mutate the
process environment while other threads may read it (getenv races are
a real CVE pattern). Prefer passing env through `Command::env` for
child processes.

## Never type fallback

Never-to-any coercion fallback changed from `()` to `!`. As of Rust
1.100 (beta as of October 2026) this applies on every edition and the
`dependency_on_unit_never_type_fallback` migration lint was removed.
In practice: code like `let x = return;` at tail position that relied
on inferring `()` now infers `!`; annotate the binding if a type
mismatch appears.

## Cargo.toml renames

Removed spellings in 2024 packages:

```
[project]            -> [package]
default_features     -> default-features
crate_type           -> crate-type
proc_macro           -> proc-macro
dev_dependencies     -> dev-dependencies
build_dependencies   -> build-dependencies
```

Plus: `edition = "2024"` implies `resolver = "3"` (rust-version aware
resolution), and since Rust 1.99 an inherited dep's `default-features`
in a 2024 package overrides the `[workspace.dependencies]` value.

## Rustdoc

- All doctests in a crate compile into ONE binary and run as tests of
  that binary. Faster, but doctests relying on isolated binaries,
  crate-level globals, or panic-abort behavior may need `standalone`
  attributes or restructuring. `no_run` doctests still compile but are
  skipped at run time.
- A doctest pulled in via `include_str!` resolves its own nested
  `include!`/`include_str!`/`include_bytes!` relative to the Markdown
  file, not the Rust source file.

## Rustfmt

- `style_edition` in rustfmt.toml selects the style set independently
  of `edition`. `style_edition = "2024"` enables the new style.
- Raw identifiers sort by the name without the `r#` prefix; `use`
  sorting of `Version`-like items changed. Run `cargo fmt` in its own
  commit when adopting the new style to keep blame clean.

## Migration checklist

```
git status           # must be clean for cargo fix
cargo fix --edition  # applies rust-2024-compatibility fixes
# set edition = "2024" in Cargo.toml
cargo build
cargo test
cargo fix --edition --clippy   # optional, extra lint suggestions
```

Then manually audit: lock/guard temporaries in tail and if-let
positions, `Box<[T]>::into_iter()` call sites, `static mut` users, and
any custom `Future`/`IntoFuture` name collisions.
