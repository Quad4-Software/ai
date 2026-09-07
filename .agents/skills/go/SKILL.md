---
name: go
description: >
  This skill covers the Go toolchain and conventions used in this repo.
  Use when you are writing Go, choosing versions, vendoring offline, or
  working with TinyGo or the legacy Windows 7 fork.
compatibility: go-1.27
---

## When to use this skill

- You need to know which Go version or toolchain this repo uses.
- You are deciding between vendored, offline, or local `third_party/` builds.
- You are writing code for TinyGo, embedded targets, or legacy Windows.

## How to use

1. Check `AGENTS.md` and each `go.mod` for the toolchain.
2. Run `make all` from the repo root to format, vet, test, and build all `mcp/*` servers.
3. Use `go mod vendor` or `GOFLAGS=-mod=vendor` for offline or air-gapped builds.
4. See the `mcp-toolkit` skill for repo-wide build and module conventions.

## Examples

- "Build all servers with `make all` and `make gosec`."
- "Vendor dependencies for offline use with `go mod vendor`."
- "Check the `micron` vendored parser under `third_party/`."

# Go

This skill covers the Go toolchain versions used in this repo, offline and
air-gapped build workflows, TinyGo for embedded and WebAssembly, and the
legacy Windows 7 fork.

## Go 1.27

Go 1.27 arrived in August 2026. Most changes are in the language, toolchain,
runtime and libraries. The Go 1 compatibility promise is preserved.

### Language

- **Generic methods** are now supported. A method declaration may declare its
  own type parameters. Example from `math/rand/v2`:

  ```
  func (r *Rand) N[Int intType](x Int) Int
  ```

  Interface methods cannot declare type parameters and cannot be implemented by
  generic methods.

- **Struct literal keys** may now be any valid field selector, including nested
  or embedded fields.

- **Function type inference** now applies in all assignment contexts: composite
  literals, type conversions and channel sends.

### Tools

- Response file (`@file`) parsing is supported for `compile`, `link`, `asm`,
  `cgo`, `cover` and `pack`.
- `go doc` supports `package@version`, such as `go doc example.com/pkg@v1.2.3`.
- `go doc -ex` lists executable examples.
- `go fix` adds `atomictypes`, `embedlit`, `slicesbackward` and `unsafefuncs`
  modernizers.
- `go mod tidy` consolidates `require` blocks into one direct and one indirect
  block for `go 1.27` modules.
- `go test` runs the `stdversion` vet check by default.

### Runtime and standard library

- **Faster small allocations**: specialized routines for objects under 80 bytes
  reduce their allocation cost by up to 30%, with about 1% overall improvement
  in allocation-heavy programs. Binaries grow by about 60 KB. Disable with
  `GOEXPERIMENT=nosizespecializedmalloc` if needed.
- **Goroutine leak profile**: `runtime/pprof` now supports the `goroutineleak`
  profile type, also exposed at `/debug/pprof/goroutineleak`.
- `encoding/json/v2` and `encoding/json/jsontext` provide a new JSON API.
- `crypto/mldsa` implements the ML-DSA post-quantum signature scheme.
- `crypto/tls` adds MLKEM1024 and ML-DSA for TLS 1.3.
- `uuid` is added to the standard library.

## Go 1.26

Go 1.26 shipped in February 2026. It added new packages and security features.

### New packages

- `crypto/hpke` and `crypto/mlkem/mlkemtest`
- `testing/cryptotest`
- `simd/archsimd` (experimental)
- `runtime/secret` (experimental, `GOEXPERIMENT=runtimesecret`) for securely
  erasing temporary data in cryptographic code

### Methods and runtime

- `bytes.Buffer.Peek` returns the next n bytes without advancing.
- `crypto/ecdh` and `crypto/ecdsa` now ignore the user-supplied random reader and
  always use a CSPRNG. Use `testing/cryptotest.SetGlobalRandom` for
  deterministic tests. Re-enable the old behavior with `cryptocustomrand=1`.
- `sync.WaitGroup.Go` runs a function in a new goroutine and calls `Done`.

### Security

- **Green Tea garbage collector** is now the default (see below).
- 64-bit platforms now randomize the heap base address at startup, making cgo
  exploits harder. Opt out with `GOEXPERIMENT=norandomizedheapbase64`.
- Point releases 1.26.1 through 1.26.6 fixed many security issues, including:
  - `crypto/x509` name and email constraints
  - `html/template` meta content URL escaping
  - `net/url` IPv6 literal validation
  - `os` FileInfo escaping from a root
  - `x/mod/sumdb/tlog` transparency log verification bypass (CVE-2026-56865)
  - `x/mod/sumdb` unauthenticated hash lookup (CVE-2026-56864)
  - `encoding/xml` recursion depth exhaustion
  - `net/http` and `net/http/httputil` issues

