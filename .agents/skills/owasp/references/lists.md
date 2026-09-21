# OWASP category lists

## Top 10 2025 (current web edition)

Released Nov 2025, first update since 2021.

| Code | Category | Note |
| --- | --- | --- |
| A01:2025 | Broken Access Control | Absorbs SSRF (was A10:2021) |
| A02:2025 | Security Misconfiguration | Up from #5 |
| A03:2025 | Software Supply Chain Failures | NEW. Replaces Vulnerable & Outdated Components. Covers deps, build systems, distribution |
| A04:2025 | Cryptographic Failures | Was A02:2021 |
| A05:2025 | Injection | Was A03:2021 |
| A06:2025 | Insecure Design | Was A04:2021 |
| A07:2025 | Authentication Failures | Renamed from Identification and Authentication Failures |
| A08:2025 | Software or Data Integrity Failures | Unchanged position |
| A09:2025 | Security Logging and Alerting Failures | Renamed from Logging and Monitoring |
| A10:2025 | Mishandling of Exceptional Conditions | NEW. Fail-open, bad error handling, logic errors |

## API Security Top 10 2023 (current)

API1 Broken Object Level Authorization (BOLA) - the dominant API
finding. API2 Broken Authentication. API3 Broken Object Property
Level Authorization (mass assignment + excessive data exposure
merged). API4 Unrestricted Resource Consumption. API5 Broken Function
Level Authorization. API6 Unrestricted Access to Sensitive Business
Flows (scalping, fake accounts). API7 SSRF. API8 Security
Misconfiguration. API9 Improper Inventory Management. API10 Unsafe
Consumption of APIs (trusting third-party responses).

## LLM Top 10 2026 (current, shipped Aug 4 2026)

| Code | Category |
| --- | --- |
| LLM01:2026 | Prompt Injection |
| LLM02:2026 | Sensitive Information Disclosure |
| LLM03:2026 | Excessive Agency (climbed from #6 in 2025) |
| LLM04:2026 | Supply Chain |
| LLM05:2026 | Data and Model Poisoning |
| LLM06:2026 | Unbounded Consumption |
| LLM07:2026 | Misinformation |
| LLM08:2026 | Hidden Context Exposure (was System Prompt Leakage) |
| LLM09:2026 | Vector and Embedding Weaknesses |
| LLM10:2026 | Improper Output Handling |

Codes do not map to the 2025 list by position. When citing older
reports, verify which edition they use.

## Agentic Applications Top 10 (Dec 2025)

ASI codes for agent systems: ASI01 Agent Goal Hijack, ASI02 Tool
Misuse and Exploitation, ASI03 Identity and Privilege Abuse, ASI04
Agentic Supply Chain Vulnerabilities, ASI05 Unexpected Code Execution
(RCE), ASI06 Memory and Context Poisoning, ASI07 Insecure Inter-Agent
Communication, ASI08 Cascading Failures, ASI09 Human-Agent Trust
Exploitation, ASI10 Rogue Agents.

Adjacent: Agent Control Standard (Sept 2026) for runtime controls, and
the AI Testing Guide (v1, Nov 2025) with test IDs AITG-APP /
AITG-MOD / AITG-INF / AITG-DAT.

## ML Security Top 10 (draft, v0.x)

ML01 Adversarial Attack, ML02 Data Poisoning, ML03 Model Inversion,
ML04 Membership Inference, ML05 Model Stealing, ML06 Corrupted
Packages, ML07 Transfer Learning Attack, ML08 Model Skewing, ML09
Output Integrity Attack, ML10 Neural Net Reprogramming. Still draft,
do not cite as finalized.

## Quick classification map for common findings

- Missing per-object authorization check -> A01 / API1 (BOLA)
- SQLi / SSTI / command injection -> A05
- Stale or malicious dependency, unpinned action, poisoned build ->
 A03:2025
- Fail-open on exception, catch-and-continue on auth -> A10:2025
- JWT alg confusion, weak crypto -> A04 / ASVS V11
- Prompt injection into an agent tool call -> LLM01 + ASI02
- Poisoned RAG/vector store -> LLM09, LLM05
- MCP tool description poisoning -> ASI02 / ASI06
