---
name: qa
description: >
  This skill covers software QA as of September 2026: test shapes
  (pyramid, trophy, honeycomb), unit/integration/E2E boundaries,
  contract testing with Pact, property-based testing (fast-check,
  Hypothesis, rapid), mutation testing (Stryker, mutmut, mutago,
  gremlins), fuzzing, visual regression, flaky-test management, test
  smells, coverage misuse, risk-based and exploratory testing,
  quality gates, ephemeral environments, chaos engineering,
  performance testing (k6, Gatling), and accessibility (axe-core,
  WCAG 2.2). Use it when designing a test strategy, triaging flaky
  tests, choosing test tooling, or pushing back on coverage targets.
---

## When to use this skill

- You are deciding what kind of tests a change needs.
- A suite is flaky, slow, or full of false confidence.
- You are picking test tooling (contract, property, mutation, E2E).
- Someone is proposing a coverage mandate.

## How to use

1. Read this file for the strategy layer.
2. Pick a test shape per component, not per org (see below).
3. Order effort by risk: probability x impact.

## Examples

- "This E2E suite takes 40 min and flakes weekly - what do we do?"
- "Do we need Pact between these two services?"
- "Is 80% coverage a good gate?"

# Software QA

## Test shapes: pick per component

- **Pyramid** (Cohn/Fowler): many unit, fewer integration, few E2E.
  Right for logic-heavy libraries and monoliths.
- **Trophy** (Kent C. Dodds): static types, unit, wide integration
  band, thin E2E. Right for frontend/component code.
- **Honeycomb** (Spotify): integration-heavy for microservices where
  the seams are the risk.
- Fowler's synthesis: the argument is semantic. Optimize test
  confidence per dollar. Integration tests got cheap (Testcontainers,
  better fakes), which pulled effort upward everywhere.

Definitions that resolve disputes: a unit test exercises a unit with
collaborators replaced. An integration test crosses a process,
network, filesystem, or data boundary against a real collaborator.
E2E drives the deployed system through real surfaces.

## Contract testing

Pact is the reference consumer-driven implementation. Consumers
publish pact files, providers verify them, the Pact Broker stores
contracts, `can-i-deploy` is the release gate. Pact v4 covers HTTP
and async messages. Bi-directional contract testing verifies a
provider's OpenAPI/AsyncAPI doc against consumer contracts without
writing consumer tests. Contract tests replace most cross-service
integration tests, not intra-service integration or E2E smoke.

## Property-based testing

- JS/TS: fast-check v4 (shrinking, model-based state machines).
- Python: Hypothesis (example replay, integrated shrinking).
- Go: `testing/quick` is frozen. Use `pgregory.net/rapid` (generics,
  minimization, `MakeFuzz` converts a property test into a
  `testing.F` fuzz target). gopter is superseded.
- Best on parsers, codecs, round-trip invariants, state machines.

## Mutation testing

Measures whether tests detect faults, not just execute lines. One
test-suite run per mutant, so scope it: critical modules (auth,
money, parsers), nightly or pre-release, never whole-repo per-commit.

- JS/TS: Stryker v10 (Node 22+, Vitest 4 support).
- Python: mutmut.
- Go: go-mutesting lineage is stale. Use quality-gates/mutago or
  gremlins for new work.

## Fuzzing

- Go native fuzzing (`testing.F`) with short CI budgets. This repo
  runs 15s per target in fuzz.yml. Long runs go to scheduled jobs.
- OSS-Fuzz for continuous open-source fuzzing, ClusterFuzzLite for
  private code in CI. AFL++ for compiled/binary targets.
- Fuzz anything touching untrusted input. Commit found crashes as
  regression tests.

## Flaky tests

Treat flakiness as a managed system:

- Detect via pass-on-retry events and per-test flip rate.
- Quarantine fast (same day), keep running non-blocking, attach a
  ~14-day expiry and a named owner. Quarantine without expiry is a
  coverage graveyard. GitLab's quarantine process is the reference.
- Max 1-2 retries. Flag pass-on-retry rather than silently passing.
- Flake budget: suite-level ceiling around 1%. Breaching it
  prioritizes flake fixes over features.
- Root causes: async timing, shared state under parallelism, real
  network calls, CI resource contention, unseeded randomness.

## Coverage

Coverage proves code executed, not that behavior was verified.
Inozemtseva & Holmes (ICSE 2014): coverage correlates weakly with
fault detection once suite size is controlled. Goodhart's law: a
coverage mandate produces tests that visit code without checking it.

Use it for: finding 0% regions, diff coverage on new code as a PR
gate, trending. Do not use it as a single quality score or target.

## Test smells

From Meszaros' xUnit Test Patterns: assertion roulette (no messages),
mystery guest (hidden fixtures), conditional test logic, fragile
tests coupled to internals, slow tests missing fakes, the liar
(passes regardless). Test code gets the same review rigor as
production code.

## Other disciplines

- **Risk-based**: order effort by probability x impact (recent change,
  complexity, criticality, integration seams).
- **Exploratory**: Session-Based Test Management - a written charter
  ("Explore X with Y to discover Z"), a 60-120 min timebox, a session
  report, a debrief. The human complement to automated regression.
- **Quality gates**: gate on deterministic fast checks (lint, unit,
  SAST, secrets) plus new-code coverage. Slow/flaky layers run
  scheduled or pre-release with their own budgets.
- **Environments**: ephemeral per-PR environments beat shared staging.
  vCluster, Signadot, or Argo CD ApplicationSet PR generators.
- **Chaos**: LitmusChaos and Chaos Mesh (both CNCF). Steady-state
  hypothesis tied to SLOs, small blast radius, GameDays first.
- **Performance**: k6 (AGPL, TS support, thresholds gate CI exit
  codes) or Gatling. JMeter is legacy for new work. For Go
  microbenchmarks, benchstat-compare in CI.
- **Accessibility**: axe-core via @axe-core/playwright finds ~57% of
  WCAG issues. Assert zero violations in CI and schedule manual
  audits for the rest. Target WCAG 2.2.
- **Visual regression**: Playwright `toHaveScreenshot` with masking
  and `maxDiffPixelRatio`. Freeze animations, clocks, and fonts to
  kill flake.

## This repo

`make test`, `go test ./...`, `go test -race` (race.yml), native
fuzzing (fuzz.yml), goroutine leak checks (leak.yml), microbenchmarks
(bench.yml). Tests must run offline. Use fakes plus fuzz targets
rather than Testcontainers. See `ci-security` for the gate layer.
