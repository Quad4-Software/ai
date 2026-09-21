# Adversarial stylometry

The field assumes unmodified text. Brennan, Afroz, and Greenstadt
broke that assumption in 2009-2012 and every result since has
deepened the problem.

## Attack classes and documented degradation

| Attack | How | Documented effect |
| --- | --- | --- |
| Obfuscation | Rewrite to hide your own style | Brennan et al. 2009: classifiers to random guessing. Replication: 14% accuracy vs 37% baseline (10 candidates) |
| Imitation | Mimic a target author's style | 67-91% attack success (Brennan et al.). Replication: 22% residual accuracy. Non-expert subjects, little prep |
| Round-trip translation | English to foreign to English MT | Weak in 2006 with bad MT. Modern NMT/LLM makes it practical. Flattens function-word and syntax signal |
| Machine paraphrase | LLM rewrite | Iterated paraphrasing monotonically destroys attribution signal (Ship of Theseus, ACL 2024). Cheapest effective attack |
| Homoglyph attacks | Unicode lookalikes | ~70% success vs AI-text detectors in multilingual tests (Findings EMNLP 2024) |
| Feature-targeted editing | Delete your discriminative features | Kacmarcik and Gamon 2006: ~14 edits per 1000 words cut attribution >83% |

## Key studies

- Rao and Rohatgi 2000: first suggested round-trip MT.
- Kacmarcik and Gamon 2006: shallow vs deep anonymization,
  classifier-driven feedback loop.
- Brennan, Afroz, Greenstadt (IAAI-09, TISSEC 2012): framework,
  human-subject attacks, released corpora (57 authors).
- Riddell-Juola replication: same conclusions, weaker effects,
  noted the original lacked a control group.
- PAN 2016 author masking (Potthast, Hagen, Stein): best
  obfuscator flipped ~47% of 44 verifiers' decisions.
- ALISON (Xing et al. 2024, arXiv 2402.00835): stylometric-feature
  obfuscation, defeats transformer attribution including on
  ChatGPT text.
- StyleRemix (Fisher et al., EMNLP 2024): LoRA modules perturb
  single style axes. AuthorMix corpus (30k texts, 14 authors).

## Defensive implications

- An attribution model doubles as an obfuscation linter:
  Anonymouth runs JStylo, reports which features expose you, and
  you edit them.
- Before trusting an attribution, ask whether the text could
  plausibly be translated, LLM-paraphrased, or edited. If yes,
  weight the result down sharply.
- No attribution method is robust to a competent adaptive
  adversary. PAN 2025-2026 Voight-Kampff evaluates detectors
  under exactly that condition.
