import json
import random

import numpy as np
import scipy

import matplotlib
import matplotlib.pyplot as plt
import seaborn

import kite.ranking.ranklearning as ranklearning


def sample_easy_features(ranks):
    ranks = np.asarray(ranks)
    return np.vstack((ranks*.01, np.random.rand(5, len(ranks)))).T


def sample_cubic_features(ranks):
    ranks = -10. + 20. * (np.asarray(ranks) / np.max(ranks))
    x = ranks * (ranks+1.) * (ranks-1.)
    x = ranklearning.normalize_minmax(x)
    return np.vstack((x * .1, np.random.rand(5, len(ranks)))).T


def sample_polar_features(ranks):
    r = ranklearning.normalize_minmax(ranks)
    x = np.cos(r*np.pi*2) * r
    y = np.sin(r*np.pi*2) * r
    return np.vstack((x, y)).T


def sample_polar_hinge_features(ranks):
    r = ranklearning.normalize_minmax(ranks)
    x = np.cos(r*np.pi*3) * np.minimum(r, .5) + 1.
    y = np.sin(r*np.pi*3) * np.minimum(r, .5) + 1.
    return np.vstack((x, y)).T


def get_plot_limits(data, margin=.25):
    xmin, ymin = np.min(data, axis=0)
    xmax, ymax = np.max(data, axis=0)
    xc = (xmin + xmax) / 2.
    yc = (ymin + ymax) / 2.
    xr = (xmax - xmin) / 2.
    yr = (ymax - ymin) / 2.
    r = max(xr, yr) * (margin + 1.)
    return xc-r, xc+r, yc-r, yc+r


def set_plot_limits(data, margin=.25):
    xmin, xmax, ymin, ymax = get_plot_limits(data, margin)
    plt.xlim(xmin, xmax)
    plt.ylim(ymin, ymax)


def evaluate_relations(scores, relations):
    return [scores[i] > scores[j] for i, j in relations]


def chunks(l, n):
    """Yields successive n-sized chunks from l."""
    for i in range(0, len(l), n):
        yield l[i:i+n]


class Query(object):
    """
    Query represents the training data and relevance scores for a query. 
    """
    def __init__(self):
        self.features = [] # feature vectors
        self.relevance = [] # relevance label
        self.snapshot_ids = []
        self.relations = [] # relation (i, j) indicates that item i should rank above item j

    def add(self, snapshot_id, features, label):
        self.snapshot_ids.append(snapshot_id)
        self.features.append(features)
        self.relevance.append(label)

    def build_relations(self):
        self.features = np.array(self.features)
        self.relevance = np.array(self.relevance)

        # Build relations
        for i, a in enumerate(self.relevance): 
            for j, b in enumerate(self.relevance): 
                if a > b: 
                    self.relations.append((i, j))


def main():
    np.random.seed(0)
    matplotlib.rc('legend', fontsize=10)
    matplotlib.rc('font', size=10)

    # Sample a ground truth order
    num_items = 500
    num_train = 200
    num_outliers = 10
    num_steps = 50

    # Construct features for each item
    true_features = sample_polar_hinge_features(range(num_items))

    plt.clf()
    plt.plot(true_features[:, 0], true_features[:, 1], '.r')
    plt.xlim(0, 2)
    plt.ylim(0, 2)
    plt.savefig('out/features.pdf')

    # Sample some features and labels
    train_indices = np.random.permutation(num_items)[:num_train]
    train_mask = np.array([i in train_indices for i in range(num_items)])
    train_features = true_features[train_indices, :]
    train_relations = ranklearning.relations_from_scores(train_indices)
    train_ordering = sorted(range(num_train), key=lambda i: train_indices[i])

    # Extract features
    queries = [] 
    
    # Create query objects
    n = int(len(train_indices) / 10)
    for i, batch in enumerate(chunks(train_indices, n)):
        query = Query()
        labels = np.floor(4 * (1 - ranklearning.normalize_minmax(batch)))
        for j, feats in enumerate(true_features[batch, :]):
            query.add(i, feats, labels[j])
        queries.append(query)

    # Build the relations 
    for _, query in enumerate(queries):
        query.build_relations()

    #
    # Build the IR measurer
    #
    ir = ranklearning.NDCG()

    #
    # Create the loss function
    #
    loss = ranklearning.CrossEntropyLoss()


    #
    # Create the learner and optimize
    #

    learner = ranklearning.BoostedTreeLearner(queries, use_sklearn_trees=False)
    trainer = ranklearning.IRTrainer(queries, learner, loss, ir)

    loss_scores = [ranklearning.compute_dataset_loss(queries, learner.current, loss)]
    ndcg_scores = [ranklearning.compute_dataset_irscore(queries, learner.current, ir)]

    for i in range(num_steps):
        print('Gradient step %d' % i)
        trainer.step()
        loss_scores.append(ranklearning.compute_dataset_loss(queries, learner.current, loss))
        ndcg_scores.append(ranklearning.compute_dataset_irscore(queries, learner.current, ir))

    loss_scores = np.array(loss_scores)
    ndcg_scores = np.array(ndcg_scores)

    #
    # Plotting
    #

    print(ndcg_scores)

    # Plot final scores
    fig, ax1 = plt.subplots()
    iterations = np.array(range(num_steps + 1))
    ax1.plot(iterations, loss_scores, 'b-')
    ax1.set_xlabel('iteration #')
    ax1.set_ylabel('Loss fucntion', color='b')
    for tl in ax1.get_yticklabels():
        tl.set_color('b')

    ax2 = ax1.twinx()
    ax2.plot(iterations, ndcg_scores, 'r.')
    ax2.set_ylabel('NDCG', color='r')
    for tl in ax2.get_yticklabels():
        tl.set_color('r')

    plt.savefig('out/scores.pdf')

    # Write output model
    with open('out/model.json', 'w') as f:
        json.dump(learner.current.to_json(), f)

if __name__ == '__main__':
    main()
