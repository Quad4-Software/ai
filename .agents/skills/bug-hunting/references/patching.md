# Full patches, not half patches

A half patch fixes the reported instance and leaves the bug class, the
sibling call sites, or the error paths open. It is the best-documented
failure mode of both automated program repair and LLM code
generation, and it is just as common in human security fixes.

## The evidence

- FLAWED (1Password, Aug 2026): across frontier-model patch runs, ~26%
 cleanly fixed the vuln, ~49% did not fix it at all. Failure
 descriptions: "addressed only a subset of vulnerable code paths" and
 "fragile guard code that satisfied tests while failing to address
 the root cause."
- Vul4J study (arXiv 2603.10072): 24.8% of LLM security patches fully
 correct. Dominant failure is wrong repair strategy, not syntax.
 Input-validation fixes: 0% success.
- SWE-bench correctness audit (ICSE 2026): 29.6% of "resolved"
 patches diverge from the gold patch's behavior. Benchmark tests
 accept half patches. Tests passing is not correctness.
- Firefox CVE study (DIVA 2025): 45% of LLM patches insufficient -
 incomplete fixes or new issues introduced.
- Human precedent: Shellshock needed five CVEs before the class was
 fixed (BASH_FUNC_* namespacing). Log4Shell needed 2.15, 2.16, 2.17.
 RegreSSHion (CVE-2024-6387) was a regression of a fix to
 CVE-2006-5051 whose earlier fix (CVE-2008-4109) was itself
 incorrect. Project Zero: ~half of in-the-wild 0days in 2022 were
 variants of already-patched bugs.

## Why half patches happen

- Test-that-passes bias: the reward signal is green tests, so the fix
 targets the failing assertion, not the root cause. Frontier agents
 have been observed calling exit(0) or skipping tests to "pass".
- Symptom fixing: the diff touches the reported line but never traces
 the data path end to end.
- No call-graph view: callers, interface implementors, string- or
 reflection-dispatched call sites, and config files are invisible to
 text-level search.
- Context limits: long-context recall is worst in the middle, which
 is exactly where "does this bug exist elsewhere" lives.
- Action bias: agents that start patching rarely stop to ask whether
 the patch should be bigger, and report "all fixed" without
 evidence.

## The discipline

Project Zero's framing: a correct patch fixes the bug with complete
accuracy. A comprehensive patch applies that fix everywhere it needs
to be applied, covering all variants. Both halves are required.

1. Reproduce the bug before patching. Agents that skip this fix a
 guess.
2. State the root cause in one sentence: "[untrusted data] reaches
 [dangerous op] without [protection]". If you cannot, you are not
 ready to write the fix.
3. Variant analysis: extract the invariant, match the known instance
 exactly, then generalize the pattern one element at a time up the
 abstraction ladder (exact, variable, structural, taint). Run it
 with grep, opengrep, or CodeQL across the whole repo and list every
 hit. Check sibling components, copy-pasted idioms, other format
 handlers, and the patch itself for new reachable sinks.
4. Call-site audit: every caller, callee, implementor, string dispatch,
 config, fixture, and doc that touches the changed surface. Resolve
 via LSP or a call graph, not grep alone.
5. Edge cases: empty/null/zero/max/malformed/duplicate/unicode inputs,
 cleanup on every exit path, TOCTOU and signal safety, partial
 failure consistency, timezone/locale.
6. Write a regression test that fails pre-patch and passes post-patch.
 One test per variant, kept permanently.
7. Patch-diff your own fix the way an n-day attacker would. The fix
 commit is a roadmap to the bug. If the class is still open, you
 published the variant.
8. Compatibility is a decision, not an accident: Hyrum's Law says
 every observable behavior is someone's dependency. Break behavior
 deliberately and document it, or preserve it deliberately.
9. No TODOs, stubs, hardcoded returns, skips, or commented-out logic
 in the diff. Watch for unexplained size: LLM patches average ~2x
 the churn of developer patches.
10. Close the loop: a lint or Semgrep/CodeQL rule for the pattern
 prevents the regression. WebKit's Zombie bug regressed after 3
 years. PetitPotam regressed. Fixes regress.

## Sources

- Project Zero Deja vu-lnerability and 0-day RCA template:
 projectzero.google/2021/02/deja-vu-lnerability.html
- FLAWED: 1password.com/files/resources/frontier-models-
 vulnerability-patches-flawed.pdf
- Trail of Bits variant-analysis skill (5-step method):
 github.com/trailofbits/skills
- Anthropic "n-days" on patch diffing as attack roadmap
- Qualys regreSSHion advisory. Red Hat Shellshock write-ups
