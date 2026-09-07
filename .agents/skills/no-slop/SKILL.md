---
name: no-slop
description: >
  Use when writing or editing prose, documentation or code comments in this
  repo. Catches AI slop, machine tics, semicolons, backtick inflation in
  docs and comments, Markdown leakage, sycophancy, emojis, em dashes,
  filler and hedging.
---

# No slop

This repo ships `no-slop-mcp`, a stdio MCP server that lints prose against
no-AI-slop and MeshChatX style rules. The server is fully offline, stdlib only,
and produces a single static binary.

Some rules below are adapted from the
`realrossmanngroup/no_ai_slop_writing_rules` skill pack. The goal is the same:
replace vague, machine-shaped text with specific, checkable facts.

## Tool reference

Tool details are in [references/tools.md](references/tools.md). This section is optional if no-slop-mcp is installed.

## Core rule

Every claim must end on a concrete, checkable detail. If a sentence could be
moved to any other document without changing a word, it is generic. Anchor it to
a date, a number, a name, a file or a specific behaviour.

- WRONG: "This practice has had a significant impact on people."
- RIGHT: "The company replaced 11 million batteries in 2018, against the 1 to 2
  million it had expected."

## 2026 machine tics to kill

LLMs in 2026 overuse the same small tics. The most obvious are semicolons,
backticks, em dashes, emojis and dramatic headings. The `comment` mode in
no-slop-mcp is especially strict about code comments.

### No semicolons in prose

A semicolon in prose is almost always a machine trying to sound formal. Break
the sentence in two or use a comma.

- WRONG: "The fix is simple, just restart the daemon." (joined as one sentence with a
  semicolon-like pause)
- RIGHT: "Restart the daemon. That is the fix."

### No backticks in code comments

Backtick formatting inside source-code comments is a recent LLM tic. Code
comments are plain text. Use the language's own comment markers. If you need to
refer to a name, write it without backticks.

- WRONG:

  ```
  // `processData` handles the `input` and returns the `result`.
  ```

- RIGHT:

  ```
  // processData handles the input and returns the result.
  ```

This is the rule the `comment` mode enforces. The `doc` and `prose` modes still
allow backticks for code terms, but use them sparingly.

### Backtick inflation in docs

Models trained on Markdown-saturated corpora wrap every technical noun in
backticks. Research on Markdown leakage shows the formatting impulse survives
even when models are told to write plain prose. It is the same mechanism that
produces em dash overuse: structural markup bleeding into running text.

Rules for README and Markdown files:

- Reserve backticks for things a reader would type or paste: commands, flags,
  file paths, config keys, function names in API reference sections.
- Do not backtick plain English nouns that happen to be technical: the parser,
  the server, the token, the cache. Write them as words.
- Do not backtick project names, product names or protocol names.
- One inline code span per paragraph is plenty. Three or more in a single
  sentence is a machine tell.
- Never backtick a name inside a heading.

- WRONG: "The `parser` reads `micron` input and produces a `token` stream."
- RIGHT: "The parser reads Micron input and produces a token stream."

### Other Markdown leakage

The same training artifact produces formatting habits that belong in chat
output, not in documentation:

- Bulleted or numbered lists where every item starts with a bold term, a
  colon, then a description. This is the single most characteristic chat-shape
  artifact. In docs, use a plain list or rewrite as prose.
- Bolding key terms mechanically, "key takeaways" style. Bold is for genuine
  warnings, not emphasis.
- Title Case On Every Heading. Use sentence case.
- Bullet points for answers that need one sentence.
- A summary at the end that restates what the doc just said. Cut it.
- Curly quotes and apostrophes. Some models default to them. Use straight
  ASCII quotes in source and docs.

### No em dashes

The em dash character is banned. Use a comma, a period, parentheses or rewrite.

- WRONG: "The policy -- which affected millions -- was later reversed."
- RIGHT: "The policy affected millions of devices. The company reversed it in
  December 2017."

### No emojis or decorative arrows

Rocket, fire, checkmark, pointing finger and arrow emojis are AI slop. Do not
use them in prose, headings or commit messages.

- WRONG: "Key features include: secure, fast and reliable."
- RIGHT: "The server is read-only, path-jailed and redacts secrets."

## Banned words

### Overused verbs

| Avoid | Use instead |
|-------|-------------|
| delve | explore, examine, investigate |
| leverage | use, apply, draw on |
| harness | use, apply |
| showcase | show, demonstrate |
| garner | get, earn, attract |
| elevate | raise, improve |
| empower | enable, allow |
| unlock | open, enable, allow |
| embark | start, begin |
| optimise | improve, refine, enhance |
| utilize | use |
| facilitate | help, enable, support |
| foster | encourage, support, develop |
| bolster | strengthen, support, reinforce |
| underscore | emphasise, highlight |
| unveil | reveal, show, introduce |
| navigate | manage, handle, work through |
| streamline | simplify, make more efficient |
| enhance | improve, strengthen |
| endeavour | try, attempt |
| ascertain | find out, determine |
| elucidate | explain, clarify |

