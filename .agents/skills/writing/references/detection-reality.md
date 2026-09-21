# Detection reality check

Why detectors are not the audience and why "humanizer" paraphrase
is dead.

## Detector unreliability, measured

- Liang et al., Patterns 2023 (arXiv:2304.02819): 7 detectors on
  91 TOEFL essays, 61.3% average false-positive rate. 97% flagged
  by at least one detector. Non-native writing gets flagged. US
  8th-grade essays passed near-perfectly. Perplexity proxies
  penalize limited linguistic proficiency.
- OpenAI's own classifier: withdrawn July 2023, 26% true-positive,
  9% false-positive on its own challenge set.
- Weber-Wulff et al. 2023 (arXiv:2306.15666): 14 tools including
  Turnitin and GPTZero. 96% on human text, 74% on raw ChatGPT,
  42% after light manual edits. "Neither accurate nor reliable."
- Opace 2026: detection is mostly a length function. ~17% flag
  rate at 100-199 words vs ~96% on long documents. For comments,
  emails, and forum replies, detectors are near-blind anyway.
- Sadasivan et al. (arXiv:2303.11156): theoretical result, for a
  sufficiently good LM the best detector approaches random.

## Why paraphrase humanizers do not work

- SlopShape detects machine text at 98% from structure alone,
  unchanged after full rewording. Synonym swapping and word
  shuffling leave the shape intact.
- Dedicated evasion tools do beat detector-class instruments
  (DIPPER, TempParaphraser, adversarial paraphrasing at 80%+
  detection drop), but the output still reads machine-shaped to a
  human reader.
- Real humanization is rewriting structure and adding accountable
  specifics, which is indistinguishable from writing well. That
  is the practical sweet spot.

## Practical rules

- Write for human readers, not detectors. Short genres are
  detector-blind anyway.
- Never present a detector verdict as proof in either direction.
  Flagged text is not machine text. Passed text is not human.
- If a venue requires disclosure or forbids machine assistance,
  follow the rule. This skill is for matching the user's own
  voice, not laundering false authorship.
