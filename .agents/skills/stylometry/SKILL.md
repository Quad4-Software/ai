---
name: stylometry
description: >
  This skill covers stylometry and authorship attribution: what
  stylometric features capture (function words, n-grams,
  punctuation, lexical richness, syntax), the classic and modern
  methods (Burrows' Delta, SVM, transformer embeddings, Unmasking,
  compression distance), free tooling (stylo, JStylo, JGAAP,
  faststylometry, pystylometry), and adversarial stylometry
  (imitation, obfuscation, translation and LLM paraphrase attacks
  with documented accuracy degradation). Use it for comparing
  writing styles between texts, assessing authorship claims,
  checking sockpuppet or ghostwriting suspicions, understanding
  deanonymization risk, or building an obfuscation plan. Includes
  reliability limits, Daubert-era legal status, and LLM-era
  caveats.
metadata:
  sources:
    - https://github.com/computationalstylistics/stylo
    - https://github.com/psal/jstylo
    - https://pan.webis.de/shared-tasks.html
    - https://github.com/fastdatascience/faststylometry
---

## When to use this skill

- Comparing two texts or a corpus for same-author evidence.
- Evaluating an authorship claim (pseudonym, ghostwriter,
  sockpuppet).
- Assessing deanonymization risk of someone's public writing.
- Building or testing an obfuscation plan for anonymous
  publication.
- Knowing when NOT to trust a stylometric claim (short texts,
  genre mismatch, LLM paraphrase, open-set problems).

## Ground rules

- Stylometry produces leads and corroboration, not identity proof.
  Treat every result as one signal to combine with non-textual
  evidence.
- Below roughly 5,000 words per sample, attribution accuracy
  degrades sharply (Eder 2015). Below that, report uncertainty
  loudly or refuse.
- Match genre and register before comparing. Genre mismatch is the
  single biggest confound. Function-word methods reduce but do not
  remove topic leakage.
- Closed-set accuracy claims (N candidate authors) do not transfer
  to open-set questions. The true author may not be in your
  candidate list.
- Assume the subject may be adversarial. Manual obfuscation drives
  standard methods to near-chance accuracy, and LLM paraphrase
  makes obfuscation near-free.

## Task taxonomy

- Attribution: pick the author from a closed candidate set.
- Verification: decide whether two texts share an author (open
  world). Harder and more realistic. PAN ran verification
  2013-2015, 2020-2023.
- Style-change detection: locate author boundaries inside one
  document. PAN multi-author style analysis, running since 2016.
- Profiling: infer author demographics rather than identity.

## Quick comparison workflow

1. Assemble texts: same genre, same language, >= 5,000 words each
   if possible. Note editing history (collaborative docs mix
   styles).
2. Normalize mechanically: encoding, whitespace, quotes, but keep
   spelling errors and punctuation intact (they are signal).
3. Run Burrows' Delta over 500-2000 most frequent words, or a
   char n-gram SVM. See references/tools.md for copy-paste
   commands.
4. Sanity-check: distance between two texts by the same known
   author gives a calibration floor. Report a ranking, not a
   verdict.
5. Check adversarial surface: could the text have been
   paraphrased, translated, or edited? See references/adversarial.md.

## Tool map (Sept 2026)

| Tool | Lang | Best for | Notes |
| --- | --- | --- | --- |
| stylo | R | Full pipeline, Delta, classify, imposters | CRAN, field standard |
| stylo2gg | R | ggplot2 visualization of stylo output | GitHub only, small |
| JStylo | Java | Writeprints feature sets, forensic workflows | Branch 2.3.0 for UI |
| Anonymouth | Java | Obfuscation guidance, edit suggestions | Ships with JStylo |
| JGAAP | Java | Non-expert GUI attribution | Duquesne EVL Lab |
| faststylometry | Python | Quick Burrows' Delta between texts | MIT, calibrated proba |
| pystylometry | Python | 50+ metrics: Delta, Zeta, NCD, MATTR | MIT |
| pystyl | Python | Early stylo port | Unmaintained |
| NLTK + sklearn | Python | DIY char n-gram SVM, unmasking | See tools.md |
| npc_gzip | Python | Parameter-free NCD classification | gzip + kNN |

## Reference files

- `references/features.md` - what stylometry measures and which
  features carry the strongest signal.
- `references/methods.md` - Delta and variants, classifiers,
  embeddings, Unmasking, imposters, compression, PAN history.
- `references/adversarial.md` - imitation, obfuscation,
  translation and LLM attacks with measured degradation.
- `references/reliability.md` - sample-size floors, confounds,
  legal status, LLM-era problems.
- `references/cases.md` - forensic and literary case history,
  including failures.
- `references/tools.md` - install and run recipes for a fast
  two-text comparison.

## Pitfalls

- Never report a stylometric match as identification. Report
  ranking and margin, then demand corroborating evidence.
- A strong result on short text is noise, not skill. Sub-3,000
  word samples produced >60% false attribution in Eder's tests.
- Collaborative or copy-edited text dilutes signal toward house
  style. Wikipedia, corporate docs, and newsroom prose are the
  worst case.
- LLM paraphrase destroys style signal cheaply.
- AI-text detectors are not stylometry and are unreliable (61%
  false positives on non-native English writing, Liang et al.
  2023). Do not substitute one for the other.
