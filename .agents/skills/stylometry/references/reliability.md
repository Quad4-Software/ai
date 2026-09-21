# Reliability limits

## Sample size

- ~5,000 words per sample is the practical floor for attribution
  in English, German, Polish, Hungarian prose (Eder 2015). Latin
  prose ~2,500. The floor was method-independent across Delta,
  SVM, kNN, MFW, char n-grams, POS n-grams.
- Burrows 2002: Delta useful above ~1,500 words. ~100 words only
  narrows a candidate field.
- Sub-3,000-word samples produced >60% false attribution in
  Eder's tests. Short-text attributions deserve suspicion.
- Verification also needs length: Unmasking wants multiple
  ~500-word chunks per document.

## Confounds

- Genre/register mismatch is the top confound (Burrows flagged it
  in 2002). PAN 2015 made cross-genre explicit and accuracy
  dropped hard.
- Topic leakage: content features smuggle topic into style. Even
  function-word profiles shift across topics.
- Temporal drift: authors change over a career.
- Collaborative editing: ghostwriters, editors, house style, and
  copyediting blur the fingerprint.
- Open-set reality: published accuracies are closed-set.
  Narayanan et al. 2012 (IEEE S&P): 100k blog authors, ~20%
  top-1, ~35% top-20. Abstention pushed precision >80% at half
  recall. At internet scale, stylometry is a shortlist generator,
  not a matcher.
- Demographics correlate with style (dialect, L1 interference,
  age), which helps profiling and biases attribution.

## Legal status (US)

- Courts treat stylometry like handwriting comparison: admitted
  for describing similarities, excluded for ultimate authorship
  opinion (US v. Van Wyk 2000, US v. Zajac 2010).
- Daubert problems: no established error rates, no single
  accepted methodology, results depend on corpus and features.
- Germany is more accepting: BKA Autorenerkennung unit (since
  ~1990) does style-and-error analysis with the KISTE system.
- Cautionary tale: Donald Foster. Right on Primary Colors (1996),
  retracted his Funeral Elegy attribution (2002), wrongly
  implicated Hatfill in the anthrax case (settled suit, 2007).

## The LLM-era problem

- Style transfer is a commodity: any text can be paraphrased,
  mimicked, or co-written, degrading attribution and verification.
- AI-text detectors are a different, weaker technology. OpenAI
  retired its classifier in 2023. Liang et al. 2023: 61.3% false
  positives on non-native English essays.
- Watermarking is the proposed countermeasure. PAN added a Text
  Watermarking task in 2026. Robustness still being benchmarked.
- Net: stylometry still works on unmodified human text of
  sufficient length, but every negative result must be read as
  possibly adversarial.

## Confidence tiers for reporting

- State the setup: closed or open set, sample sizes, genre match,
  method, feature set.
- Report rank and margin, not identity.
- Label outputs: consistent-with, inconsistent-with, inconclusive.
