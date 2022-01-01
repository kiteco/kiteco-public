import kite.classification.svm as svm
import kite.classification.utils as utils

SIZE_CHAR_HASH_VEC = 20  # size of vector that characters will be hashed to
SIZE_WORD_HASH_VEC = 5  # size of vector that words will be hashed to
N_TOP_FREQ_WORDS = 50  # how many top words to retrieve from each training file


class PromptFeaturizer(svm.SVMFeaturizer):

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
        feat_vec.extend(utils.compute_special_ch_vec(text))
        feat_vec.extend(utils.compute_regex_vec(text))
        return feat_vec
