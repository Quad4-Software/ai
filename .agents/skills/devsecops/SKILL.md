---
name: devsecops
description: >
  This skill covers DevSecOps as of September 2026: where security
  controls sit in the pipeline (pre-commit through runtime), SLSA
  v1.2 and in-toto attestation, Sigstore/cosign keyless signing,
  SBOM generation and continuous scanning (CycloneDX, SPDX, cdxgen,
  Syft, Grype, Dependency-Track), policy-as-code admission (Kyverno
  graduated, OPA/Gatekeeper, ValidatingAdmissionPolicy), dependency
  review and secret scanning, SAST/DAST placement (ZAP post-OWASP),
  CI-pipeline static analysis (zizmor), security champions, threat
  modeling (Threat Dragon), compliance-as-code (OpenSCAP, OSCAL),
  runtime detection (Falco), and vulnerability triage with KEV,
  EPSS v4, and SSVC. Use it when wiring security gates, signing and
  attesting artifacts, triaging CVEs, or assessing program maturity
  with DSOMM.
---

## When to use this skill

- You are placing a security control in a pipeline (SAST, SCA,
  secrets, IaC, DAST, admission).
- You are signing artifacts, generating SBOMs, or emitting
  provenance.
- You are triaging a CVE or deciding what to gate on.
- You are assessing a program's DevSecOps maturity.

## How to use

1. Read this file for control placement and tool choices.
2. For scanner catalogs see `sast`. For GitHub Actions hardening see
   `ci-security`. For OWASP category lists see `owasp`.
3. Version notes are pinned to September 2026. Verify before quoting.

## Examples

- "Wire cosign keyless signing into our release workflow."
- "Which findings should block the merge vs warn?"
- "How do we triage 400 CVEs across our images?"

# DevSecOps

## Where controls sit

1. Pre-commit: secrets (gitleaks/trufflehog), lint, fast tests.
2. PR gate, fast and deterministic: diff-aware SAST, dependency
   review on the lockfile diff, secrets diff-scan, IaC scan on
   changed modules, license policy.
3. Build: hardening flags, SBOM generation, keyless signing, SLSA
   provenance attestation.
4. Pre-deploy admission: policy-as-code verification of signatures,
   provenance, base-image policy.
5. Post-deploy: DAST against ephemeral/staging envs, scheduled
   full-history secret scans, continuous re-scanning of shipped
   SBOMs (new CVEs hit old builds).
6. Runtime: Falco-style detection feeding triage and new policy.

Gate only on high-confidence, high-severity, actionable findings
(secrets, KEV-listed vulns, critical SAST with low false-positive
rate). Everything else warns with a tracked SLA. Gating on noisy
scanners trains bypass behavior. Baseline-and-ratchet: document
existing findings, block only new ones.

## Provenance and signing

- SLSA v1.2 (Nov 2025): adds the Source track alongside Build
  L1-L3. Dependency and Build Environment tracks in development.
- in-toto (CNCF graduated Apr 2025): Attestation Framework v1 adds
  the bundle layer - one signed envelope for provenance plus SBOM
  plus scan results. DSSE is the recommended envelope.
- Sigstore/cosign keyless is the default over long-lived keys:
  OIDC identity -> Fulcio short-lived cert -> Rekor transparency
  log. In CI use ambient OIDC (`id-token: write`) and `--yes`.
  For files: `cosign sign-blob --bundle b.sigstore.json <file>`,
  verify with `--certificate-identity` + `--certificate-oidc-issuer`.

## SBOM workflow

- Formats: CycloneDX 1.6/1.7 (security-oriented, VEX-native) and
  SPDX 2.3/3.0.1 (license-oriented, ISO 5962). Generate both if
  consumers differ.
- Generate at build time via build-tool plugins (cdxgen,
  cyclonedx-gomod) for the true resolved graph. Scan the final
  image with Syft for OS packages. Best practice is both.
- Store SBOMs as signed attestations, feed Grype for scanning and
  Dependency-Track for inventory. Re-scan shipped SBOMs
  continuously.
