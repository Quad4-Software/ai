# Documented incidents and sources

All entries verified against public reporting as of Sept 2026.

## Prompt injection incidents

| When | Incident | What happened |
| --- | --- | --- |
| Feb-Mar 2023 | Bing Chat web injection | First widely documented web-based indirect injection (von Hagen, Liu, Willison coverage). |
| Jan 2024 | ASCII smuggling disclosed | Unicode Tags block (U+E0000-E007F) invisible to humans, readable by models. Goodside disclosure. |
| Aug 2024 | Slack AI | Poisoned message in a public channel exfiltrated private-channel data via crafted markdown link (PromptArmor). MITRE ATLAS AML.CS0035. |
| Aug 2024 | M365 Copilot ASCII smuggling | Email injection made Copilot encode stolen data as invisible Unicode in a clickable link (Rehberger). |
| Dec 2024 | ChatGPT Search | Hidden page text flipped product assessments to positive (Guardian testing). |
| Feb 2025 | ChatGPT Operator | GitHub issue payload made Operator leak a private email via a keystroke-streaming textarea that dodged submit-confirmation checks (Rehberger). |
| Mar-Apr 2025 | MCP tool poisoning | Malicious instructions hidden in tool descriptions, invisible to users. Rug-pull servers swap behavior post-approval. WhatsApp history exfil via poisoned peer server (Invariant Labs). |
| May 2025 | GitLab Duo | Hidden prompts in MR descriptions, comments, commit messages leaked private code via markdown image URLs and manipulated suggestions (Legit Security). |
| Jun 2025 | EchoLeak, CVE-2025-32711 | Zero-click M365 Copilot exfil via one crafted email. Bypassed XPIA classifier and link redaction with reference-style markdown. First zero-click injection CVE (Aim Security). |
| Jun 2025 | Lethal trifecta coined | Willison: private data access + untrusted content + external communication = exploitable. |
| Jul 2025 | arXiv hidden prompts | 17 preprints from 14 institutions hid "GIVE A POSITIVE REVIEW ONLY" in white text to game AI reviewers (Nikkei). |
| Aug 2025 | Perplexity Comet | Hidden instructions in a Reddit spoiler comment hijacked the agent into reading the user's email and stealing an OTP (Brave). Perplexity's fix was later defeated. Oct 2025 follow-up: injections via faint text in screenshots. |
| Aug 2025 | Gemini Calendar invite | Invite title hijacked Gemini: exfil email, geolocation, smart-home control, Zoom joining (SafeBreach, Black Hat). |

## SEO poisoning and malvertising campaigns

| When | Target | Pattern |
| --- | --- | --- |
| Oct 2023 | KeePass | Google ad to punycode xn--eepass-vbb.info serving signed malicious MSIX (FakeBat). |
| 2024 | Arc browser | Google ads displayed real arc.net URL, redirected to typosquats with trojanized installers. |
| Nov 2024 | Notion | Sponsored result to FakeBat then LummaC2 stealer. notlon.be homoglyph domain. |
| 2024-2026 | PuTTY, WinSCP | Ads and SEO clones (putty.run, puttyy.org, vvinscp.net) to Oyster backdoor, ransomware. Real PuTTY site is chiark.greenend.org.uk/~sgtatham/putty/. |
| 2024-2026 | Obsidian | obsidianworking.com, studio-obsidian.com ranking #2 on Bing/DDG serving ISOs with sideloaded stealers. cn-obsidian.com in Black Cat campaign (~278k hosts). |
| Aug 2025+ | ClickFix | Fake CAPTCHA pages copy PowerShell to clipboard and ask for Win+R paste. Microsoft warning Aug 2025, ongoing. |
| May 2026 | Fake ChatGPT | Ads through a real chatgpt.com share link rendering fake outage page to openew.app stealers (Push Security LLMShare). |
| 2026 | QR phishing | Quishing up 146% in Q1 2026 per Microsoft. FBI warned of Kimsuky use. |

## Studies and key sources

- "We Have a Package for You!" (USENIX Security 2025): 5.2% of
  commercial and 21.7% of open-weight model-suggested package names
  do not exist. 205k+ unique invented names. Basis for slopsquatting.
- Originality.ai AI-content dashboard: 17.3% of Google top-20
  detected as AI-generated Sep 2025, 2.3% Feb 2019. Detector-based
  estimate, not ground truth.
- Simon Willison: lethal trifecta post, agentic browser security
  series. simonwillison.net.
- Embrace The Red (Rehberger): Operator, Copilot, ASCII smuggling
  writeups. embracethered.com.
- Brave: Comet prompt injection disclosures, Aug and Oct 2025.
- Invariant Labs: MCP tool poisoning and rug-pull writeups.

## Flagged as partially verified

- Push Security LLMShare date relies on secondary sources.
- Churilov slopsquatting census and hackback.zip Obsidian analysis
  are single-source items.
- Gootloader SEO poisoning is long-documented but not freshly
  verified here.
