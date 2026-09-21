# OWASP projects and standards

## Verification and testing standards

**ASVS 5.0** (May 2025) - application security verification standard.
Three levels: L1 baseline, L2 standard for apps with sensitive data,
L3 highest assurance. 17 chapters: V1 Encoding/Sanitization, V2
Validation and Business Logic, V3 Web Frontend, V4 API and Web
Service, V5 File Handling, V6 Authentication, V7 Session Management,
V8 Authorization, V9 Self-contained Tokens, V10 OAuth and OIDC, V11
Cryptography, V12 Secure Communication, V13 Configuration, V14 Data
Protection, V15 Secure Coding and Architecture, V16 Logging and Error
Handling, V17 WebRTC. Requirement IDs: `v5.0.0-<ch>.<sec>.<req>`.

**WSTG** - Web Security Testing Guide. v4.2 (Dec 2020) last numbered
release. Web version rolls. Test IDs like WSTG-INPV-19 (SSRF),
WSTG-APIT-01 (GraphQL). Pair with the Checklist.

**MAS (mobile)** - MASVS 2.1.0 categories: STORAGE, CRYPTO, AUTH,
NETWORK, PLATFORM, CODE, RESILIENCE, PRIVACY. Verification levels
removed. Profiles live in MASWE/MASTG. MASTG 2.0.0 (July 2026) maps
each test to a MASWE weakness. MAS checklist spreadsheet removed in
v2 - the site is authoritative.

**Cheat Sheet Series** - ~120 sheets at cheatsheetseries.owasp.org,
indexed against ASVS, MASVS, Top 10, Proactive Controls. First stop
for "how do I do X securely" answers.

**SAMM 2.0** - maturity model: Governance, Design, Implementation,
Verification, Operations x 3 practices each, streams A/B, three
maturity levels. Use it to score a program, not a codebase.

**AI Testing Guide** (v1, Nov 2025) - test IDs by layer: AITG-APP
(prompt injection, hallucination), AITG-MOD (evasion, poisoning,
membership inference), AITG-INF (supply chain, resource exhaustion),
AITG-DAT (data). AIVSS v0.8 scores AI vulns like CVSS with agentic
amplification factors.

## SBOM and dependency projects

- **CycloneDX 1.7** (Oct 2025, ECMA-424 2nd ed Dec 2025): adds CBOM
 crypto inventory, patents, TLP constraints, citations.
- **Dependency-Track 5.x** (v5.0 June 2026 "Hyades" redesign, v5.1
 Aug 2026): SBOM platform, PostgreSQL only, horizontal scale. v4
 EOL ~Dec 2026, no in-place upgrade.
- **Dependency-Check 13.x**: CLI/plugins SCA scanner. Needs NVD API
 key for decent throughput (12.1+ mandatory after NVD changes).

## Training apps

- **Juice Shop v20** (May 2026): flagship deliberately-insecure app,
 100+ challenges incl. AI/LLM (prompt injection), mapped to Top 10.
- **WebGoat v2025.3**: guided Java/Spring lessons.
- **Cornucopia v2**: threat-modeling card game, ASVS 4.x website
 edition + MASVS 2.x mobile edition, play at copi.owasp.org.

## Tools table

| Tool | Status | Note |
| --- | --- | --- |
| ZAP | Left OWASP Sept 2023, "ZAP by Checkmarx" since 2024, Apache-2.0, Java 17+ | DAST proxy. Not OWASP anymore |
| Dependency-Track | v5.x GA, v4 EOL ~Dec 2026 | SBOM/SCA platform |
| Dependency-Check | v13.x | SCA CLI/plugins, needs NVD API key |
| Amass | Active, repo at owasp-amass/amass, v5.x | External attack surface, OSINT |
| Nettacker | Active | Recon/pentest framework |
| Threat Dragon | v2.x (v3 planned) | STRIDE threat modeling, TM-BOM |
| Cornucopia | v2.0 | Card game |
| Juice Shop / WebGoat | Active | Vulnerable-by-design apps |

## AI project umbrella (genai.owasp.org)

LLM Top 10 2026, Agentic Applications Top 10, Agent Control Standard,
AI Testing Guide, ML Top 10 (draft), Agentic Threats and Mitigations,
GenAI Data Security Top 10, AI Exchange (feeds EU AI Act standards).
