# Voice profile extraction

Goal: match the user's own writing, not a generic human voice.
The user should provide handwritten samples for real matching.
Machine drafting grounded in a corpus of the author's own work is
the legitimate version of this technique. It is what Mark Qvist
describes doing for the Brandolini chapter: ~100k tokens of his
own handwritten archive as grounding.

## What to ask for

3-10 same-genre samples, 2k+ words total ideal. Essays for essays,
comments for comments. Genre matters because everyone writes
differently across genres.

If the user cannot or will not provide samples, ask these instead:

- Register: formal, casual, or in between.
- Audience and English variant (US, UK, other).
- Typical length. Short or long by default.
- Contractions, emoji, swearing: yes or no.
- How they disagree: blunt, hedged, sarcastic.
- Complete sentences or fragments.
- One or two things they always say or never say.

## Extraction checklist

Given samples, measure or note:

1. Sentence shape: mean length, spread, fragments, 30+ word
   sentences, comma splices, run-ons.
2. Paragraph habits: one-liners vs wall of text, average
   sentences per paragraph.
3. Punctuation fingerprint: parenthetical asides, dash habits
   (hyphen, spaced dash, none), ellipsis, exclamation rate,
   question rate, semicolons, Oxford comma.
4. Lexical register: contractions, plain vs Latinate vocabulary,
   jargon density, profanity, pet phrases, actual filler words
   the user uses (not imagined ones).
5. Errors and idiosyncrasy: characteristic typos, its/it's, teh,
   lowercase sentence starts, regional spelling, non-native
   patterns.
6. Discourse habits: how they open (greeting vs direct), how they
   hedge (I think, afaik), how they disagree, how they close,
   quoting conventions, list use.

## Output format

A voice card: a short bullet list of measured habits, PLUS 2-3
verbatim exemplar passages. Exemplars carry more signal than
prose rules.

## What the research says

- EMNLP 2025 Findings "Catch Me If You Can? Not Yet": LLMs
  approximate style in structured formats (news, email) but
  struggle with informal writing (blogs, forums). More samples
  help but do not close the gap. Compensate with more exemplars
  for informal genres.
- Writeprints (Abbasi & Chen, ACM TOIS 2008): canonical feature
  taxonomy. Function words are the strongest attribution signal
  and the least consciously controlled, which is why "write
  casually" fails but function-word-profile matching partially
  works.
- Instruction-tuned models struggle to reproduce author style
  from a few hundred words of demonstration (StyleMC,
  arXiv:2312.17242).

## Override rule

The user's voice overrides generic anti-tell lists. If the user
writes em dashes, semicolons, lowercase starts, or flagged words,
matching them means using those. Bans describe default model
output, not a human floor.
