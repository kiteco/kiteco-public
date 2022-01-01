import os
import json
import numpy as np

from sklearn.linear_model import LogisticRegression
from sklearn import cross_validation

class LogisticRegressionClassifier(object):
    def __init__(self, featurizer):
        self.featurizer = featurizer
        # we use the default values for initialization.
        # can be changed if we need to pass in customized params.
        self.model = LogisticRegression()

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
        """Given a feature vector f = (query, package), test it against
        the model to see if the query refers to the package.
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

    def export(self, path):
        model_params = {
                'ScorerType': 'logistic_regression',
                'Scorer': {
                    'coefs': self.model.coef_.tolist(),
                    'bias': self.model.intercept_.tolist()}
                }
        with open(path, 'w') as model_file:
            json.dump(model_params, model_file)
