import os
import argparse
import json

import kite.classification.svm as svm
import kite.classification.utils as utils
import kite.errorclassifier.svm as error

POSITIVE_LABEL = 1  # label for positive training data
NEGATIVE_LABEL = -1  # label for negative training data


def main():
    # allows one to evaluate the model against the given text input.
    parser = argparse.ArgumentParser()
    parser.add_argument("train_input")
    # optional since sometimes you may just want to train a new model and not
    # eval it
    parser.add_argument("--eval_input")
    parser.add_argument("output_dir")
    parser.add_argument("--text")
    parser.add_argument("--kernel", default=svm.Kernel.linear.name)
    parser.add_argument("--featvec", dest="featvec", action="store_true")
    parser.set_defaults(featvec=False)
    args = parser.parse_args()

    pos_files, neg_files = utils.get_training_files(args.train_input)
    eval_files = utils.get_eval_files(args.eval_input)

    # instantiate classifier
    most_freq_words = utils.compute_most_freq_words(
        pos_files +
        neg_files,
        error.N_TOP_FREQ_WORDS)
    featurizer = error.ErrorFeaturizer(most_freq_words=most_freq_words)
    model = svm.SVMClassifier(args.kernel, featurizer)

    # train classifier
    all_data, all_labels = utils.load_training_data(
        pos_files, neg_files, POSITIVE_LABEL, NEGATIVE_LABEL)
    model.train(all_data, all_labels)

    # if a particular input text is specified, evaluate the model against it
    # and print results
    if args.text != "":
        if args.featvec:
            feat_vec = featurizer.features(args.text)
            print(utils.feat_vec_to_string(feat_vec))
        else:
            print(model.classify(args.text))

    if not os.path.isdir(args.output_dir):
        print("Output dir for model not found.")
        return

    # evaluate model
    for test_file in eval_files:
        html_output = utils.evaluate_and_output_pretty_html(
            model,
            test_file)
        with open(os.path.join(args.output_dir, os.path.basename(test_file)), "w+") as f:
            f.write(html_output)

    # Save model
    utils.export_model(model, args.kernel, args.outputdir)

    # write additional data useful for feature computation out to json file
    feat_data = {
        'most_freq_words': most_freq_words,
        'size_char_hash_vec': error.SIZE_CHAR_HASH_VEC,
        'size_word_hash_vec': error.SIZE_WORD_HASH_VEC}

    # Save feat data
    utils.save_feature_data(feat_data, args.output_dir)

if __name__ == "__main__":
    main()
