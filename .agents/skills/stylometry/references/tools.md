# Tools and quick recipes

## Fastest path: faststylometry (Python)

```bash
pip install faststylometry
```

```python
from faststylometry import load_corpus_from_folder
from faststylometry import tokenise_remove_pronouns_en
from faststylometry import calculate_burrows_delta

corpus = load_corpus_from_folder("corpus/")   # author_title.txt
corpus.tokenise(tokenise_remove_pronouns_en)
for doc in test_corpus:
    print(calculate_burrows_delta(corpus, doc))
```

Delta near 0 means strong stylistic similarity. Above ~1 usually
means different authors on matched genres.

## Broad metrics: pystylometry (Python)

```bash
pip install pystylometry        # core lexical metrics
pip install pystylometry[all]   # adds spaCy syntax and viz
```

Covers Burrows' Delta, cosine delta, Zeta, Kilgarriff
chi-squared, MinMax, NCD, MATTR, MTLD, Yule's K/I, POS ratios,
punctuation, char n-grams, dialect markers, drift windows.

## Field standard: stylo (R)

```r
install.packages("stylo")
library(stylo)
# texts in ./corpus/ named author_title.txt, then:
stylo(gui = FALSE)     # cluster analysis / Delta
classify(gui = FALSE)  # supervised: Delta, SVM, kNN, NB, NSC
oppose(gui = FALSE)    # two-corpus contrast (Zeta)
```

## DIY baselines (Python)

```python
# char n-gram SVM, the durable baseline
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.svm import LinearSVC
vec = TfidfVectorizer(analyzer="char", ngram_range=(3,5), min_df=2)
X = vec.fit_transform(train_texts)
clf = LinearSVC().fit(X, train_labels)
pred = clf.predict(vec.transform([unknown_text]))
```

```python
# gzip NCD distance, parameter-free (npc_gzip recipe)
import gzip
def ncd(x, y):
    cx, cy = len(gzip.compress(x)), len(gzip.compress(y))
    cxy = len(gzip.compress(x + y))
    return (cxy - min(cx, cy)) / max(cx, cy)
```

## Java stack (forensic-oriented)

- JStylo (github.com/psal/jstylo): Writeprints-style feature
  sets, Weka backend, needs JGAAP jar. Use branch 2.3.0 for the
  UI per the project's own recommendation.
- Anonymouth: feeds JStylo analysis back as edit suggestions for
  obfuscation. Needs a reference corpus.
- JGAAP (github.com/evllabs/JGAAP): GUI attribution lab.

## Corpora for testing

- Federalist Papers (tiny, classic).
- Blog Authorship Corpus (Schler et al. 2006): ~19k bloggers.
- Enron email corpus, IMDb62, CCAT50, Guardian corpus.
- PAN datasets on Zenodo.
- Adversarial: Extended Brennan-Greenstadt corpus,
  Riddell-Juola corpus, AuthorMix.
- Wikipedia sockpuppet corpus (Solorio et al.).

## Comparison checklist

1. Same language, same genre, comparable era.
2. >= 5,000 words per side, or downgrade to inconclusive.
3. Keep spelling errors and punctuation, strip only markup noise.
4. Baseline: delta between two known-same-author texts first.
5. Report ranking, margin, and sample sizes. Recommend
   corroboration.
