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
old vs new parser, native vs reference decoder, live vs replayed traffic, vendored
copy vs upstream release tarball. Disagreement is a bug in one of them, investigate
before assuming which. For supply chain: diff the published artifact against the
tagged source, an XZ-style backdoor hides in the gap.

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
timeout, partial write, corrupt record mid-file, empty input, oversized input,
cancelled context, closed channel, DNS failure mid-request). Check that errors
propagate with context (Go %w not %v so errors.Is works), that partial state is
rolled back, and that a retried operation is idempotent.

## Boundary analysis

For every numeric bound or size cap: test exactly at, one below, one above, zero,
negative, and maximum. For strings: empty, whitespace-only, overlong, unicode
edge cases, format-string markers, path traversal segments. For time: expiry at
exactly now, clock skew, timezone and DST edges, year-2038 fields. For integers:
values that overflow when widened or multiplied before an allocation.

## Combinatorial testing

When a feature takes several independent options, exhaustive coverage explodes but
pairwise coverage (all pairs of option values) catches most interaction bugs with a
small case set. Generate the pairwise matrix, run each row once, check invariants.

## State machine testing

Extract the real state machine from code: states, transitions, guards. Then test
illegal transitions (each one must be rejected or handled), re-entrant transitions
(event during its own handler), and persistence round-trips (save, reload, resume
mid-state). Membership and session bugs are usually illegal-transition bugs.
Session fixation, token reuse after logout, and upgrade-then-downgrade flows are
all illegal-transition families.

## Attack surface mapping

Enumerate every mutating entry point: HTTP POST/PUT/DELETE routes, WS message types
that write state, file writes, exec calls, plugin invokes, MCP tool handlers,
deserializers, webhook receivers. For each: auth required, CSRF/allowlist applied,
input validated, output bounded, rate limited. Any mutator missing one of those is
a candidate finding.

## Regression mining

Past bugs predict future ones. git log for fix/hotfix/revert commits, then read the
fix and ask whether the same pattern exists elsewhere. A fix in one route with three
siblings often means three unreported bugs. Also mine reverted commits: whatever the
revert tried to fix may still be broken upstream of the revert.

## Injection and taint tracing

Pick a source (request field, argv, env var, file contents, MCP tool argument) and
trace it to a sink (exec, SQL, template render, file path, redirect, HTTP request).
Sanitizers in the middle must be output-context correct: escaping for SQL does not
help HTML, escaping for HTML does not help a shell. Confirm with a payload that
survives the sanitizer: quote breakout for SQL, ; or $() for shells, ../ for paths,
{{7*7}} for templates. Naive intra-file taint pairs are a starting list, not proof:
cross-function flow still needs manual tracing.

## Secrets and credential hunting

Search for token-shaped literals (AKIA, ghp_, sk-, xoxb, private key blocks, JWTs)
and key-shaped assignments (password=, api_key=, secret=). Check .env files, config
snapshots, test fixtures, and git history: a secret committed once and later removed
is still leaked. Every finding is already compromised; report, rotate, then purge.
Never print the full value in a report, show enough to identify it.

## Crypto misuse

The recurring hits: md5/sha1 for integrity or passwords, des/rc4/ecb anywhere,
math/rand or Math.random for tokens and keys, InsecureSkipVerify / verify=False /
rejectUnauthorized:false in TLS, hs256-vs-rs256 algorithm confusion in JWT
verification, == comparison of secrets instead of constant-time compare, hardcoded
IVs and salts. Rule of thumb: if the primitive choice came from a tutorial or an AI
completion, assume it is wrong until checked.

## Concurrency review

Beyond data races: goroutine and connection leaks (spawned workers with no cancel,
response bodies never closed, channels nobody drains), WaitGroup.Add inside the
goroutine it guards, defer inside a loop pinning resources until return, loop
variable capture (Go <1.22 shares the variable, >=1.22 still shares captured
pointers), close() followed by a send. Confirm with -race under stress and by
counting goroutines/fds before and after a load burst.

## Supply chain audit

Audit what the build pulls in, not just the code in the tree. Check for: lifecycle
scripts in package.json and setup.py (preinstall/postinstall/cmdclass), deps pinned
by name only with no lockfile or hash, git+ and http: dependency specs, go.mod
replace directives, unpinned GitHub Actions (a tag like @v5 is mutable; only a full
40-char SHA pins), curl|sh installers, base64 blobs or eval in packaging code.
Known-shape incidents to pattern-match: typosquats (one-char edits of popular
names), slopsquats (AI-hallucinated package names an attacker then registers),
maintainer-account takeover (npm qix/phishing campaign, Sept 2025), tag rewriting
(tj-actions/changed-files, March 2025, CVE-2025-30066), tarball-vs-git divergence
(XZ Utils, CVE-2024-3094), cached typosquats in module proxies (boltdb-go/bolt).
Verify provenance where offered: cosign/Sigstore signatures, SLSA provenance,
reproducible builds.

## CI/CD pipeline review

Treat workflows as code that runs attacker-influenced input. Flag: triggers that run
on untrusted input (pull_request_target, workflow_run, issue_comment) combined with
secrets access or checkout of the PR head; script injection through
${{ github.event.* }} interpolation into run: blocks; unpinned or third-party
actions; overly broad GITHUB_TOKEN permissions; artifacts and caches shared between
untrusted and trusted jobs. The tj-actions incident showed tags are mutable, pin by
SHA and review workflow diffs like dependency diffs.

## MCP and agent-tool security

MCP tool descriptions are executable instructions to a model, treat them as an
attack surface. Tool poisoning: a description carrying hidden directives (read this
file, send it here, ignore other rules) that the client LLM obeys without the user
seeing it. Rug pulls: a server that changes tool definitions after approval. Exec
sinks: tool handlers that pass caller-controlled strings to a shell. Read tools can
still leak: arbitrary path reads, unbounded output. Audit every server you run:
read its source, pin its version, check its tool descriptions for imperative
language aimed at the model rather than documentation for the user.

## Authorization matrix

Build the actor-by-action matrix from code, not intent: which roles can call which
mutators, read which objects, cross which tenant boundaries. Then test the missing
cells: every combination of role x object-owner x action should have a decision, and
every decision should be enforced at the handler, not assumed from a front-end gate.
IDOR and privilege escalation are almost always an absent cell, not a broken check.
Compare sibling handlers: if nine of ten routes check ownership, the tenth is the bug.

## AI-generated code review

Assume generated code carries generated flaws: hallucinated dependencies (verify
every import resolves to a real, intended package before installing), insecure
defaults it was trained on (verify=False, md5, shell=True, pickle), plausible-but-
wrong API usage, and missing error paths. Review generated diffs with the same
suspicion as a drive-by PR: the author did not verify it either.