### Overused adjectives

| Avoid | Use instead |
|-------|-------------|
| robust | strong, reliable, solid |
| comprehensive | complete, thorough, full |
| pivotal | key, critical, central |
| crucial | important, essential |
| vital | important, essential |
| transformative | significant, major |
| cutting-edge | new, advanced, recent |
| groundbreaking | new, original |
| innovative | new, original, creative |
| seamless | smooth, easy |
| intricate | complex, detailed |
| nuanced | subtle, complex |
| multifaceted | complex, varied |
| holistic | complete, whole |
| meticulous | careful, thorough |
| vibrant | lively, active |

Also flag figurative uses of landscape, realm, tapestry, testament, interplay
and boasts meaning "has".

### Overused transitions

| Avoid | Use instead |
|-------|-------------|
| furthermore | also, in addition, and |
| moreover | also, and |
| additionally | also, and |
| consequently | so, as a result |
| notably | note, or cut |
| importantly | or cut |
| in summary | or cut |
| ultimately | or cut |
| notwithstanding | despite, still |
| that being said | however, but |
| at its core | essentially, basically |
| to put it simply | in short |
| it is worth noting that | note that |
| in the realm of | in, regarding |
| in the landscape of | in, within |
| in today's [anything] | currently, now |

### Filler phrases and openers

Avoid these openers and transitions in any prose:

- "In today's fast-paced world..."
- "In today's digital age..."
- "In an era of..."
- "In the ever-evolving landscape of..."
- "In the realm of..."
- "It's important to note that..."
- "Let's dive in..."
- "Here's the thing..."
- "But here's the kicker..."
- "In a world where..."
- "That being said..."
- "With that in mind..."
- "It's worth mentioning that..."
- "To put it simply..."
- "In essence..."
- "This begs the question..."
- "In conclusion..."
- "To sum up..."
- "All things considered..."
- "At the end of the day..."
- "By [doing X], you can [achieve Y]..."
- "When it comes to..."
- "Let's delve into..."
- "Embark on a journey..."
- "Based on the information provided..."
- "Setting the stage for..."
- "As we navigate..."
- "Without further ado..."

### Inflated symbolism

Avoid these metaphorical multi-word phrases:

- "provide a valuable insight"
- "left an indelible mark"
- "play a significant role in shaping"
- "an unwavering commitment"
- "open a new avenue"
- "a stark reminder"
- "gain a comprehensive understanding"
- "serves as a testament"
- "watershed moment"
- "deeply rooted"
- "a tapestry of..."
- "a symphony of..."
- "a beacon of..."
- "a treasure trove of..."
- "a game-changer"
- "the transformative power of..."
- "plays a pivotal role"
- "plays a crucial role"
- "stands as a..." and "serves as a..." instead of "is"
- "seamless integration"

Only use the underlying word literally, not metaphorically.

### Academic slop

| Avoid | Use instead |
|-------|-------------|
| shed light on | clarify, explain |
| pave the way for | enable, allow |
| a myriad of | many, numerous |
| a plethora of | many, several |
| paramount | very important, essential |
| pertaining to | about, regarding |
| prior to | before |
| subsequent to | after |
| in light of | because of, given |
| with respect to | about, regarding |
| in terms of | regarding, for |
| the fact that | that, or rewrite |

## Heading anti-patterns

A heading names the section. It does not tease, dramatize or abstract.

| Pattern | Bad | Good |
|---------|-----|------|
| "The [Concept] Trap" | "The Pricing Trap" | "How subscription pricing adds up" |
| "The [Adjective] [Noun]" | "The Hidden Cost" | "Economic impact of shortened lifespans" |
| "The [Noun] [Dramatic Noun]" | "The Silent Killer" | "Bad sector growth on aging platters" |
| "Why [X] [Verb] [Y]" | "Why Rebuilding Destroys Everything" | "Forced rebuilds overwrite parity on degraded arrays" |
| "[Noun]: The [Adjective] [Noun]" | "Encryption: The Hidden Trap" | "Hardware AES-256 encryption on WD Passport bridges" |

Ask: could this heading be a thriller chapter title? If yes, rewrite it.

## Structural slop

### Paragraph uniformity

If every paragraph in a section is the same length, the section looks machine
made. Vary paragraph length to match the complexity of the point.

- WRONG: four paragraphs, each exactly three sentences
- RIGHT: one short paragraph, one long explanation, one short punch

### Sentence length

Human writing alternates short and long sentences. Machine text clusters around
15 to 20 words. Use a mix of short declarative sentences and longer
clause-heavy ones.

### Transition density

Do not start every paragraph with a transition word. More than 30% of
paragraphs opening with a transition reads as artificial.

### Opening word repetition

Three or more consecutive paragraphs that start with the same word or pattern
indicate mechanical generation. Vary the openings.

### Contrasting parallelism

