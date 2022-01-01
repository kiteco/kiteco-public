#!/usr/bin/env python
import argparse
import os

import kite.classification.svm as svm
import kite.classification.utils as utils
import kite.classification.logit as logit
import kite.classification.featurizers as featurizers

from logging import getLogger, DEBUG, INFO, StreamHandler
from sys import stdout

handler = StreamHandler(stdout)
logger = getLogger()
logger.addHandler(handler)

CV_FOLD = 10

def main():
    # allows one to evaluate the model against the given text input.
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--train_input',
        help='training file that contains the positive and negative examples',
        required=True)
    parser.add_argument(
        '--test_input',
        help='test file that contains the positive and negative examples')
    parser.add_argument(
        '--output_dir',
        help='dir where the trained model and test output go',
        required=True)
    parser.add_argument(
        '--kernel',
        default=svm.Kernel.linear.name,
        choices=[kernel.name for kernel in svm.Kernel])
    parser.add_argument(
        '--svm_type',
        default='svm',
        choices=['nu_svm', 'svm'])
    parser.add_argument(
        '--model',
        default='logit',
        choices=['svm', 'logit'])
    parser.add_argument(
        '--verbosity', default=0, type=int)
    args = parser.parse_args()

    # set logger verbosity level
    if args.verbosity > 0:
        logger.setLevel(DEBUG)
    else:
        logger.setLevel(INFO)

    # instantiate classifier
    featurizer = featurizers.IdentFeaturizer()
    if args.model == 'svm':
        model = svm.SVMClassifier(args.kernel, featurizer, args.svm_type)
    elif args.model == 'logit':
        model = logit.LogisticRegressionClassifier(featurizer)
    else:
        logger.error('%s is not supported' % args.model)
        exit(1)

    # load training data
    logger.info('Loading data for training...')
    all_data, all_labels = utils.load_training_data_with_labels(
        args.train_input)

    # evaluate model by cross validation
    acc = model.cross_validate(all_data, all_labels, CV_FOLD)
    logger.info('Accuracy of %d fold shuffled CV: %.2f' % (CV_FOLD, acc))

    # train classifier
    logger.info('Training model...')
    model.train(all_data, all_labels)

    # save model
    logger.info('Saving model...')
    if not os.path.isdir(args.output_dir):
        print("Output dir for model not found. Creating one.")
        os.makedirs(args.output_dir)
    model.export(os.path.join(args.output_dir, 'logistic_regression.json'))

if __name__ == '__main__':
    main()
