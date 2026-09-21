# Stylometric feature categories

Stylometry rests on the assumption that writers have stable,
unconscious habits. Features the writer cannot easily control
(function words, character-level patterns) carry the strongest
signal. Content words carry topic, not style, so most methods
downweight them.

## Feature categories

| Category | Examples | Signal strength |
| --- | --- | --- |
| Word frequencies | Relative frequencies of the most frequent words, dominated by function words (the, of, and, to, in) | Strongest. Foundation of Burrows' Delta. Mostly unconscious and topic-robust |
| Character n-grams | n=3..5 sequences incl. spaces and punctuation | Strong. Captures spelling, morphology, punctuation habits, robust cross-language |
| Lexical richness | TTR, MATTR (window ~500 words), MTLD, Yule's K/I, hapax legomena, mean word length | Moderate. Length-sensitive, use MATTR not raw TTR |
| Sentence/structure | Sentence length mean and variance, paragraph length, quotation density | Moderate. Genre-dependent |
| Punctuation | Frequency and distribution of each mark, comma style, exclamation, ellipses | Moderate to strong, underused |
| POS/syntax | POS n-grams, parse depth, dependency distance, passive voice, T-units | Moderate. Needs a tagger/parser, adds pipeline fragility |
| Idiosyncratic | Misspellings, spelling variants (clew/clue, wilfully), dialect terms, pet phrases | High precision when present, sparse. Broke the Unabomber case |
| Orthographic/markup | Capitalization, emoji, emoticons, markup habits, quote style, whitespace | Strong for social-media and informal text |

## Practical notes

- Function words beat content words because they are frequent,
  unconscious, and topic-light. Mosteller and Wallace resolved
  the disputed Federalist papers on words like whilst/while (1964).
- Hoover 2004 showed Delta improves by removing personal pronouns
  (they encode point of view, not author) and words dominated by
  a single text.
- Char n-grams are the standard language-agnostic baseline and
  capture much of the punctuation signal for free.
- Error patterns are the highest-precision single features: the
  Unabomber's clew and wilfully traced to 1940s-50s Chicago
  Tribune spelling reforms.
- Writeprints (Abbasi and Chen 2008) combines lexical, syntactic,
  structural, content-specific, and idiosyncratic groups, roughly
  300+ features, reaching ~94% at 100 authors on curated testbeds.
  Do not extrapolate that number to the wild.
