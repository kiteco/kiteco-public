import json
import argparse

from operator import itemgetter
from collections import namedtuple
import subprocess
import os

import numpy as np

import matplotlib.pyplot as plt

import sklearn
from sklearn.tree import export_graphviz

import kite.ranking.ranklearning as ranklearning

from logging import getLogger, DEBUG, INFO, StreamHandler
from sys import stdout

handler = StreamHandler(stdout)
logger = getLogger()
logger.addHandler(handler)

divider = '=' * 80

def load_dataset(path, min_training_examples=10):
    """Load a dataset from the given path."""
    with open(path) as f:
        train_data = json.load(f)

    # Load training data
    queries = dict()
    example_ids = dict()
    features = dict()
    relevance = dict()

    for item in train_data['Data']:
        query_id = item['query_id']
        if query_id not in queries:
            queries[query_id] = {
                'query_text': item['query_text'],
                'query_code': item['query_code'],
                'query_id': query_id}
            example_ids[query_id] = []
            features[query_id] = []
            relevance[query_id] = []

        example_ids[query_id].append(item['snapshot_id'])
        features[query_id].append(item['features'])
        relevance[query_id].append(item['label'])

    data = []
    for hash_id in queries:
        query = ranklearning.Query(queries[hash_id], features[hash_id],
                                   relevance[hash_id], example_ids[hash_id])
        if len(query.relations) > 0:
            data.append(query)

    return ranklearning.Dataset(train_data['FeatureLabels'], data, train_data['FeaturerOptions'])