- Publish VEX statements with SBOMs so consumers stop filing the same
  unreachable CVE. OpenVEX (openvex.dev, OpenSSF project) is the
  minimal JSON format: product, vulnerability, status
  (`not_affected`/`affected`/`fixed`/`under_investigation`) plus
  justification (`vulnerable_code_not_in_execute_path` etc.). CycloneDX
  embeds VEX natively in `vulnerabilities[].analysis`. Tooling:
  `vexctl` generates and merges OpenVEX docs, `grype --vex doc.json`
  suppresses matching findings, Trivy accepts VEX via `--vex`, and
  Dependency-Track ingests them for triage. The trust rule: only sign
  VEX you produced, and treat third-party `not_affected` claims as
  UNVERIFIED until audited.
- Caution: Trivy was reportedly compromised in a March 2026
  supply-chain window. Verify current status before pinning it into
  a pipeline.

## Admission policy

- Kyverno (CNCF graduated Mar 2026): `verifyImages` for cosign
  signature/attestation checks at admission, resource generation,
  PolicyReports as audit evidence. Strategic direction is the
  CEL-based `policies.kyverno.io/v1` types.
- OPA/Gatekeeper: Rego covers K8s plus non-K8s policy (Terraform
  plans, CI gates). Broader surface, steeper curve.
- K8s built-in ValidatingAdmissionPolicy (GA since 1.30) covers
  simple cases with no controller. You still need a controller for
  image verification, generation, and audit reports.

## Dependency and secret scanning

- Dependency review on PRs (this repo uses
  actions/dependency-review-action with `fail-on-severity:
  moderate`) plus Dependabot or Renovate with cooldown periods.
  Recent incidents argue for cooling down fresh releases before
  adopting them.
- Secret scanning is layered: platform push protection always on,
  gitleaks in pre-commit and CI diff-scan, TruffleHog for periodic
  deep history scans (live-credential verification separates real
  leaks from noise).
- Licensing notes: gitleaks is feature-complete (security patches
  only) with changed org terms at v8.19. TruffleHog is AGPL-3.0.

## Scanner placement

- SAST: language-native first (gosec for Go), Semgrep/OpenGrep for
  polyglot, CodeQL for depth. See `sast` for the catalog.
- DAST: ZAP is no longer OWASP. It is "ZAP by Checkmarx" (2.17.0,
  Apache-2.0, active). Nuclei complements it. Run against ephemeral
  envs, warn first, graduate to gates.
- IaC: Checkov, Trivy IaC, KICS on changed modules at PR time.
- Pipeline self-analysis: zizmor audits the workflow files
  themselves. This repo gates on it. Most guides still miss this
  category.

## Program layer

- Security champions: OWASP Security Champions Guide. Nominated not
  assigned, defined time commitment, outcome metrics. BSIMM15 shows
  adoption tracks maturity.
- Threat modeling: OWASP Threat Dragon v2.x (Production status)
  with STRIDE-per-element and the Threat Modeling Manifesto.
- Compliance-as-code: OpenSCAP maintained but NIST SCAP Validation
  ended Sep 2025, so do not claim NIST validation. Content lives in
  ComplianceAsCode/content. OSCAL and compliance-trestle are the
  modern complement. Chef InSpec now needs a Progress license.
- Maturity assessment: OWASP DSOMM (five dimensions, YAML evidence
  tracking) pairs with SAMM for the program-level view.

## Runtime detection

- Falco (CNCF graduated): eBPF/kmod syscall detection plus plugins.
  Falcosidekick routes alerts to Slack/SIEM/webhooks. Falco
  Captures writes SCAP forensic files on rule trigger.
- The feedback loop: runtime alerts -> triage -> root cause -> new
  detection rules, tests, or admission policies. This is the
  shift-right half of the model.

## Vulnerability triage

Never sort by CVSS alone:

1. CISA KEV first: known-exploited entries get patched regardless
   of score.
2. EPSS (v4 model): daily exploitation probability. Above 0.5 is urgent,
   <0.1 deprioritize. It is a forecast, not evidence.
3. SSVC decision trees (Track/Track*/Attend/Act) for the formal
   path.
4. Asset context: exposure, reachability, criticality. A critical
   CVE in a library you never call is not a critical incident.
5. Exceptions need an expiry date, a compensating control, and an
   owner. Exceptions without expiry are permanent risk acceptance.

## This repo

The house pipeline already implements the reference pattern: SHA
pins, harden-runner, minimal permissions, dependency-review,
gosec, zizmor, Scorecard SARIF, race and fuzz CI. See
`ci-security` for the rules and motivating incidents.
