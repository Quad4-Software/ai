# Measured AI-writing tells, 2026

Organized by durability. The headline finding: lexical tells are
perishable, structural tells persist.

## The decay problem

- Opace Research (Aug 2026): 4,016 AI documents from 21 models vs
  4,144 human documents. 98 folk-wisdom "AI phrases" tested, only
  21 survived a dual-corpus 2x elevation gate, and 13 now point the
  wrong way (they are human markers).
- Kobak et al., Science Advances 2025 (arXiv:2406.07016): 15M+
  PubMed abstracts, 2010-2024. 454 excess words in 2024, nearly all
  style words. delves r=28.0, underscores r=13.8, showcasing
  r=10.7. Lower bound 13.5% of 2024 abstracts LLM-processed.
- Regression Desk (2026): 394M words of arXiv abstracts. delve ran
  2.3/million for a decade, hit 67/million Dec 2023, then halved
  every 5 months back below baseline. The tell worked, so it died.
- Yakura et al. (arXiv:2409.01754): the vocabulary leaked into
  human speech (YouTube, podcasts), then dropped once publicly
  known as a tell.

Consequence: a banned-word list is a dated snapshot. Re-check
periodically. Structural rules are the durable layer.

## Structural tells (durable)

- SlopShape (arXiv:2609.15369, Sep 2026): 187 structural features
  detect AI posts at 98.0 macro-F1, unchanged (98.1) after full
  rewording by the source model. Machine text has a tidy,
  self-announcing shape. Paraphrasing does not launder it.
- StoryScope (arXiv:2604.03136): 304 narrative features, 93.2%
  macro-F1 on fiction. AI stories over-explain themes, favor tidy
  single-track plots.
- Opace measured shape tells:
  - Words-per-paragraph CV <= 0.2: AI 13.2% vs human 0.8% (~16x).
    Strongest single document tell measured.
  - >=90% of sections within 15% of median length: AI 10.5% vs
    human 0.76% (~14x).
  - Sentence-length CV <= 0.3: AI 31.0% vs human 8.6% (3.6x).
  - Uniform list items: 49x rate difference.
  - List density inverted: humans list in 34% of sections vs AI
    18%. On 2026 models, absence of lists leans machine. The tell
    is the bold-term-colon-description item shape, not lists.

## Punctuation and construction tells

- Em dash: presence is normal, density is the tell. Human literary
  prose ~6.5 dashes per 1,000 words pooled. GPT-4.1 measured at
  10.62/1,000 vs 3.23 matched human baseline. Congressional press
  releases: em-dash density flat 2021-2024 then doubled in 2025.
  medRxiv preprints: Discussion em-dash presence rose 4.2% to 11.6%
  post-ChatGPT (OR 2.96). arXiv:2603.27006 shows suppression bans
  kill headers and bullets but em dashes persist. It is a
  fine-tuning fingerprint.
- Negative parallelism ("not X, but Y"): quadrupled in corporate
  communications 2023-2025 per Barron's. ~6% of ChatGPT messages in
  a WaPo analysis of 328k chats. Company/newsroom data, not peer
  reviewed.

## Stylistic tells

- Sycophancy (Sharma et al., ICLR 2024, arXiv:2310.13548): all 5
  tested assistants sycophantic. GPT-4o April 2025 rolled back for
  being "overly flattering or agreeable". Manifests as Great
  question openers, pseudo-empathy, I hope this helps closers,
  reflexive validation.
- Hedging: the reliable tell is miscalibrated hedging, not raw
  count. Machine text hedges settled facts and asserts significance
  confidently. Published excess-hedge figures cluster near 2x, not
  higher.
- Fake neutrality: "it depends" and "both sides" as terminal
  answers.

## Semantic tells (Wikipedia Signs of AI writing)

- Undue significance: "stands as a testament", "plays a pivotal
  role", "reflects broader trends" on ordinary facts.
- Trailing participle tails: "highlighting its importance",
  "reflecting continued relevance". Asserts meaning without
  analysis.
- Vague attribution: "experts say", "studies show", no named
  source.
- Promotional register: everything scenic, breathtaking, clean and
  modern.
- False specificity: "ranging from X to Y" that conveys nothing.
- Formatting: excess boldface, Title Case headings, markdown
  artifacts in prose, raw asterisks, oaicite or turn0search strings.
- Canned conclusions: "In conclusion", "challenges and future
  prospects" regardless of content.
- The one-line theory: LLMs produce the most statistically likely
  result for the widest variety of cases. That explains every other
  tell.

## Sources

- Wikipedia:WikiProject AI Cleanup, Signs of AI writing and AI
  catchphrases pages.
- Opace "The tells we tested" (Aug 2026), self-published but
  transparent methods.
- SlopShape arXiv:2609.15369, StoryScope arXiv:2604.03136, 2026
  preprints, not peer reviewed.
- Kobak et al. arXiv:2406.07016, peer reviewed.
- COLING 2025 "Why Does ChatGPT Delve So Much" (aclanthology
  2025.coling-main.426).
- Sharma et al. sycophancy arXiv:2310.13548.
- arXiv:2604.19768 "Saying More Than They Know" for performed
  hesitancy and tricolon rates.
