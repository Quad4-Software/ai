# Advisory and benchmark verification

## CVE sources disagree constantly

NVD and GHSA independently score the same CVEs and frequently
conflict: thousands of scoring mismatches (average drift over 4 CVSS
points), CVSS 3.1 vs 4.0 rescoring differences, patched-version
ranges that differ between repo and global advisories, and CVEs NVD
rejected that GHSA still lists as active. Resolution order:

1. Vendor or repo advisory (GHSA on the repo, project security page)
 for affected-version truth.
2. CISA KEV for confirmed exploitation - authoritative and binary.
3. OSV.dev for package-level aggregation across ecosystems.
4. NVD for the canonical description, cross-checked.
5. EPSS only as a forecast. It is not exploit evidence: ransomware-
 used CVEs have scored ~2% EPSS.
6. CVSS measures technical severity, not exploit likelihood or your
 environment's risk.

## Vendor benchmarks

Before trusting a benchmark number:

- Who paid for it, who wrote the test items, who wrote the answer
 key, was the harness public, was the model/tool frozen before or
 after failure review.
- A credible page names the benchmark version, task subset,
 evaluation harness, and effort/thinking settings.
- Asymmetric baselines and pre-configured environments are the
 classic sponsored-benchmark tricks.

## Statistics literacy

- "Up to X%" is a best-case bound. Ask for the base rate and the
 denominator.
- Vulnerability statistics carry selection bias (what gets looked
 at gets found). Detection-rate claims without a base rate commit
 the base-rate fallacy.
- "Projects using X report Y" claims ignore survivorship.
- Security report counts (packages affected, repos touched) vary by
 counting method. Cite the tracker and its number, not a rounded
 blend.

## Conflicts and unverified claims

- Quote both sides with dates instead of picking one: "NVD scores
 9.8 under CVSS 3.1. GHSA lists High under CVSS 4.0, checked
 <date>".
- Label anything unverified as UNVERIFIED. Never upgrade a claim
 during summarization.
