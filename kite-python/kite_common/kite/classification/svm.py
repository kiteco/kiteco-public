from abc import ABCMeta, abstractmethod
from enum import Enum

from sklearn.svm import NuSVC
from sklearn.svm import SVC
from sklearn import cross_validation

GAMMA = 0.05  # gamma value for SVM RBF kernel
NU = 0.1  # soft margin


class Kernel(Enum):
    linear = 1
    rbf = 2


class SVMFeaturizer(metaclass=ABCMeta):

    @abstractmethod
    def features(self):
        pass


class SVMClassifier(object):

    def __init__(self, kernel, featurizer, svm_type='nu_svm'):
        self.kernel = kernel
        self.featurizer = featurizer
        self.svm_type = svm_type
        self.model = None

        if self.svm_type == 'nu_svm':
            """The NuSVC model is used because it allows us to specify a value 'nu'
            which governs how strictly we require the training data be correctly
            classified. A value of 0.1 means that we allow at most 10 percent of
            the training data to be incorrectly classified in choosing the separating
            hyperplane. This 'soft margin' is useful to prevent overfitting.
            Currently supports linear and rbf kernels.
            """
            self.model = NuSVC(kernel=self.kernel, nu=NU, probability=True)
        else:
            self.model = SVC(kernel=self.kernel, probability=True)

        if self.kernel == 'rfb':
            # initialize model with kernel type
            self.model.gamma = GAMMA

    def train(self, data, labels):
        """Given training examples and an array of
        labels, one per training example, computes feature vector
        out of each training example and trains a svm model.
        """
        # compute feature vectors from data
        feat_vecs = []
        for line in data:
            feat_vecs.append(self.featurizer.features(line))

        self.model.fit(feat_vecs, labels)

    def classify(self, text):
        """Given a text input, tests it against the model to see if it is
        classified as an error (if prediction == +1) or
        non-error (if prediction == -1).
        """
        if self.featurizer is None:
            raise ValueError('featurizer is required for classify()')
        if self.model is None:
            raise ValueError('model is required for classify()')

        features = self.featurizer.features(text)
        return self.model.predict([features])

    def predict_proba(self, text):
        """Given an input, tests it against the model to see how likely
        the two titles are un-related and related to each other.
        """
        if self.featurizer is None:
            raise ValueError('featurizer is required for classify()')
        if self.model is None:
            raise ValueError('model is required for classify()')

        features = self.featurizer.features(text)
        return self.model.predict_proba([features])

    def cross_validate(self, data, labels, fold):
        """Cross validation to test the performance of the model"""
        feat_vecs = []
        for line in data:
            feat_vecs.append(self.featurizer.features(line))

        scores = cross_validation.cross_val_score(
            self.model,
            feat_vecs,
            labels,
            cv=fold)
        return sum(scores) / len(scores)

    def __getstate__(self):
        """Remove featurizer before pickling"""
        copy = self.__dict__.copy()
        del copy['featurizer']
        return copy
