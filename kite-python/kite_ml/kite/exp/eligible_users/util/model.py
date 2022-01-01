from typing import Dict, List, Set

import logging
import numpy as np
import pandas as pd
import random

from sklearn.linear_model import LogisticRegression
from sklearn.metrics import accuracy_score
from sklearn.model_selection import train_test_split
from sklearn import preprocessing


def balance_populations(pops: Dict[str, Set[str]]) -> Dict[str, Set[str]]:
    """
    :param pops: a Dict of population name to a set of user IDs
    :return: a version in which the groups are balanced in size 
    """
    argmin = None
    min_count = 0
    for k, v in pops.items():
        if argmin is None or len(v) < min_count:
            argmin = k
            min_count = len(v)

    balanced = {}
    for k, v in pops.items():
        users = list(v)
        random.shuffle(users)
        balanced[k] = set(users[:min_count])
    return balanced


def _feature_col_names(df: pd.DataFrame) -> List[str]:
    cols = sorted(list(df.columns))
    cols.remove("label")
    return cols


def features_and_labels(dataset_df: pd.DataFrame) -> (np.ndarray, np.ndarray):
    """
    :param dataset_df:  contains N records with D columns containing features, and a 'label' column
    :return: numpy arrays containing the features (NxD) and labels (Nx1) - these can be fed directly into sklearn models
    """
    cols = _feature_col_names(dataset_df)
    features = dataset_df[cols].values
    labels = np.ravel(dataset_df[['label']].values)
    return features, labels


def train_logistic_model(dataset: pd.DataFrame) -> (LogisticRegression, preprocessing.StandardScaler):
    train, test = train_test_split(dataset)
    logging.info("{} train samples, {} test samples".format(len(train), len(test)))

    X_train, y_train = features_and_labels(train)
    scaler = preprocessing.StandardScaler().fit(X_train)
    clf = LogisticRegression(solver='lbfgs').fit(scaler.transform(X_train), y_train)

    X_test, y_test = features_and_labels(test)
    y_pred = clf.predict(scaler.transform(X_test))
    logging.info("logistic model accuracy: {}".format(accuracy_score(y_test, y_pred)))

    return clf, scaler


def logistic_weights(clf: LogisticRegression, dataset: pd.DataFrame) -> pd.Series:
    col_names = _feature_col_names(dataset)

    feature_vals = {}
    for name, val in zip(col_names, clf.coef_[0]):
        feature_vals[name] = val
    feature_vals["intercept"] = clf.intercept_[0]
    return pd.Series(feature_vals).sort_values(ascending=False)


