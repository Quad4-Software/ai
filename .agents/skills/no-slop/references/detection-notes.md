# Detection research notes

Measured data on what actually separates machine text from human text in
2026. Use this to calibrate the rules in SKILL.md. The linter enforces
style. It is not an AI detector and must not be used to accuse anyone of
misconduct.

## Sources

- Wikipedia, Signs of AI writing (WikiProject AI Cleanup field guide).
  The canonical public catalogue of tells.
- Opace Research, "The tells we tested", 2026-08-31. 4,016 AI documents
  from 21 models against 4,144 human web documents, with a same-day
  re-measurement against 3,529 structure-preserved human documents.
- SlopShape, arXiv:2609.15369. 2,250 pre-ChatGPT company blog posts
  against 11,250 AI mirrors from five frontier models. Structural
  features alone detect AI posts at 98.0 macro-F1, unchanged after
  rewording.
- humzakt/ai-writing-markers. Source-backed marker dataset, 87 markers
  in 6 categories, ships a zero-dependency checker.
- Liang et al., Patterns 2023. GPT detectors flag non-native English
  writing at high false-positive rates. Reported detector
  false-positive range across studies: roughly 5% to over 60%.

## What the measurements say

Vocabulary tells decay. Shape tells persist.

Ninety-eight famous "AI phrases" were measured on two independent
AI/human pairings. Twenty-one survive. Thirteen now point the wrong
way and are human markers. A static banned-phrase list flags human
writing and misses current model output. Treat the banned-word tables
in SKILL.md as a style preference for this project's docs, not as
evidence about who wrote a text.

The strongest surviving signals are structural:

- Paragraph-length uniformity. 13.2% of AI documents hold the
  coefficient of variation of paragraph length under 0.2, against 0.8%
  of structured human documents. The strongest document-shape tell
  measured so far, and it holds on hard negatives.
- List density inverted. Humans put lists in 34% of sections against
  18% for AI. Heavy bullet use fires on 15.7% of human documents
  against 4.9% of AI. On 2026 models, the absence of lists leans
  machine, not their presence. The tell is the list shape (every item
  a bold term, colon, description), not the list itself.
- Self-announcing structure. SlopShape detects AI posts from 187
  structural features at 98 macro-F1, and attributes the source model
  correctly 79.3% of the time against a 16.7% chance rate. The signal
  survives full rewording: how information is presented, in what order,
  with what evidence, and in what voice. AI documents share a tidy,
  self-announcing shape. Humans occupy rare structural configurations.

## Practical consequences for the linter

1. Keep the shape rules (paragraph uniformity, self-announcing
   scaffolding, bold-colon list items, canned conclusions). These are
   the durable tells.
2. Keep the banned-word tables as project style rules. Do not present
   them as authorship evidence.
3. Do not strip normal lists to look less machine. That now points the
   wrong way.
4. Never accuse a writer based on these markers. False-positive rates
   are high and biased against non-native speakers and formal
   register. The output of any check is a prompt for a human read,
   nothing more.

## Slop examples

Self-announcing scaffolding, the SlopShape signature:

- WRONG: "This report examines the factors that influence adoption and
  outlines the key findings." Followed by sections that execute the
  announced plan in order.
- RIGHT: open on the first finding. The reader learns the structure by
  reading it.

Uniform paragraphs, the Opace signature:

- WRONG: five paragraphs in a row, each two to four sentences, each
  opening with a different transition word but the same rhythm.
- RIGHT: a one-line paragraph, a dense six-sentence paragraph, a
  two-line paragraph. Length tracks the weight of the point.

Bold-colon list items, still the strongest single chat-shape artifact:

- WRONG:
  - Speed: the server answers in under 10 ms.
  - Security: all traffic is encrypted.
  - Reliability: retries are automatic.
- RIGHT: "The server answers in under 10 ms, encrypts all traffic, and
  retries on failure." Or a plain list without the bold-lead shape.
