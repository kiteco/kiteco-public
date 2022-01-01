import argparse
import random
import tensorflow as tf
import numpy as np
import subprocess
import os
import json

from kite.infer_keyword import Config, Dataset, KeywordModelEncoder, Model, TrainInputs
from kite.utils.save import save_model, save_frozen_model

parser = argparse.ArgumentParser(description='train situation/keyword classifier')
parser.add_argument('features_json', metavar='FEATURES_JSON', type=str, help='Path to features JSON file')
parser.add_argument('export_dir', metavar='EXPORT_DIR', type=str, help='Path to model export dir')
parser.add_argument('frozen_model', metavar='FROZEN_MODEL', type=str, help='Path to exported frozen model')

stats_filepath = "/data/kite/keywords_model/stats.json"

TEST_SPLIT = 0.2


def get_base_stats(is_keyword_training=True, feature_encoder=None):
    source_path = os.path.dirname(os.path.abspath(__file__))
    git_hash = str(subprocess.check_output(['git', 'rev-parse', '--short','HEAD'], cwd=source_path))[2:-4]
    description = "5 epochs, categorical prefix style and previous keywords"
    training = "isKeyword" if is_keyword_training else "whichKeyword"
    features = ""
    if feature_encoder:
        features = feature_encoder.get_features_str()
    pid = os.getpid()
    return dict(description=description, git_hash=git_hash, training=training, pid=pid, features=features)


def get_stats():
    try:
        with open(stats_filepath, "r") as i:
            result = json.load(i)
    except Exception:
        result = []
    return result


def save_stats(s: list):
    with open(stats_filepath, "w") as o:
        json.dump(s, o)
        print("Stats saved in {}".format(stats_filepath))


def main():
    random.seed(1)  # for reproducibility

    args = parser.parse_args()

    config = Config()

    feature_encoder = KeywordModelEncoder(config)

    stats_file = get_stats()

    print("feature encoder: in_size={0}, out_size={1}".format(feature_encoder.in_size(), feature_encoder.out_size()))
    print("features:")
    for feature in feature_encoder.features:
        print("{0}: in_size={1}".format(feature.__class__.__name__, feature.in_size))

    model = Model(feature_encoder=feature_encoder)

    all_data = Dataset(config=config).load(args.features_json, max_count=-1)
    all_train, all_test = all_data.train_test_split(test_size=TEST_SPLIT)

    kw_data = all_data.filter(lambda rec: rec.is_keyword == 1)
    kw_train, kw_test = kw_data.train_test_split(test_size=TEST_SPLIT)

    with tf.Session() as sess:
        sess.run(tf.global_variables_initializer())

        print("training is-keyword (vs name) classifier")
        inputs = TrainInputs(config, sess, all_train, all_test, model.is_keyword, lambda batch: batch.is_keyword)
        model.train_classifier(inputs, stats=stats_file, baseStat=get_base_stats(True, feature_encoder))

        print("training keyword classifier")
        # keyword cat are 1-indexed (to avoid using 0 as it's the null value for int)
        # So we remove 1 to the category to be 0-indexed
        inputs = TrainInputs(config, sess, kw_train, kw_test, model.which_keyword,
                             lambda batch: np.array(batch.keyword_cat) - 1)
        model.train_classifier(inputs, stats=stats_file, baseStat=get_base_stats(False, feature_encoder))

        print("saving model to {0}".format(args.export_dir))
        outputs = {
            "features/x": model.x,
            "classifiers/is_keyword/logits": model.is_keyword.logits,
            "classifiers/is_keyword/weights": model.is_keyword.weights,
            "classifiers/is_keyword/biases": model.is_keyword.biases,
            "classifiers/which_keyword/logits": model.which_keyword.logits,
            "classifiers/which_keyword/weights": model.which_keyword.weights,
            "classifiers/which_keyword/biases": model.which_keyword.biases,
        }
        save_stats(stats_file)

        save_model(
            sess,
            args.export_dir,
            inputs={"features/features": model.features},
            outputs=outputs)

        print("saving frozen model to {0}".format(args.frozen_model))
        save_frozen_model(sess, args.frozen_model, list(outputs.keys()))



if __name__ == "__main__":
    main()