def main():
    # Command line args
    parser = argparse.ArgumentParser()
    parser.add_argument('train')
    parser.add_argument('--test', type=str)
    parser.add_argument('--validate', type=str)
    parser.add_argument('--normalizer',
                        choices=['min_max', 'mean_std'],
                        default='mean_std')
    parser.add_argument('--kfold', type=int, default=-1)

    parser.add_argument('--model',
                        choices=['linear', 'rbf', 'mart'], default='mart')
    parser.add_argument('--loss_function',
                        choices=['hinge_loss', 'cross_entropy'],
                        default='cross_entropy')
    parser.add_argument('--init_weight', nargs='+', type=float)
    parser.add_argument('--random_init', action='store_true')
    parser.add_argument('--num_seeds', type=int, default=10)
    parser.add_argument('--t', type=int, default=10, help='cut-off T for NDCG')
    parser.add_argument('--ir_measure', choices=['ndcg'])

    parser.add_argument('--root', type=str, default="training_output")
    parser.add_argument('--base', type=str, required=True)

    parser.add_argument('--use_sklearn_trees', action='store_true')
    parser.add_argument('--num_steps', type=int, default=100)
    parser.add_argument('--learning_rate', type=float, default=1.)
    parser.add_argument('--cv_learning_rates',
                        nargs='+', type=float, default=[1.])
    parser.add_argument('--max_depth', type=int, default=5)
    parser.add_argument('--min_samples_leaf', type=int, default=3)

    parser.add_argument('--verbosity', default=0, type=int)
    args = parser.parse_args()

    if args.verbosity > 0:
        logger.setLevel(DEBUG)
    else:
        logger.setLevel(INFO)

    ranklearning.logger = logger

    # set up model paramters
    num_steps = args.num_steps
    learning_rate = args.learning_rate
    max_depth = args.max_depth
    min_samples_leaf = args.min_samples_leaf

    # set up loss function
    logger.info('[Training] set up loss function')
    loss = None
    if args.loss_function == 'hinge_loss':
        loss = ranklearning.HingeLoss()
    elif args.loss_function == 'cross_entropy':
        loss = ranklearning.CrossEntropyLoss()
    else:
        exit("Loss function '%s' is not supported" % args.loss_function)

    # set up ir
    logger.info('[Training] set up ir measurer')
    ir = None
    if args.ir_measure == 'ndcg':
        ir = ranklearning.NDCG(cutoff=args.t)
    elif args.ir_measure is not None:
        exit("IR measure '%s' is not supported" % args.ir_measure)

    # cross validate if requested
    if args.validate is not None or args.kfold > 0:
        if args.validate is not None and args.kfold > 0:
            exit('cannot specify both --validate and --kfold')
        if args.model == 'mart':
            best_config = cross_validation_mart(args, loss, ir)
            learning_rate, max_depth, num_steps = best_config
        else:
            logger.warn('cross validation is only for MART. Proceed with default parameters')
    if args.random_init:
        if args.model == 'linear':
            best_random_weight = choose_init_random_weight(args, loss, ir)
        else:
            logger.warn('random init is only valid for linear models now. Proceed with an initial all-zero vector.')


    info = divider + '\n' 
    info += '[Training] learning_rate: %f\n' % learning_rate
    info += '[Training] max_depth: %d\n' % max_depth
    info += '[Training] num_steps: %d\n' % num_steps
    info += '[Training] learner: %s\n' % args.model
    if args.random_init and args.model == 'linear':
        info += '[Training] random_init: true\n'
    else:
        info += '[Training] random_init: false\n'
    if args.ir_measure is None:
        info += '[Training] use ir: none\n'
    else:
        info += '[Training] use ir: %s\n' % args.ir_measure
    if args.init_weight is not None:
        info += '[Training] initial weights: %s\n' % ' '.join(str(w) for w in args.init_weight)
    info += '[Training] loss function: %s\n' % args.loss_function
    info += divider 
    logger.info(info)

    # Load training data
    logger.info('[Training] loading training data...')
    train_data = load_dataset(args.train)

    logger.info('[Training] num of relations: %d' % train_data.num_relations())

    # Build the normalizer
    logger.info('[Training] setting up the normalizer...')
    if args.normalizer == 'mean_std':
        normalizer = ranklearning.meanstd_normalizer(train_data.features())
    else:
        normalizer = ranklearning.minmax_normalizer(train_data.features())

    # Normalize the training data
    logger.info('[Training] normalizing the training data...')
    for query in train_data.queries:
        query.features = normalizer(query.features)

    # Set up test data
    if args.test is not None:
        logger.info('[Training] loading test data...')
        test_data = load_dataset(args.test)
        for query in test_data.queries:
            query.features = normalizer(query.features)

    # Set up learner
    learner = None
    if args.model == 'linear':
        if args.init_weight is not None:
            if len(args.init_weight) != train_data.nd:
                error = 'length of init weight %d not match with feature length %d' % (len(args.init_weight), train_data.nd)
                exit(error)
            weight = args.init_weight
        elif args.random_init:
            weight = best_random_weight 
        else:
            weight = np.zeros(train_data.nd, dtype=float)
        learner = ranklearning.LinearRankLearner(weight, train_data.nd)
    elif args.model == 'rbf':
        learner = ranklearning.KernelRankLearner(
            train_data.features(), kernel=ranklearning.Rbf(-1.))
    elif args.model == 'mart':
        logger.info('[Training] use sklearn trees: %s' % str(args.use_sklearn_trees))
        learner = ranklearning.BoostedTreeLearner(
            train_data.queries,
            use_sklearn_trees=args.use_sklearn_trees,
            max_depth=max_depth,
            min_samples_leaf=min_samples_leaf)
    else:
        exit('Learner %s is not supported' % args.model)

    # set up trainer
    if loss is not None and learner is not None:
        if ir is None:
            trainer = ranklearning.FullLossTrainer(
                train_data.queries, learner, loss, seed_rate=learning_rate)
        else:
            trainer = ranklearning.IRTrainer(
                train_data.queries, learner, loss, ir, seed_rate=learning_rate)

    #
    # training
    #

    train_loss = []
    test_loss = []

    train_ir = []
    test_ir = []

    for i in range(num_steps):
        trainer.step()
        if args.ir_measure is None:
            ir_scorer = ranklearning.NDCG()
        else:
            ir_scorer = trainer.ir

        ir_score = ranklearning.compute_dataset_irscore(
            train_data.queries, trainer.learner.current, ir_scorer)
        train_ir.append(ir_score)

        if args.test is not None:
            ir_score = ranklearning.compute_dataset_irscore(
                test_data.queries, trainer.learner.current, ir_scorer)
            test_ir.append(ir_score)

        loss_score = ranklearning.compute_dataset_loss(
            train_data.queries, trainer.learner.current, trainer.loss)
        train_loss.append(loss_score)


        if args.test is not None:
            loss_score = ranklearning.compute_dataset_loss(
                test_data.queries, trainer.learner.current, trainer.loss)
            test_loss.append(loss_score)

        logger.debug('-------- Report --------')
        logger.debug('[Training] ir score at step %d: %f' % (i, train_ir[-1]))
        logger.debug('[Training] loss score at step %d: %f' % (i, train_loss[-1]))
        logger.debug('[Test] ir score at step %d: %f' % (i, test_ir[-1]))
        logger.debug('[Test] loss score at step %d: %f' % (i, test_loss[-1]))


    logger.info('-------- Training Overview --------')
    logger.info('[Training] ir score at step %d: %f' % (i, train_ir[-1]))
    logger.info('[Training] loss score at step %d: %f' % (i, train_loss[-1]))
    logger.info('[Test] ir score at step %d: %f' % (i, test_ir[-1]))
    logger.info('[Test] loss score at step %d: %f' % (i, test_loss[-1]))

    train_loss = np.array(train_loss)
    test_loss = np.array(test_loss)

    train_ir = np.array(train_ir)
    test_ir = np.array(test_ir)

    #
    # Generate the final output file for test data
    #

    directory = os.path.join(args.root, args.base)
    if not os.path.exists(directory):
        os.makedirs(directory)

    if args.test is not None:
        fname = os.path.join(args.root, args.base, 'test-results.json')
        f = open(fname, 'w')

        for query in test_data.queries:
            payload = dict()
            payload['query_id'] = query.info['query_id']
            payload['query_text'] = query.info['query_text']
            payload['query_code'] = query.info['query_code']

            payload['labels'] = list()
            payload['example_ids'] = list()
            payload['expected_rank'] = list()
            payload['scores'] = list()
            payload['features'] = list()
            payload['featurelabels'] = test_data.featurelabels

            scores = trainer.learner.current(query.features)
            ranking = ranklearning.ranks_from_scores(scores)
            expected_rankings = ranklearning.ranks_from_scores(query.relevance)

            if args.ir_measure is not None:
                payload['ndcg'] = trainer.ir(query.relevance, scores)
            else:
                ir_scorer = ranklearning.NDCG()
                payload['ndcg'] = ir_scorer(query.relevance, scores)

            for r in ranking:
                payload['labels'].append(float(query.relevance[r]))
                payload['example_ids'].append(query.example_ids[r])
                payload['expected_rank'].append(expected_rankings.index(r))
                payload['scores'].append(scores[r])
                payload['features'].append(query.features[r].tolist())
            json.dump(payload, f)

        f.close()

    #
    # Write model as json if requested
    #

    fname = os.path.join(args.root, args.base, 'model.json')
    f = open(fname, 'w')

    ranker = {
        "Normalizer": {
            "Offset": normalizer.offset.tolist(),
            "Scale": normalizer.scale.tolist(),
        },
        "Scorer": learner.current.to_json(),
        "FeatureLabels": train_data.featurelabels,
        "FeaturerOptions": train_data.options,
    }
    if isinstance(learner.current, ranklearning.LinearScorer):
        ranker["ScorerType"] = "Linear"
    elif isinstance(learner.current, ranklearning.KernelScorer):
        ranker["ScorerType"] = "RbfKernel"
    elif isinstance(learner.current, ranklearning.TreeEnsembleScorer):
        ranker["ScorerType"] = "TreeEnsemble"
    else:
       logger.warning("Unknown model type: %s" % learner.current)

    with open(fname, "w") as f:
        json.dump(ranker, f)

    f.close()

    #
    # Plot basic training info
    #

    logger.info("[Training] plotting loss function...")

    plt.clf()
    plt.subplot2grid((1, 3), (0, 0), colspan=2)  # make space for legend
    plt.plot(train_loss, label='training')
    if args.test is not None:
        plt.plot(test_loss, label='test')

    plt.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)
    plt.xlabel('iteration #')
    plt.ylabel('Loss fucntion')

    filename = os.path.join(args.root, args.base, "loss.pdf")
    plt.savefig(filename)

    logger.info("[Training] plotting IR measure")
    plt.clf()
    plt.subplot2grid((1, 3), (0, 0), colspan=2)  # make space for legend
    plt.plot(train_ir, label='training')
    if args.test is not None:
        plt.plot(test_ir, label='test')

    plt.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)
    plt.xlabel('iteration #')
    plt.ylabel('IR measure')

    filename = os.path.join(args.root, args.base, "ndcg.pdf")
    plt.savefig(filename)

    #
    # visualize the model
    #

    if args.model == 'mart' and args.use_sklearn_trees:
        logger.info("[Training] visualizing model...")
        for i, t in enumerate(learner.current.trees):
            filename = os.path.join(args.root, args.base, str(i) + '.dot')
            pngfilename = os.path.join(args.root, args.base, str(i) + '.png')

            with open(filename, 'w') as f:
                export_graphviz(t.tree, out_file=f,
                                feature_names=train_data.featurelabels)
            command = ['dot', '-Tpng', filename, "-o", pngfilename]
            try:
                subprocess.check_call(command)
            except:
                exit("Could not run graphviz to produce visualization")