## Green Tea garbage collector

Green Tea is the first major GC redesign since Go 1.5. It keeps the concurrent
mark-sweep foundation but improves marking and scanning of small objects through
better locality and CPU scalability.

- Became the default in **Go 1.26**.
- Uses 8 KiB spans instead of tracking individual objects for queuing work.
- Leverages vector instructions on newer amd64 CPUs (Intel Ice Lake, AMD Zen 4
  and newer) for another ~10% improvement.
- Expected 10 to 40% reduction in GC overhead for many workloads.
- In Go 1.26, opt out with `GOEXPERIMENT=nogreenteagc`. In Go 1.27 the opt-out
  is removed and Green Tea is the only collector.

## Go vendoring for offline and air-gapped builds

The Go toolchain normally fetches modules from a proxy. For offline or
reproducible builds, vendor the modules into the repo and build from the
`vendor/` directory.

### Create the vendor directory

Run this with network access:

```
go mod vendor
```

This creates `vendor/` and `vendor/modules.txt` from `go.mod`, except for
modules replaced by local paths.

### Build from vendor

```
go build -mod=vendor ./...
go test -mod=vendor ./...
```

Make this the default with `GOFLAGS`:

```
export GOFLAGS=-mod=vendor
```

### Force offline builds

```
GOPROXY=off GOFLAGS=-mod=vendor go build ./...
```

`GOPROXY=off` ensures any missing module fails the build instead of fetching.

### Air-gapped workflow

1. On a networked machine, run `go mod download` or `go mod vendor`
2. Copy the repo, including `vendor/`, into the air-gapped network
3. Run `GOPROXY=off GOFLAGS=-mod=vendor make all` or `go test ./...`
4. Do not run `go mod tidy` or `go get` inside the air gap

### Local third_party modules

For modules that cannot be fetched from a proxy, use a local `third_party/`
directory and a `replace` directive:

```
require micron-parser-go v1.1.0
replace micron-parser-go => ./third_party/micron-parser-go
```

Run `go mod vendor` if needed, then commit the source and the `go.mod` change.

### License and reproducibility

- `go mod vendor` copies full module source, including license files
- Keep `vendor/` in version control for fully reproducible builds
- Audit `vendor/` licenses before distribution
- Keep `go.mod` and `go.sum` in sync before vendoring
- Pin `go` and `toolchain` versions in `go.mod`

## TinyGo

TinyGo is a Go compiler for small places: microcontrollers, WebAssembly and
command-line tools. It uses LLVM. See https://tinygo.org/ and
https://github.com/tinygo-org/tinygo.

### TinyGo 0.42 (September 2026)

- Go 1.27 support and generic methods
- Moved to LLVM 22
- **Recoverable panics**: nil pointer, divide by zero, out-of-range, channel
  panics are now recoverable through `defer` and `recover`. Out-of-memory and
  other fatal errors remain unrecoverable.
- `testing` now supports `Goexit`, `SkipNow` and `FailNow`, so `t.Skip` and
  `t.Fatal` behave correctly.
- New **UEFI target**: compile and run Go programs as UEFI applications.
- ESP32: original ESP32 gains interrupts, timer alarms, GPIO `SetInterrupt`,
  ADC, UART, flash XIP and WiFi. Bluetooth on ESP32-C3 and ESP32-S3.
- STM32: OTG FS USB driver for F4/F7, STM32H7 and NUCLEO-H753ZI support,
  STM32U031, and UART fixes.
- Puya PY32F microcontrollers, Pimoroni Blinky 2350 and Badger 2350, Seeed
  Studio XIAO nRF52840 Plus, Game Boy Advance mGBA debugging.
- Dynamic USB endpoints and descriptors on RP2, SAMD21, SAMD51 and nRF52840.
- New `USBDevice.Attach` and `USBDevice.Detach` methods.

Use `tinygo build`, `tinygo flash` and `tinygo test` for targets.

## Legacy Windows support

The `thongtech/go-legacy-win7` fork keeps Go running on legacy Windows systems:

- Supports Windows 7, 8, 8.1, Server 2008 R2, Server 2012 and Server 2012 R2
- Restores classic `go get` behaviour for `GO111MODULE=off` or `auto`
- Based on Go 1.27.0 as of release `go-legacy-win7-1.27.0-2`
- Reverts changes that break older Windows, including `RtlGenRandom` /
  `ProcessPrng`, `LoadLibraryA`, socket syscalls, and the race detector

See https://github.com/thongtech/go-legacy-win7 for releases and install
instructions. This fork is not official; use it only when you must support
legacy Windows systems.
