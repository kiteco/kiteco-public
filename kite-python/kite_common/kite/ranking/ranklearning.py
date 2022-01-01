import numpy as np
import scipy.spatial

import kite.ranking.tree as tree
import kite.ranking.sklearn_tree as sklearn_tree

from copy import deepcopy

class Dataset(object):
    """
    Represents a set of features together with ground truth relations, which are pairs of the form (i, j)
    specifying that the i-th item should be ranked above the j-th item.
    """
    def __init__(self, labels, queries, options):
        """Create a dataset with feature labels (i.e., feature names) and a list of queries."""
        self.queries = queries
        self.featurelabels = labels
        self.options = options
        self.nd = 0
        if len(self.queries) != 0:
            for query in self.queries:
                self.nd = query.features.shape[-1] 

    def features(self):
        return np.vstack([query.features for query in self.queries])

    def num_relations(self):
        return sum(len(q.relations) for q in self.queries)
    

class Query(object):
    """
    Query represents the training data and relevance scores for a query. 
    """
    def __init__(self, query_info, features, relevance, example_ids):
        # basic info about the query
        self.info = query_info

        # feature vectors
        self.features = np.asarray(features) 

        # relevance scores of the feature vectors
        self.relevance = np.array(relevance) 

        # example ids
        self.example_ids = example_ids 

        # relation (i, j) indicates that item i should rank above item j
        self.relations = relations_from_scores(self.relevance) 


def listify(f):
    """A decorator that converts generators to functions that return lists."""
    return lambda *args, **kwargs: list(f(*args, **kwargs))


class Normalizer(object):
    def __init__(self, offset, scale):
        self.offset = offset
        self.scale = scale

    def __call__(self, x):
        return (x + self.offset) * self.scale


def minmax_normalizer(x, axis=0):
    """Normalize x to min=0, max=1."""
    if len(x) <= 1:
        return x
    lo, hi = np.min(x, axis=axis), np.max(x, axis=axis)
    s = hi - lo
    if np.isscalar(s):
        if s < 1e-8:
            s = 1.
    else:
        s[s < 1e-8] = 1.
    return Normalizer(-lo, 1./s)


def normalize_minmax(x, axis=0):
    return minmax_normalizer(x, axis)(x)


def meanstd_normalizer(x, axis=0):
    """Normalize x to mean=0, variance=1."""
    if len(x) <= 1:
        return x
    m = np.mean(x, axis=axis)
    s = np.std(x - m, axis=axis)
    if np.isscalar(s):
        if s < 1e-8:
            s = 1.
    else:
        s[s < 1e-8] = 1.
    return Normalizer(-m, 1./s)


def normalize_meanstd(x, axis=0):
    return meanstd_normalizer(x, axis)(x)


@listify
def relations_from_ranks(ranks):
    """Given a list containing ranks for a set of data points, return the set of pairs
    (i, j) such that rank(i) < rank(j)."""
    for i, ri in enumerate(ranks):
        for j, rj in enumerate(ranks[:i]):
            if ri < rj:
                yield (i, j)
            elif rj < ri:
                yield (j, i)


@listify
def relations_from_scores(scores):
    """Given a list containing relevance scores for a set of data points, return the set of pairs
    (i, j) such that relevance(i) < relevance(j)."""
    for i, ri in enumerate(scores):
        for j, rj in enumerate(scores[:i]):
            if ri > rj:
                yield (i, j)
            elif rj > ri:
                yield (j, i)


def rank(features, scorer):
    """Compute scores for each of the specified data points and return the ordering of
    the features under the given score function.
    """
    scores = [scorer(x) for x in features]
    return ranks_from_scores(scores)


def ranks_from_ordering(ordering):
    """Convert an ordering vector into a vector of ranks."""
    ranks = [0] * len(ordering)
    for r, x in enumerate(ordering):
        ranks[x] = r
    return ranks


def ranks_from_scores(scores):
    """Return the ordering of the scores"""
    return sorted(range(len(scores)), key=lambda i: scores[i], reverse=True)