def split_dataset(train_data, validate_data, kfold):
    if validate_data is not None:
        yield train_data, validate_data
    else:
        folds = sklearn.cross_validation.KFold(len(train_data.queries), kfold)
        for train_index, validate_index in folds:
            yield (ranklearning.Dataset(
                        train_data.featurelabels,
                        itemgetter(*train_index)(train_data.queries),
                        train_data.options),
                   ranklearning.Dataset(
                        train_data.featurelabels,
                        itemgetter(*validate_index)(train_data.queries),
                        train_data.options))


def choose_init_random_weight(args, loss, ir):
    train_data = load_dataset(args.train)
    Config = namedtuple('Config', ['weight', 'score'])

    # normalize the features
    if args.normalizer == "mean_std":
        normalizer = ranklearning.meanstd_normalizer(train_data.features())
    else:
        normalizer = ranklearning.minmax_normalizer(train_data.features())
    for query in train_data.queries:
        query.features = normalizer(query.features)

    # try as many random initial seeds as args.num_seeds
    ir_scores = []
    for seed in range(args.num_seeds):
        weight = np.random.uniform(size=train_data.nd)
        learner = ranklearning.LinearRankLearner(weight, train_data.nd)
        trainer = ranklearning.IRTrainer(
            train_data.queries,
            learner, loss, ir,
            seed_rate=args.learning_rate)
        for i in range(args.num_steps):
            trainer.step()
            ir_score = ranklearning.compute_dataset_irscore(
                train_data.queries, trainer.learner.current, trainer.ir)
        logger.info(divider)
        logger.debug('%d-th initial random weights: %s' % (seed, ' '.join(str(w) for w in weight)))
        logger.info('%d-th final random weights: %s' % (seed, ' '.join(str(w) for w in trainer.learner.current.w)))
        logger.info('IR score: %f' % ir_score)
        ir_scores.append(Config(trainer.learner.current.w, ir_score))

    ir_scores.sort(key=itemgetter(-1))
    weight = ir_scores[-1].weight
    score = ir_scores[-1].score
    logger.info('Chosen weights: %s, IR score: %f' % (' '.join(str(w) for w in weight), score))
    return weight 


