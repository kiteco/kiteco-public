from typing import List, NamedTuple

import logging

import numpy as np
import tensorflow as tf

from sklearn.linear_model import LogisticRegression
from sklearn.metrics import roc_auc_score

from .feed import Feed
from .file_feeder import FileFeeder
from .model import Model


def train_logistic_model(sess: tf.Session, model: Model, train_feeder: FileFeeder, val_feeder: FileFeeder):
    """Train via scikit-learn because it seems to train logistic regression models more effectively"""
    train_data = _get_dataset(train_feeder)
    val_data = _get_dataset(val_feeder)

    logging.info("read {} train records and {} validation records".format(
        train_data.num_records(), val_data.num_records()))

    clf: LogisticRegression = LogisticRegression(solver='lbfgs').fit(train_data.features, train_data.labels)
    logging.info("sklearn classifier intercept: {}, weights: {}".format(clf.intercept_, clf.coef_))

    val_predictions = clf.predict_proba(val_data.features)[:, 1]
    auc = roc_auc_score(val_data.labels, val_predictions)
    logging.info("AUC: {}".format(auc))

    weights = _get_classifier_weights(clf)
    model.set_weights(sess, weights)


class _Dataset(NamedTuple):
    features: np.ndarray  # shape: num samples x feature depth
    labels: np.ndarray  # shape: num samples x 1

    def num_records(self) -> int:
        return len(self.labels)


def _get_dataset(feeder: FileFeeder) -> _Dataset:
    features: List[List[float]] = []
    labels: List[int] = []

    for _ in range(feeder.count()):
        sample = feeder.next()
        feed = Feed.from_samples([sample])

        contextual_features = feed.contextual_features[0]

        for i, comp_features in enumerate(feed.comp_features):
            # We assume the first element of the contextual features is the intercept (aka the features we already
            # augmented), so we discard it because sklearn already uses it
            assert np.abs(contextual_features[0] - 1.0) < 1e-9, \
                "first elem of contextual features should be intercept"
            total_features = contextual_features[1:] + comp_features
            features.append(total_features)

            labels.append(1 if i == sample.label else 0)

    return _Dataset(
        features=np.array(features),
        labels=np.array(labels),
    )


def _get_classifier_weights(clf: LogisticRegression) -> np.ndarray:
    return np.array([[clf.intercept_[0]] + clf.coef_.tolist()[0]]).T
