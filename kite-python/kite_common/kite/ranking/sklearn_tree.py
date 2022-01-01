import numpy as np
from sklearn.tree import DecisionTreeRegressor

class SKLearnDecisionTree(object):
    """
    A wrapper of the Scikit-learn decision tree regressor class. We use a wrapper to provide a set of APIs that is
    consistent with the one provided by DecisionTree.
    """
    def __init__(self, tree):
        self.tree = tree

    def __call__(self, x):
        return self.tree.predict(np.atleast_2d(x))

    def to_json(self):
        pass


def fit_least_squares(data, targets, learning_rate, **kwargs):
    tree = DecisionTreeRegressor(max_features=1, random_state=0, **kwargs)
    tree.fit(data, targets)
    return SKLearnDecisionTree(tree) 