def cross_validation_mart(args, loss, ir):
    train_data = load_dataset(args.train)
    validate_data = None
    if args.validate is not None:
        validate_data = load_dataset(args.validate)

    Config = namedtuple(
        'Config', ['learning_rate', 'max_depth', 'num_step', 'score'])
    logger.info('[Cross-validation] starting...')

    grid_search = []
    for learning_rate in args.cv_learning_rates:
        for max_depth in range(1, args.max_depth + 1):
            ir_scores = []
            folds = split_dataset(train_data, validate_data, args.kfold)
            for training, validation in folds:
                if args.normalizer == "mean_std":
                    normalizer = ranklearning.meanstd_normalizer(
                        training.features())
                else:
                    normalizer = ranklearning.minmax_normalizer(
                        training.features())

                for query in training.queries:
                    query.features = normalizer(query.features)

                for query in validation.queries:
                    query.features = normalizer(query.features)

                learner = ranklearning.BoostedTreeLearner(
                    training.queries,
                    use_sklearn_trees=args.use_sklearn_trees,
                    max_depth=max_depth,
                    min_samples_leaf=args.min_samples_leaf)

                trainer = ranklearning.IRTrainer(
                    training.queries,
                    learner, loss, ir,
                    seed_rate=learning_rate)

                fold_scores = []
                for i in range(args.num_steps):
                    trainer.step()
                    ir_score = ranklearning.compute_dataset_irscore(
                        validation.queries, trainer.learner.current,
                        trainer.ir)

                    n = len(ir_scores) + 1
                    info = '[Cross-validation] '
                    info += '(split %d of %d) ' % (n, args.kfold)
                    info += 'score: %f, ' % ir_score
                    info += 'learning_rate: %f, ' % learning_rate
                    info += 'max_depth: %d, step: %d' % (max_depth, i)
                    logger.debug(info)

                    fold_scores.append(ir_score)

                ir_scores.append(fold_scores)

            ir_scores = np.asarray(ir_scores)
            ir_scores = np.mean(ir_scores, axis=0)
            grid_search.extend([Config(learning_rate, max_depth, i, score)
                                for i, score in enumerate(ir_scores)])

    #
    # Plotting
    #

    directory = os.path.join(args.root, args.base)
    if not os.path.exists(directory):
        os.makedirs(directory)

    # Plot the best performance for each max_depth
    scores = []
    for i in range(1, args.max_depth + 1):
        candidates = filter(lambda x: x.max_depth == i, grid_search)
        scores.append(max(candidates, key=lambda x: x.score).score)

    plt.clf()
    plt.subplot2grid((1, 3), (0, 0), colspan=2)  # make space for legend
    plt.plot(scores, 'ro', label='Best ir score for each max_depth')

    plt.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)
    plt.xlabel('max depth')
    plt.ylabel('ndcg')

    filename = os.path.join(args.root, args.base, 'cv-max_depth.pdf')
    plt.savefig(filename)

    # Plot model performance as a function of number of iterations
    plt.clf()
    num_learning_rates = len(args.cv_learning_rates)
    for i, l in enumerate(args.cv_learning_rates):
        ax = plt.subplot2grid((num_learning_rates, 3), (i, 0), colspan=2)
        ax.set_title('learning_rate: %f' % l)
        for d in range(1, args.max_depth + 1):
            candidates = filter(lambda x: x.learning_rate == l
                                and x.max_depth == d, grid_search)
            scores = [c.score for c in candidates]
            ax.plot(scores, label=('max_depth=%d' % d))

        ax.legend(bbox_to_anchor=(1.05, 1), loc=2, borderaxespad=0.)
        ax.set_ylabel('ndcg')

    plt.xlabel('iteration')

    filename = os.path.join(args.root, args.base, 'cv-iteration.pdf')
    plt.savefig(filename)

    #
    # Find the best config
    #

    grid_search.sort(key=itemgetter(-1))
    best_config = grid_search[::-1][0]
    info = divider + '\n'
    info += '[Cross-validation] '
    info += 'Best setup: learning_rate: %f, ' % best_config.learning_rate
    info += 'max_depth: %d,' % best_config.max_depth
    info += 'step: %d. IR-score: %f' % (best_config.num_step, best_config.score)
    info += '\n[Cross-validation] Using this configuration for training\n'
    info += divider + '\n' 
    logger.info(info)
    return (best_config.learning_rate,
            best_config.max_depth, best_config.num_step)


if __name__ == '__main__':
    main()
