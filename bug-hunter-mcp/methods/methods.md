# Bug hunting methods

Each method lists when to use it, how to run it, and what a real result looks like.
The shared discipline: state a hypothesis before touching code, and confirm with an
oracle that accepts or rejects independently of the buggy path.

## Exploratory testing (chartered sessions)

Time-boxed hunt with an explicit charter: target area, oracles, and what "done" means.
Write 5 to 15 concrete hypotheses (Hn) with file references and predicted wrong
behaviour before running anything. Bad hypothesis: "maybe chat is broken". Good:
"PART from a non-member fans PARTED to real members (server.py _handle_part)".
Confirm each hypothesis with a focused test before fixing. Record intentional
behaviours separately so they are not "fixed" by accident.

## Oracle testing

An oracle predicts the correct outcome from the input alone or from a trusted model.
Oracle types: accept/reject on invalid input, parse round-trips, jail invariants
(result stays under root), closed error-reason sets, membership invariants
(no fanout to non-members). Refuse soft fuzz: tests that only assert no crash, wrap
the unit in try/except pass, or mock the security check away.

## Property-based testing

State an invariant in one sentence, then generate inputs (Hypothesis for Python,
fast-check for JS, testing/quick or go fuzz for Go). Strong properties: round-trip
(decode(encode(x)) == x), invariants under reordering, idempotence, bounds
(output never exceeds cap), equivalence to a simple reference implementation.
Set a deadline cap and an example budget, shrink to the minimal failing case.

## Metamorphic testing

For code with no obvious oracle, define metamorphic relations: how output should
change when input changes in a known way. Examples: sorting the input must not
change the result set, doubling a value must double the measured cost, removing
a permission must never widen access. Pairs of related runs expose bugs single
runs cannot.

## Differential testing

Run two implementations of the same spec against identical inputs and diff outputs:
old vs new parser, native vs reference decoder, live vs replayed traffic.
Disagreement is a bug in one of them, investigate before assuming which.

## TOCTOU and race hunting

Look for check-then-use pairs: exists/stat/access followed by open/read/write on the
same path, permission check separated from the action it guards, cache read then
trust of a stale entry. The exploitable gap is between check and use. Confirm by
swapping the checked object between the two operations or forcing a retry.
For concurrency: shared mutable state without a lock, map iteration during writes,
goroutine captures of loop variables, send-on-closed-channel. Run with -race or
the sanitizer equivalent, races found under stress are still races.

## Churn and hotspot analysis

Bug probability correlates with change frequency and file size: files that change
often AND are large accumulate defects (Microsoft bug-cache research, defects cluster
in the most-edited 20% of files). Rank files by commits in a window weighted by
recency, then hunt in the top files first. Cross-reference with TODO density,
complexity, and recent security-surface edits.

## Error-path injection

Most bugs live on paths nobody exercised: force each failure mode (permission denied,
timeout, partial write, corrupt record mid-file, empty input, oversized input).
Check that errors propagate with context, that partial state is rolled back, and
that a retried operation is idempotent.

## Boundary analysis

For every numeric bound or size cap: test exactly at, one below, one above, zero,
negative, and maximum. For strings: empty, whitespace-only, overlong, unicode
edge cases, format-string markers, path traversal segments. For time: expiry at
exactly now, clock skew, timezone and DST edges, year-2038 fields.

## Combinatorial testing

When a feature takes several independent options, exhaustive coverage explodes but
pairwise coverage (all pairs of option values) catches most interaction bugs with a
small case set. Generate the pairwise matrix, run each row once, check invariants.

## State machine testing

Extract the real state machine from code: states, transitions, guards. Then test
illegal transitions (each one must be rejected or handled), re-entrant transitions
(event during its own handler), and persistence round-trips (save, reload, resume
mid-state). Membership and session bugs are usually illegal-transition bugs.

## Attack surface mapping

Enumerate every mutating entry point: HTTP POST/PUT/DELETE routes, WS message types
that write state, file writes, exec calls, plugin invokes. For each: auth required,
CSRF/allowlist applied, input validated, output bounded. Any mutator missing one of
those is a candidate finding.

## Regression mining

Past bugs predict future ones. git log for fix/hotfix/revert commits, then read the
fix and ask whether the same pattern exists elsewhere. A fix in one route with three
siblings often means three unreported bugs.
