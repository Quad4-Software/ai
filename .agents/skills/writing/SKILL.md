---
name: writing
description: >
  This skill covers writing prose that reads as human-authored:
  essays, papers, forum and comment replies, emails, and responses
  to posts. Use it whenever drafting long-form prose or replies for
  a real person, or when auditing text for machine-writing tells.
  Covers the measured AI tells as of 2026 (structural tells are
  durable, lexical tells decay), genre-specific shapes, matching a
  user's own voice from handwritten samples, and the classic style
  rules. For linting generated docs and comments see the no-slop
  skill.
metadata:
  sources:
    - https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing
    - https://reticulum.network/manual/brandolinis.html
    - https://opace.agency/tools/ai/content-verification-integrity/research/the-tells-we-tested/
---

## When to use this skill

- Drafting or editing essays, papers, forum replies, comments, or
  emails meant to come from a real person.
- Auditing a draft for machine-writing tells.
- Matching a user's voice when they provide writing samples.

## The core argument

Brandolini's law: refuting bullshit takes an order of magnitude
more effort than producing it. Machine generation made production
nearly free, so readers and gatekeepers now penalize the *shape* of
unaccountable text: fluent, self-announcing, unspecific, marketing
register. "Reads as human" really means "reads as accountable": a
voice that could answer for its own claims, specifics that can be
checked, no borrowed authority.

The fix is never a word blacklist. Lexical tells decay (delve fell
94% from its 2024 peak once flagged). Structural tells persist:
the SlopShape study detects machine text at 98% macro-F1 from
structure alone, unchanged after full rewording. Write like a
person, do not paraphrase like a machine.

## Workflow

1. Pick the genre shape. See references/genre-shapes.md. A reply
   with headings is slop regardless of the words in it.
2. Establish the voice. If the user provided handwritten samples,
   extract a voice profile first (references/voice-profile.md).
   If no samples exist, ask for some, or at minimum ask the
   register questions. Voice matching from a real corpus is the
   only reliable personalization. Mark Qvist's Brandolini essay was
   itself drafted with a model grounded in ~100k tokens of his own
   handwritten archive: that is the legitimate version of this
   technique.
3. Draft with specifics. Concrete claims, real numbers, named
   sources, the actual situation. Text that could be pasted into
   any context unchanged is machine-shaped regardless of wording.
4. Slop pass. Kill the measured tells: uniform paragraph and
   section lengths, uniform sentence rhythm, bold-term-colon list
   items, announce-then-execute scaffolding, significance padding
   ("stands as a testament"), trailing participle tails
   ("highlighting its importance"), vague attribution ("experts
   say"), canned conclusions ("In conclusion"), negative
   parallelism overuse ("not X, but Y"), em dash density above
   ~3-7 per 1,000 words, markdown artifacts in prose.
5. Accountability pass. Read it as a skeptical stranger: could the
   author answer for every claim? Does anything assert without
   evidence? Cut what cannot be defended.
6. Lint with the no-slop MCP tools for the mechanical checklist.

## Voice overrides bans

The banned-word and punctuation rules describe default model
output, not a human floor. If the user's own writing uses em
dashes, semicolons, lowercase starts, or "delve", matching their
voice means using those. A voice profile from real samples beats
any generic rule list.

## Genre invariants

- Every genre punishes the same thing: text that fits any context.
- Short genres (comments, emails, forum replies) are where
  detectors are blind anyway. The audience is human readers, so
  write for them: answer the specific thing, stay short, sound
  like a person who was actually there.
- Essays: thesis emerges, does not announce. Endings pay off,
  they do not summarize.
- Papers: hedge specific claims with citations, never hedge
  everything. Machine text hedges everything and cites nothing,
  or invents citations.
- Replies: open on the exact point being answered. 1-3 short
  paragraphs. First person. No scaffolding.

## The tells, in one table

Durable (structural): uniform paragraph lengths (AI 13% vs human
0.8% for paragraph-length CV <= 0.2), uniform section lengths,
uniform sentence rhythm, self-announcing scaffold, tidy resolution.
Decaying (lexical): delve, underscore, showcase, tapestry,
testament, pivotal, vibrant, realm, landscape-as-metaphor.
Register: sycophancy ("Great question"), performed hesitancy at 2x
human rate, hedging settled facts while asserting significance,
negative parallelism, em dash density.

Full measured data in references/tells-2026.md.

## Boundaries

Matching the user's own voice and making machine-assisted prose
read like the person who stands behind it is the job. Do not use
this to impersonate someone else, astroturf, or violate explicit
authorship rules (academic integrity, disclosure-required venues).
Detectors themselves are unreliable (up to 61% false positives on
non-native speakers) so never present a detector verdict as proof,
in either direction.

## References

- `references/tells-2026.md` - measured tells with numbers and
  sources, organized by durability.
- `references/genre-shapes.md` - essay, paper, reply, email shapes.
- `references/voice-profile.md` - voice extraction checklist and
  the questions to ask when no samples exist.
- `references/style-classics.md` - Orwell, Gopen, Zinsser, Strunk,
  King condensed.
- `references/brandolini.md` - the asymmetry argument and grounded
  generation model.
- `references/detection-reality.md` - detector error rates, evasion
  research, why human readers are the real audience.