The "It is not X, it is Y" pattern is a dead giveaway. More than two of these
in a 500-word block is a high-confidence AI indicator.

- WRONG: "This is not just a price increase. It is a betrayal of trust."
- RIGHT: "The price went up and the company knew it would lose customers."

Related patterns, same verdict:

- "Not only does X do Y, but it also does Z."
- "No setup, no config, just results."
- More than one "It is not X, it is Y" reframing per document.

### Rule of three

Models pad shallow analysis into triples: "fast, reliable and secure" or three
parallel phrases where one fact would do. A triple is fine when all three items
are real and distinct. It is slop when the items are synonyms.

- WRONG: "The tool is powerful, flexible and easy to use."
- RIGHT: "The tool parses 40,000 lines per second on a single core."

### Vague attribution

Models generalize opinion with unsourced plurals. Ban these unless followed by
a named source:

- "Studies show..."
- "Experts say..." or "Experts agree..."
- "Research suggests..."
- "It is widely believed..." or "It is widely acknowledged..."
- "Some argue..." or "Many believe..."

### Canned conclusions

Model text ends by pivoting to "challenges and future prospects" or "the road
ahead" regardless of content. A document ends when the facts end. No outlook
paragraph, no summary of what was just said, no call to action the reader did
not ask for.

## Sycophancy and chat leftovers

Current alignment training produces flattery and filler that has no place in
docs, comments or commit messages. Studies across frontier models in 2026 show
sycophantic openers correlate strongly with low perceived naturalness.

Banned openers:

- "Great question!" or "That's a great question!"
- "Absolutely!" or "Certainly!" as a standalone opener
- "I'd be happy to help!"
- "Sure!" or "Of course!" before answering

Banned closers:

- "I hope this helps!"
- "Let me know if you need anything else!"
- "Feel free to..."
- "Happy to elaborate further!"

Banned mid-text tics:

- Pseudo-empathy: "I completely understand your concern."
- Announcements: "Now let's look at..." or "Let me explain..."
- Cheerleading the reader's own idea back at them.

Docs and commit messages need none of this. Answer directly.

## Hedging

AI models hedge 4 to 7 times more than humans. Established facts do not need
caveats.

- WRONG: "Serialization may potentially prevent independent repair in some
  cases."
- RIGHT: "Replacing an iPhone 15 camera module without the manufacturer's
  calibration software disables optical image stabilization."

Hedging is acceptable only for genuinely disputed or pending facts.

### Hedging markers to flag

- may, might, could, potentially
- probably, generally, usually, arguably, likely
- unclear, remains to be seen, further research is needed
- "It is worth noting that..."
- "It should be noted that..."
- "One could argue that..."
- "It is widely acknowledged that..."

More than three hedging markers in a single paragraph is a red flag.

### Fake neutrality

A related tic is refusing to state anything: "it depends on your specific
needs", "there are pros and cons to both", "both have their merits", "varies
from person to person". Pick the recommendation that fits the documented case
and say why. If the answer genuinely depends, state the deciding factor, not
the shrug.

## Filler and empty intensifiers

Remove or replace these words:

- absolutely, actually, basically, certainly, clearly, definitely
- essentially, extremely, fundamentally, incredibly, interestingly, naturally
- obviously, quite, really, significantly, simply, surely
- truly, ultimately, undoubtedly, very

## Hallucinated markup artifacts

Any of these strings in text means it was pasted from an AI tool without
editing. Zero tolerance.

- `oaicite`
- `contentReference`
- `grok_card`
- `attributableIndex`
- `turn0search0`

## Code and doc modes

- `prose`: general writing. Allows backticks for code terms but not em dashes.
- `doc`: documentation files. Bans semicolons in addition to the prose rules.
- `comment`: code comments. Bans backticks inside comments in addition to the
  prose rules.

When in doubt, use the `comment` mode for source code and the `doc` mode for
README and Markdown files.

## Self-check before returning text

Run this pass on every piece of prose before handing it back.

1. Search for the em dash character. Remove every one.
2. Scan for banned verbs, adjectives, transitions and openers.
3. Check every number. Is it real and attributable? If not, cut it.
4. Check that every sentence ends on a concrete detail.
5. Check headings. Does each name the content, not tease it?
6. Look for repeated points and repeated section shapes.
7. Count hedging markers per paragraph. More than three is a red flag.
8. Search for hallucinated markup artifacts.
9. In code comments, remove every backtick around names.
10. In docs, count inline code spans. Backticks only for typed input: commands,
    flags, paths, config keys, API names in reference sections.
11. Remove semicolons from prose and documentation.
12. Remove bold-header-colon list items, decorative bold, Title Case headings,
    curly quotes and restating summaries.
13. Remove sycophancy: openers, closers, announcements, empathy filler.
14. Check triples and negative parallelisms. Cut synonyms dressed as lists.
15. Check attributions. Every "studies show" needs a named source or a rewrite.
16. Read it aloud. If a phrase sounds unnatural, rewrite it.