def compute_dataset_loss(queries, scorer, loss):
    cost = .0
    for query in queries:
        scores = scorer(query.features)
        cost += sum(loss(scores[i], scores[j]) for i, j in query.relations)
    return cost 


def compute_dataset_irscore(queries, scorer, ir):
    return sum(ir(query.relevance, scorer(query.features)) for query in queries) / len(queries)


class HingeLoss(object):
    def __call__(self, score_i, score_j): 
        """Compute the hinge loss for the given data point."""
        return max(0., 1. - score_i + score_j)

    def gradient(self, score_i, score_j):
        """Compute the gradient of the loss function w.r.t score[i] and
        score[j]. s_i and s_j must have the relationship that data_i is
        more relevant to the query than s_j.
        """
        if score_i < score_j + 1.:
            return -1., 1.
        return 0., 0.


class ZeroOneLoss(object):
    def __call__(self, score_i, score_j):
        """Compute the loss for the given scoring function on the given dataset."""
        return float(score_j >= score_i)

    def gradient(self, score_i, score_j):
        raise NameError('Zero-one loss is not differentiable.')


class CrossEntropyLoss(object):
    def __init__(self, sigma=1):
        """Initialize a corss-entropy loss function. 

        According to "From RankNet to LambdaRank to LambdaMART: An Overview", the choice of
        sigma doesn't affect the results. Therefore, we set it to 1 as the default value.
        """
        self.sigma = sigma

    def __call__(self, score_i, score_j):
        """Compute the cross entropy against the targets for the given scoring
        function on the given dataset.
        """
        return np.log1p(np.exp(score_j - score_i))

    def gradient(self, score_i, score_j):
        """Compute the gradient of the loss function w.r.t to score_i and score_j.
        It is assumed that data_i should rank higher than data_j.
        """
        d = -self.sigma/(1. + np.exp(self.sigma*(score_i - score_j)))
        return d, -d


class NDCG(object):
    """NDCG provides the API to compute scores related to the 'normalized discounted cumulative gain'.

    Note that NDCG is not used as a loss function, bur rather it is used as a weighting function
    for the gradient of the loss function.
    """
    def __init__(self, cutoff=10):
        # labels are the relevance labels for the code examples.
        # The value of a label is between 0 and 4 in our system.
        self.cutoff = cutoff 

    def __call__(self, labels, scores):
        ranks = ranks_from_scores(scores)
        return sum(self.dcg(i, labels[ranks[i]]) for i in range(min(self.cutoff, len(ranks)))) / self.max_dcg(labels)
    
    def dcg(self, rank, label):
        """Compute the IR score (DCG) contributed by an entry."""
        if rank >= self.cutoff:
            return 0
        return (np.power(2, label) - 1) / np.log(rank + 2)

    def delta(self, label_i, rank_i, label_j, rank_j):
        """Compute delta dcg by swapping (i, j)."""
        dcg_old = self.dcg(rank_i, label_i) + self.dcg(rank_j, label_j)
        dcg_new = self.dcg(rank_j, label_i) + self.dcg(rank_i, label_j)

        return abs(dcg_new - dcg_old)

    def max_dcg(self, labels):
        sorted_labels = sorted(labels, reverse=True)
        return sum(self.dcg(i, l) for i, l in enumerate(sorted_labels[:self.cutoff]))


class Rbf(object):
    """Represents a radial basis function."""
    def __init__(self, gamma):
        self.gamma = gamma

    def __call__(self, a, b):
        aa = np.atleast_2d(a)
        bb = np.atleast_2d(b)
        distances = scipy.spatial.distance.cdist(aa, bb, 'sqeuclidean')
        if np.ndim(a) < 2 or np.ndim(b) < 2:
            distances = np.squeeze(distances)
        return np.exp(self.gamma * distances)


class LinearScorer(object):
    """
    Represents a score function of the form w' * x
    """
    def __init__(self, w):
        self.w = np.asarray(w).astype(float)

    @property
    def parameters(self):
        return self.w

    def __call__(self, x):
        return np.dot(x, self.w)

    def to_json(self):
        return {"Weights": self.w.tolist()}


