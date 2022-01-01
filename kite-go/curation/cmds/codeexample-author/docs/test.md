# Test doc

This is a test doc to demonstrate compilation of Markdown-formatted curation documents to HTML.

 - this should
 - be a
 - list

For example, a fancy-free artist avoids pure mathematics or logic not because he understands them and could say something about them if he wished, but because he instinctively inclines toward other things. Such instinctive and violent inclinations and disinclinations are signs by which you can recognize the pettier souls. In great souls and superior minds, these passions are not found. Each of us is merely one human being, merely an experiment, a way station. But each of us should be on the way toward perfection, should be striving to reach the center, not the periphery. Remember this: one can be a strict logician or grammarian, and at the same time full of imagination and music. […] The kind of person we want to develop, the kind of person we aim to become, would at any time be able to exchange his discipline or art for any other.

Apparently this example is giving us some trouble:

```
# PRELUDE
from collections import deque

# CODE
d = deque([1, 2, 3])
d.reverse()
print d

# OUTPUT
deque([3, 2, 1])
```

You can also syntax highlight as "good" or "bad" using the language tag:

```bad
def fib(n):
    return 1 if n < 2 else fib(n-1) + fib(n-2)
```

And you can show what a good example looks like instead:

```good
def fib(n):
	prev = 0
	curr = 1
	for i in range(n):
		next = prev + curr
		prev = curr
		curr = next
	return curr
```

All the code is automatically syntax highlighted.  Here's a longer example to demonstrate more complex syntax highlighting:

```
import kite.classification.svm as svm
import kite.classification.utils as utils

SIZE_CHAR_HASH_VEC = 20  # size of vector that characters will be hashed to
SIZE_WORD_HASH_VEC = 5  # size of vector that words will be hashed to
N_TOP_FREQ_WORDS = 50  # how many top words to retrieve from each training file


class ErrorFeaturizer(svm.SVMFeaturizer):

    def __init__(self, most_freq_words):
        self.most_freq_words = most_freq_words

    def features(self, text):
        """ Computes feature vector representing distribution of
        chars, words, and most freq words (across all lang errors).
        This feature vector is then used for classification.
        Three main components to the feature vector:
        1. Distribution of the characters in the text, hashed into a fixed
           slice of length classifier.SizeCharHashVec.
        2. Distribution of the words in the text, hashed into a fixed slice
           of length classifier.SizeWordHashVec.
        3. Presence of most frequently encountered words across the training set.
        """

        text = text.lower()
        feat_vec = utils.compute_char_hash_vec(text, SIZE_CHAR_HASH_VEC)
        feat_vec.extend(utils.compute_word_hash_vec(text, SIZE_WORD_HASH_VEC))
        feat_vec.extend(
            utils.compute_most_freq_words_feature_vec(
                text,
                self.most_freq_words))
        return feat_vec
```

Looks like it works!
