import collections
import numpy as np


def weighted_variance(outputs):
    return len(outputs) * np.var(outputs - np.mean(outputs))


class Node(object):
    """
    Represents a decision node of the form:
        if x[feature_index] < threshold then LEFT else RIGHT
    """
    def __init__(self, feature_index, threshold, left_child, right_child):
        self.feature_index = feature_index
        self.threshold = threshold
        self.left_child = left_child
        self.right_child = right_child

    def depth(self):
        left_depth = self.left_child.depth() if isinstance(self.left_child, Node) else 0
        right_depth = self.right_child.depth() if isinstance(self.right_child, Node) else 0
        return max(left_depth, right_depth) + 1


def walk(node):
    yield node
    if isinstance(node.left_child, Node):
        for subnode in walk(node.left_child):
            yield subnode
    if isinstance(node.right_child, Node):
        for subnode in walk(node.right_child):
            yield subnode


def drop(x, node):
    """Get the feature_index of the bin into which the given data point falls."""
    x = np.atleast_1d(x)
    cur = node
    while isinstance(cur, Node):
        if x[cur.feature_index] < cur.threshold:
            cur = cur.left_child
        else:
            cur = cur.right_child
    return cur


class DecisionTree(object):
    """
    Implements a generic decision tree suitable for when the splitting criteria is separate
    from the output criteria (for example, when minimizing a ranking loss)
    """
    def __init__(self, root, outputs, feature_size):
        assert isinstance(root, Node)
        self.root = root
        self.outputs = outputs
        self.feature_size = feature_size

    def __call__(self, x):
        """Get the output for the given data point."""
        if np.ndim(x) == 2:
            return np.array([self(xi) for xi in x])
        else:
            return self.outputs[drop(x, self.root)]

    def to_json(self):
        nodes = list(walk(self.root))
        nodes_json = []
        for node in nodes:
            json = dict(
                feature_index=node.feature_index,
                threshold=node.threshold)
            if isinstance(node.left_child, Node):
                json['left_child'] = nodes.index(node.left_child)
                json['left_is_leaf'] = False
            else:
                json['left_child'] = node.left_child
                json['left_is_leaf'] = True
            if isinstance(node.right_child, Node):
                json['right_child'] = nodes.index(node.right_child)
                json['right_is_leaf'] = False
            else:
                json['right_child'] = node.right_child
                json['right_is_leaf'] = True
            nodes_json.append(json)
        return dict(nodes=nodes_json, outputs=self.outputs, feature_size=self.feature_size, depth=self.root.depth())


def fit_skeleton(data, targets, max_depth=5, min_samples_leaf=3, criterion=weighted_variance):
    """
    Construct a tree by greedily minimizing the given criterion on the given dataset.
    """
    bins = []
    data = np.asarray(data)
    if data.ndim == 1:
        data = data[:, None]
    featuresize = data.shape[1]

    def split(indices, data, targets, curdepth):
        # Try each feature feature_index and each splitting point
        if len(data) < min_samples_leaf or curdepth >= max_depth:
            bins.append(list(zip(indices, data, targets)))
            return len(bins)-1

        # Find the best split
        best_cost = None
        best_split = None
        identical_features = True
        for j in range(featuresize):
            # Check all the unique values to decide the splitting point 
            col = sorted(set(data[:, j]))
            if len(col) > 1:
                identical_features = False 
            for i in range(len(col)-1):
                t = (col[i] + col[i+1]) / 2
                mask = data[:, j] < t
                left_targets = [targets[k] for k in np.flatnonzero(mask)]
                right_targets = [targets[k] for k in np.flatnonzero(~mask)]
                cost = criterion(left_targets) + criterion(right_targets)

                if best_cost is None or cost < best_cost:
                    best_cost = cost
                    best_split = (j, t)

        # This happens when all the features have the same value.
        if identical_features:
            bins.append(list(zip(indices, data, targets)))
            return len(bins)-1

        # Split the data and recurse
        j, t = best_split
        mask = data[:, j] < t
        left_targets = [targets[i] for i in np.flatnonzero(mask)]
        right_targets = [targets[i] for i in np.flatnonzero(~mask)]
        left_child = split(indices[mask], data[mask], left_targets, curdepth+1)
        right_child = split(indices[~mask], data[~mask], right_targets, curdepth+1)
        return Node(feature_index=j, threshold=t, left_child=left_child, right_child=right_child)

    # Start recursion
    root = split(np.arange(len(data)), data, targets, 0)
    return root, bins


def fit_least_squares(data, targets, learning_rate, newton_step=[], **kwargs):
    solver = lambda indices, targets: np.mean(targets)
    return fit(data, targets, solver, learning_rate, **kwargs) 

def fit_newton(data, targets, learning_rate, newton_step=[], **kwargs):
    def solver(indices, targets):
        return sum(targets) / sum(newton_step[i] for i in indices)
    return fit(data, targets, solver, learning_rate, **kwargs) 

def fit(data, targets, solver, learning_rate, **kwargs):
    """
    Fit a least squares regression tree (possibly with newton steps) to the given data and targets.
    """
    # Fit skeleton
    data = np.asarray(data)
    skeleton, bins = fit_skeleton(data, targets, **kwargs)

    # Fit outputs
    outputs = []
    for items in bins:
        indices, _, targets = zip(*items)
        outputs.append(solver(indices, targets) * learning_rate)

    return DecisionTree(skeleton, outputs, feature_size=data.shape[1])