class LinearRankLearner(object):
    """
    Learns to rank using a linear scorer function. 
    """
    def __init__(self, init_weight, nd):
        self.nd = nd 
        if nd != len(init_weight):
            error = 'length of init weight %d not match with feature length %d' % (len(init_weight), nd)
            exit(error)
        self.current = LinearScorer(init_weight)

        # previous_state is a deepcopy of the object itself
        self.previous_state = None 

    def gradient(self, features, labels, relations, ir, loss):
        # gather model scores for computing gradients
        scores = self.current(features)

        # ranks is a list of the data indexes ranked based on the data scores
        ranks = ranks_from_scores(scores)

        # normalization term for NDCG
        if ir != None:
            n = ir.max_dcg(labels)

        # compute gradients
        gradient = np.zeros(self.nd)
        for i, j in relations:
            delta_ir = 1
            if ir != None:
                delta_ir = ir.delta(labels[i], ranks.index(i), labels[j], ranks.index(j)) / n
            dloss_dsi, dloss_dsj = loss.gradient(scores[i], scores[j])
            gradient += delta_ir * np.dot([dloss_dsi, dloss_dsj], [features[i], features[j]])
        return gradient

    def perturb(self, delta):
        self.current.w += delta

    def step(self, queries, loss, ir, learning_rate=.1):
        # make a deepcopy of the object so that if we need to reverse the step,
        # which happens when the cost increases or objective decreases, we
        # can easily roll back.
        self.previous_state = deepcopy(self) 

        gradient = np.zeros(self.nd)
        for query in queries:
            gradient += self.gradient(query.features, query.relevance, query.relations, ir, loss) 

        gradient = gradient / np.linalg.norm(gradient)
        delta = -gradient * learning_rate
        self.perturb(delta)
        return delta 


class KernelScorer(object):
    """
    Represents a score function of the form sum(a_i * k(x, data_i)) where data_i is the i-th data
    point, a_i is the i-th coefficient, and k is a kernel function.
    """
    def __init__(self, data, coefs, kernel=np.inner):
        self.data = data
        self.coefs = coefs
        self.kernel = kernel

    @property
    def parameters(self):
        return self.coefs

    def __call__(self, x):
        return np.dot(self.kernel(x, self.data), self.coefs)

    def to_json(self):
        return {
            "Gamma": float(self.kernel.gamma),
            "Support": self.data.tolist(),
            "Coefs": self.coefs.tolist(),
        }


class KernelRankLearner(object):
    """
    Learns to rank using a kernel scoring function. The training labels are constraints of
    the form "x_i should be ranked higher than x_j"
    """
    def __init__(self, features, kernel=np.inner):
        self.kernel = kernel
        self.all_features = features
        self.current = KernelScorer(features, np.zeros(len(features)), self.kernel)
        self.previous_state = None

    def gradient(self, features, labels, relations, ir, loss):
        # ranks is a list of indices of the data ranked based on the model scores
        scores = self.current(features)
        ranks = ranks_from_scores(scores)

        # normalization term for NDCG
        if ir != None:
            n = ir.max_dcg(labels)

        k = self.kernel(self.all_features, features)
        score_gradients = np.zeros(len(features))
        for i, j in relations:
            delta_ir = 1.
            if ir != None:
                delta_ir = ir.delta(labels[i], ranks.index(i), labels[j], ranks.index(j)) / n
            dloss_dsi, dloss_dsj = loss.gradient(scores[i], scores[j])
            score_gradients[i] += dloss_dsi * delta_ir
            score_gradients[j] += dloss_dsj * delta_ir
        return np.dot(k, score_gradients)

    def perturb(self, delta):
        self.current.coefs += delta

    def step(self, queries, loss, ir, learning_rate=.1):
        # make a deepcopy of the object so that if we need to reverse the step,
        # which happens when the cost increases or objective decreases, we
        # can easily roll back.
        self.previous_state = deepcopy(self)
        gradient = np.sum(
                [self.gradient(query.features, query.relevance, query.relations, ir, loss) 
                    for query in queries], axis=0)
        gradient = gradient / np.linalg.norm(gradient)
        delta = -gradient * learning_rate
        self.perturb(delta)
        return delta 


