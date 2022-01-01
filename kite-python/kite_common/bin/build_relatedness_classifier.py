#!/usr/bin/env python
import argparse
import gensim

import kite.classification.svm as svm
import kite.classification.utils as utils

import kite.relatednessclassifier.svm as relatedness

CV_FOLD = 10


def main():
    # allows one to evaluate the model against the given text input.
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--train_input",
        help="file that contains the positive and negative examples",
        required=True)
    parser.add_argument("--word2vec_model", required=True,
                        help="pre-trained word2vec model")
    parser.add_argument(
        "--output_dir",
        help="dir where the trained model and test output go",
        required=True)
    parser.add_argument(
        "--kernel",
        default=svm.Kernel.linear.name,
        help="kernel type")
    parser.add_argument(
        "--svm_type",
        default='svm',
        help="svm type: nu_svm/svm")
    args = parser.parse_args()

    # instantiate classifier
    word2vec_model = gensim.models.Word2Vec.load(args.word2vec_model)
    featurizer = relatedness.TitleRelatednessFeaturizer(word2vec_model)
    model = svm.SVMClassifier(args.kernel, featurizer, args.svm_type)

    # load training data
    print('Loading data for training SVM...')
    all_data, all_labels = utils.load_training_data_with_labels(
        args.train_input)

    # evaluate model by cross validation
    acc = model.cross_validate(all_data, all_labels, CV_FOLD)
    print('Accuracy of %d fold shuffled CV: %.2f' % (CV_FOLD, acc))

    # train classifier
    print('Training model...')
    model.train(all_data, all_labels)

    # save model
    print('Pickling model...')
    utils.pickle_model(model, args.kernel, args.output_dir)

if __name__ == "__main__":
    main()
