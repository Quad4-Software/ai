---
name: owasp
description: >
  This skill covers the OWASP project landscape as of September 2026:
  the Top 10 2025 web list, API Security Top 10 2023, LLM Top 10 2026,
  the new Agentic Applications Top 10, ASVS 5.0, MASVS/MASTG 2.x, WSTG,
  Cheat Sheet Series, SAMM 2.0, CycloneDX 1.7, Dependency-Check/Track,
  Juice Shop, and friends. Use it when classifying a finding, picking a
  testing guide, or citing the current category list.
---

## When to use this skill

- You need the current Top 10 categories (2025 edition, not 2021).
- You are mapping a vulnerability to an OWASP or MASWE identifier.
- You are choosing a testing/verification standard (WSTG, ASVS, MAS).
- You are citing AI/LLM/agent risks with the right codes.

## How to use

1. Read this file for what is current and what changed hands.
2. Load [references/lists.md](references/lists.md) for the category
   lists themselves, and [references/projects.md](references/projects.md)
   for tools, maturity models, and SBOM projects.
3. Canonical sources: https://owasp.org/Top10/2025/,
   https://genai.owasp.org/, https://mas.owasp.org/,
   https://cheatsheetseries.owasp.org/

## Examples

- "Which OWASP category does a poisoned CI cache fall under?"
- "Give me the LLM Top 10 2026 codes for prompt injection and memory
  poisoning."
- "Is ZAP still an OWASP project?"

# OWASP in 2026

## What is current

- **Top 10 2025** is the released web list (final, Nov 2025). First
  update since 2021. New: A03 Software Supply Chain Failures, A10
  Mishandling of Exceptional Conditions. SSRF folded into A01 Broken
  Access Control.
- **API Security Top 10 2023** still current.
- **LLM Top 10 2026** shipped Aug 4, 2026. Renumbered: only LLM01
  (Prompt Injection) and LLM02 (Sensitive Information Disclosure) kept
  their slots. Do not map 2025 codes to 2026 by position.
- **Agentic Applications Top 10** (Dec 2025, ASI01-ASI10 codes) covers
  agent goal hijack, tool misuse, memory/context poisoning, rogue
  agents. Agent Control Standard (ACS) announced Sept 2026.
- **ASVS 5.0** (May 2025): 17 chapters, L1/L2/L3 levels, new chapters
  for Web Frontend, Self-contained Tokens, OAuth/OIDC, WebRTC. New
  requirement IDs: v5.0.0-<chapter>.<section>.<req>.
- **MAS (mobile):** MASVS 2.1.0 (Jan 2024) is current. L1/L2/R levels
  were removed, replaced by testing profiles. MASTG 2.0.0 shipped
  July 2026 with MASWE weakness traceability.
- **WSTG:** v4.2 (Dec 2020) is the last numbered release. The web
  version rolls forward continuously.
- **SAMM 2.0:** 5 business functions x 3 practices, 15 total, streams
  per practice.
- **Cheat Sheet Series:** ~120 sheets, Flagship, indexed against ASVS,
  MASVS, Top 10, and Proactive Controls.

## Ownership changes to flag

- **ZAP is no longer OWASP.** Left Sept 2023, branded "ZAP by
  Checkmarx" since Sept 2024. Still Apache-2.0. Do not call it OWASP
  ZAP.
- **Dependency-Track v5.0** (June 2026, "Hyades" redesign) is GA. V4
  line EOL ~Dec 2026, no in-place upgrade.
- **CycloneDX 1.7** (Oct 2025), ratified ECMA-424 2nd ed (Dec 2025).
- **Amass** repo moved to github.com/owasp-amass/amass (still OWASP).
- **Juice Shop v20** (May 2026) added AI/LLM challenges.

## Reference files

- [references/lists.md](references/lists.md): Top 10 2025, API 2023,
  LLM 2026, Agentic ASI codes, ML Top 10 (draft) with one-line scope
  notes per category.
- [references/projects.md](references/projects.md): ASVS 5.0 chapter
  map, MAS structure, SAMM practices, Cheat Sheet usage, WSTG sections,
  tool table (Dependency-Check, Dependency-Track, ZAP, Juice Shop,
  WebGoat, Nettacker, Amass, Cornucopia, Threat Dragon, AI Testing
  Guide, AIVSS).