class TreeEnsembleScorer(object):
    """
    Represents a score function of the form sum(fi(x)) where each fi is a decision.
    """
    def __init__(self, trees):
        self.trees = trees

    def __call__(self, x):
        if len(self.trees) == 0:
            return np.zeros(x.shape[0]) 
        return np.sum([t(x) for t in self.trees], axis=0)

    def to_json(self):
        return dict(trees=[t.to_json() for t in self.trees])


class BoostedTreeLearner(object):
    """
    Trains a boosted decision tree score function using the lambdamart algorithm.
    """
    def __init__(self, queries, subsample=.5, use_sklearn_trees=False, **tree_options):
        self.use_sklearn_trees = use_sklearn_trees 
        self.subsample = subsample
        self.tree_options = tree_options
        self.current = TreeEnsembleScorer([])
        self.current_scores = list() 
        for q in queries:
            self.current_scores.append(np.zeros(len(q.features)))
        self.previous_state = None

    def derivative_preprocess(self, i, features, labels, relations, ir, loss):
        raw_gradients_i = []
        raw_gradients_j = []
        raw_irs = []

        scores = self.current_scores[i]
        ranks = ranks_from_scores(scores)

        # normalization term for NDCG
        if ir != None:
            n = ir.max_dcg(labels)

        for i, j in relations:
            delta_ir = 1.
            if ir != None: 
                delta_ir = ir.delta(labels[i], ranks.index(i), labels[j], ranks.index(j)) / n
            dloss_dsi, dloss_dsj = loss.gradient(scores[i], scores[j])
            raw_irs.append(delta_ir)
            raw_gradients_i.append(dloss_dsi)
            raw_gradients_j.append(dloss_dsj)

        return raw_gradients_i, raw_gradients_j, raw_irs

    def gradient(self, raw_gradients_i, raw_gradients_j, weights, n, relations):
        score_gradients = n * [0]
        for k, (i, j) in enumerate(relations):
            score_gradients[i] += raw_gradients_i[k] * weights[k]
            score_gradients[j] += raw_gradients_j[k] * weights[k]
        return score_gradients

    def hessian(self, raw_gradients_i, raw_gradients_j, weights, n, relations, sigma):
        score_hessians = n * [0] 
        for k, (i, j) in enumerate(relations):
            rho = abs(raw_gradients_i[k] / sigma)
            hessian = weights[k] * sigma * sigma * rho * (1 - rho)
            score_hessians[i] += hessian
            score_hessians[j] += hessian

        return score_hessians

    def step(self, queries, loss, ir, learning_rate=.1):
        self.previous_state = deepcopy(self)

        score_gradients = [] 
        score_hessians = []

        for i, query in enumerate(queries):
            raw_gradients_i, raw_gradients_j, raw_irs = self.derivative_preprocess(
                    i, query.features, query.relevance, query.relations, ir, loss)

            # Compute gradient of loss with respect to scores and the newton step
            score_gradients.extend(
                    self.gradient(raw_gradients_i, raw_gradients_j, raw_irs, len(query.features), query.relations))
            if isinstance(loss, CrossEntropyLoss):
                score_hessians.extend(
                    self.hessian(raw_gradients_i, raw_gradients_j, raw_irs, len(query.features), query.relations, loss.sigma))

        score_gradients = np.asarray(score_gradients)
        score_hessians = np.asarray(score_hessians)

        # Minimize loss function
        score_gradients *= -1

        # Fit a tree to the gradient
        features = np.vstack([query.features for query in queries])

        if self.use_sklearn_trees:
            t = sklearn_tree.fit_least_squares(features, score_gradients, learning_rate, **self.tree_options)
        else:
            if len(score_hessians) != 0:
                t = tree.fit_newton(features, score_gradients, learning_rate, score_hessians, **self.tree_options)
            else:
                t = tree.fit_least_squares(features, score_gradients, learning_rate, **self.tree_options)

        # Update the scores (cached for efficiency)
        for i, q in enumerate(queries):
            for j, row in enumerate(q.features):
                self.current_scores[i][j] += t(row)

        # Add to ensemble
        self.current.trees.append(t)

        # Return gradients
        return t


