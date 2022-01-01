from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn import decomposition
from sklearn.feature_extraction.stop_words import ENGLISH_STOP_WORDS

from logging import getLogger, DEBUG, StreamHandler
from sys import stdout

handler = StreamHandler(stdout)
logger = getLogger()
logger.addHandler(handler)
logger.setLevel(DEBUG)


class NMF(object):
    """
    NMF uses non-negative matrix factorization for topic modeling.

    NMF first create a document-term matrix, A, and then finds
    W (document-topic matrix) and H (topic-term matrix)
    such that W * H ~ A.
    """
    def __init__(self, n_components=50, max_iter=200):
        self.n_components = n_components
        self.max_iter = max_iter

    def decompose(self, documents):
        """
        decompose creates the document-term matrix A and decomposes it to W * H
        """
        tfidf = TfidfVectorizer(stop_words=ENGLISH_STOP_WORDS,
                                lowercase=True,
                                strip_accents="unicode",
                                use_idf=True,
                                norm="l2",
                                min_df=5)
        A = tfidf.fit_transform(documents)
    
        if len(tfidf.vocabulary_) < 2:
            logger.warning("vocabulary size '%d' is too small" % len(tfidf.vocabulary_))
            return [], [] 

        model = decomposition.NMF(init="nndsvd",
                                  n_components=min(self.n_components, len(tfidf.vocabulary_)),
                                  max_iter=self.max_iter)
        W = model.fit_transform(A)
        H = model.components_ 
        return W, H
