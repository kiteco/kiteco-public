from typing import Any, Dict, Set, List, NamedTuple

import logging
import pandas as pd

from sklearn.base import BaseEstimator
from sklearn.pipeline import Pipeline

from ..util.model import balance_populations, features_and_labels, train_logistic_model, logistic_weights


class ProModel(NamedTuple):
    estimator: BaseEstimator

    def get_pro_uids(self, users: pd.DataFrame) -> Set[str]:
        """
        :param users: users DataFrame, as returned by get_user_totals()
        :return: IDs of users who are categorized as being professional users
        """
        dataset = _get_dataset(users, 0)
        X, _ = features_and_labels(dataset)
        y_pred = self.estimator.predict(X)
        pro_ids = set([])
        for i, uid in enumerate(list(dataset.index)):
            if y_pred[i]:
                pro_ids.add(uid)
        return pro_ids


def train_pro_model(users: pd.DataFrame) -> ProModel:
    """
    :param users: as returned by get_user_totals()
    :return: pro model
    """
    surveyed = users[users.last_surveyed_time > 0]
    logging.debug(f"{len(surveyed)} users answered copilot survey")

    pro_professions = ['software_engineer',
                       'data_scientist',
                       'machine_learning_engineer']
    nonpro_professions = ['student', 'other']

    user_split = {
        'pro': set(surveyed[surveyed.last_surveyed_profession.isin(pro_professions)].index),
        'non-pro': set(surveyed[surveyed.last_surveyed_profession.isin(nonpro_professions)].index),
    }
    logging.debug("user split: {}".format({k: len(v) for k, v in user_split.items()}))

    # balance the pro and non-pro populations to keep bias from creeping into the classifier
    balanced = balance_populations(user_split)

    dataset = pd.concat([
        _get_dataset(users[users.index.isin(balanced['non-pro'])], 0),
        _get_dataset(users[users.index.isin(balanced['pro'])], 1),
    ])

    clf, scaler = train_logistic_model(dataset)
    logging.debug("logistic weights:")
    logging.debug(logistic_weights(clf, dataset))
    estimator = Pipeline([('scale', scaler),
                          ('classify', clf)])
    return ProModel(estimator=estimator)


def _user_features(users: pd.DataFrame, uid: str) -> Dict[str, Any]:
    user = users.loc[uid]

    # we currently don't use git metrics as features since that was added in March, so we can't classify users
    # before that
    return {
        "running_editors": user.running_editors / user.total_days,
        "work_hours": user.python_work_hours / user.python_events if user.python_events > 0 else 0.,
        "weekdays": user.python_weekdays / user.python_events if user.python_events > 0 else 0.,
        "events_per_day": user.total_events / user.total_days,
        "python_events": user.python_events / user.total_events,
        # "git_found": user.git_found / user.total_events,
        # "has_repo": user.has_repo / user.total_events,
        "is_darwin": int(user.os == "darwin"),
        "is_windows": int(user.os == "windows"),
        "is_linux": int(user.os == "linux"),
    }


def _get_dataset(users: pd.DataFrame, label: Any = 0) -> pd.DataFrame:
    dataset = {}
    for uid in list(users.index):
        feats = _user_features(users, uid)
        feats['label'] = label
        for f, val in feats.items():
            if f not in dataset:
                dataset[f] = {}
            dataset[f][uid] = val
    return pd.DataFrame(dataset)