class FullLossTrainer(object):
    def __init__(self, queries, learner, loss, seed_rate=.1, decreasing_rate=.5,
            increasing_rate=.05, min_rate=1e-15, max_rate=1e+15):
        # Basic set up
        self.queries = queries 
        self.learner = learner
        self.loss = loss

        # Control learning rates
        self.current_rate = seed_rate
        self.decreasing_rate = decreasing_rate
        self.increasing_rate = increasing_rate
        self.min_rate = min_rate
        self.max_rate = max_rate

        # Metric to watch for during training
        self.cost = compute_dataset_loss(queries, self.learner.current, self.loss) 

    def step(self):
        self.learner.step(self.queries, self.loss, None, learning_rate=self.current_rate)
        
        if isinstance(self.learner, BoostedTreeLearner):
            return

        drop_learning_rate, cost, message = self.check_step()

        # adjust the learning_rate based on 'bold dirver algorithm'
        if drop_learning_rate: 
            self.current_rate *= self.decreasing_rate
            self.learner = self.learner.previous_state  # go back to old position
        else:
            self.current_rate *= (1. + self.increasing_rate)
            self.cost = cost

        logger.debug(message % self.current_rate)

        if self.current_rate <= self.min_rate:
            self.current_rate = self.min_rate
            logger.debug('Reached min learning rate, learning rate fixed at %f' % self.min_rate)

        if self.current_rate >= self.max_rate:
            self.current_rate = self.max_rate 
            logger.debug('Reached max learning rate, learning rate fixed at %f' % self.max_rate)

    def check_step(self): 
        cost = compute_dataset_loss(self.queries, self.learner.current, self.loss)
        if cost > self.cost:
            return True, cost, 'Loss increased, learning rate dropped to %f'
        return False, cost, 'Cost improved or unchanged, learning rate increased to %f'


class IRTrainer(object):
    def __init__(self, queries, learner, loss, ir, seed_rate=.1, decreasing_rate=.5,
            increasing_rate=.05, min_rate=1e-15, max_rate=1e+15):
        # Basic set up
        self.queries = queries 
        self.learner = learner
        self.loss = loss
        self.ir = ir 

        # Control learning rates
        self.current_rate = seed_rate
        self.decreasing_rate = decreasing_rate
        self.increasing_rate = increasing_rate
        self.min_rate = min_rate
        self.max_rate = max_rate

        # Metric to watch for during training
        self.objective = compute_dataset_irscore(queries, self.learner.current, self.ir)

    def step(self):
        self.learner.step(self.queries, self.loss, self.ir, learning_rate=self.current_rate)

        if isinstance(self.learner, BoostedTreeLearner):
            return

        drop_learning_rate, objective, message = self.check_step()

        if drop_learning_rate: 
            self.current_rate *= self.decreasing_rate
            self.learner = self.learner.previous_state  # go back to old position
        else:
            self.current_rate *= (1 + self.increasing_rate)
            self.objective = objective

        logger.debug(message % self.current_rate)

        if self.current_rate <= self.min_rate:
            self.current_rate = self.min_rate
            logger.debug('Reached min learning rate, learning rate fixed at %f' % self.min_rate)

        if self.current_rate >= self.max_rate:
            self.current_rate = self.max_rate 
            logger.debug('Reached max learning rate, learning rate fixed at %f' % self.max_rate)

    def check_step(self): 
        objective = compute_dataset_irscore(self.queries, self.learner.current, self.ir)
        if objective < self.objective:
            logger.debug('previous objective: %f, current objective: %f' % (self.objective, objective))
            return True, objective, 'IR measure decreased, learning rate dropped to %f'
        return False, objective, 'Cost improved or unchanged, learning rate increased to %f'
