# Methods

## Distance methods (unsupervised)

- Burrows' Delta (2002): relative frequencies of the M most
  frequent words per text, z-scored against the corpus, mean
  absolute difference of z-scores. Lowest delta wins. Typical M
  100-5000. Reliable above ~5,000-word samples.
- Delta variants: Eder's Delta, Argamon's quadratic delta, cosine
  delta. Evert et al. 2017 showed cosine vector normalization is
  the decisive improvement. Hoover 2004: drop pronouns, use
  larger MFW.
- Zeta / Craig's Zeta: two-class contrast on marker-word usage,
  in stylo as oppose().
- Rolling stylometry: sliding-window attribution to find
  boundaries and ghostwritten passages (stylo rolling.stylo()).

## Classifiers (supervised, closed set)

- SVM on char n-grams or MFW: the durable strong baseline.
- kNN, Naive Bayes, Random Forest, Nearest Shrunken Centroids.
  All in stylo classify().
- Writeprints (Abbasi and Chen 2008): rich multi-category feature
  set, Karhunen-Loeve transforms, sliding window, pattern
  disruption, per-author feature selection.

## Verification methods (open world)

- Unmasking (Koppel, Schler, Bonchek-Dokow 2007): chunk both
  texts, cross-validate a linear SVM, iteratively delete the most
  discriminative features, track accuracy decay. Same-author
  pairs decay fast. Needs long texts, ~500-word chunks.
- General Imposters (Koppel and Winter 2014): compare the
  disputed text against rotating impostor sets, score how often
  the candidate wins. In stylo as imposters().
- Profile-based vs instance-based (Stamatatos 2009 survey).

## Embedding and neural methods

- LUAR (Rivera-Soto et al., EMNLP 2021, LLNL): transformer
  authorship embeddings, strong at hundreds of thousands of
  authors, cross-domain transfer uneven.
- LLM-era survey (arXiv 2408.08946) splits modern attribution
  into four problems: human-to-human, AI detection,
  which-LLM-or-human, and human/AI co-authorship.

## Compression-based

- Benedetto et al. 2002 (PRL): compressibility distance, early
  claims overstated.
- NCD (Li et al. 2004): NCD(x,y) = (C(xy) - min C) / max C.
- npc_gzip (Jiang et al., Findings ACL 2023): gzip + kNN,
  parameter-free, beat BERT on OOD sets. Cheap sanity baseline in
  ~20 lines.

## PAN shared task history (pan.webis.de, running since 2007)

- 82 shared tasks by 2026, 1,100+ submissions via TIRA.
- Attribution: 2011, 2012, 2018, 2019.
- Verification: 2013-2015 (cross-genre from 2015), 2020-2023
  (2022 was cross-discourse-type: essays vs emails vs texts).
- Author masking/obfuscation: 2016-2018.
- Multi-author style-change detection: annually since 2016.
- Voight-Kampff generative AI detection: 2024-2026, with LLMs
  instructed to mimic specific human authors plus surprise
  obfuscations.
- New in 2026: Text Watermarking, Reasoning Trajectory Detection.
