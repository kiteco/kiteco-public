import argparse
import random
import tensorflow as tf
import numpy as np
import subprocess
import os
import json

from kite.infer_keyword import Config, KeywordModelEncoder, Model, TrainInputs
from kite.utils.save import save_model, save_frozen_model, load_frozen_model

from kite.infer_keyword.data import Batch, Dataset



model_path = "/data/kite/keywords_model/out/tmp/keyword_model.frozen.pb"
dataset_path = "/data/kite/keywords_model/out/comparison_data.json"

item_to_skip = {14, 20, 30, 31, 32, 33, 41, 42, 46, 47, 48, 54, 58, 71, 80, 88, 89, 94, 101, 105, 113, 114, 115, 116, 125, 133, 136, 137, 138, 139, 143, 148, 149, 153, 171, 173, 176, 179, 180, 185, 186, 187, 196, 200, 202, 203, 204, 206, 207, 212, 213, 219, 224, 243, 244, 250, 252, 253, 259, 264, 270, 271, 278, 281, 283, 288, 306, 317, 323, 324, 328, 329, 332, 334, 335, 336, 339, 344, 356, 366, 367, 374, 379, 381, 382, 389, 391, 393, 394, 399, 402, 405, 410}

def main():
    random.seed(1)  # for reproducibility

    config = Config()

    feature_encoder = KeywordModelEncoder(config)
    print("Features : {}".format(feature_encoder.get_features_str()))

    print("feature encoder: in_size={0}, out_size={1}".format(feature_encoder.in_size(), feature_encoder.out_size()))
    print("features:")
    for feature in feature_encoder.features:
        print("{0}: in_size={1}".format(feature.__class__.__name__, feature.in_size))

    # model = Model(feature_encoder=feature_encoder)

    all_data = Dataset(config=config).load(dataset_path, max_count=-1)

    filtered_records = []
    for i, rec in enumerate(all_data.records):
        if i not in item_to_skip:
            filtered_records.append(rec)
    all_data.records = filtered_records

    kw_data = all_data.filter(lambda rec: rec.is_keyword == 1)

    graph = load_frozen_model(model_path)
    print([n.name for n in graph.as_graph_def().node])
    with tf.Session(graph=graph) as sess:
        sess.run(tf.global_variables_initializer())

        inputs = TrainInputs(config, sess, None, all_data, None, lambda batch: batch.is_keyword)

        test_batch = Batch(inputs.test.records, feature_encoder)
        test_feed_dict = {
            "features/features:0": test_batch.features,
        }
        logits_is_keyword, features_is_keyword = sess.run(["classifiers/is_keyword/logits:0", "features/x:0"], feed_dict=test_feed_dict) #
        all_vectors = test_batch.features
        # keyword cat are 1-indexed (to avoid using 0 as it's the null value for int)
        # So we remove 1 to the category to be 0-indexed
        inputs = TrainInputs(config, sess, None, kw_data, None,
                             lambda batch: np.array(batch.keyword_cat) - 1)
        test_batch = Batch(inputs.test.records, feature_encoder)
        test_feed_dict = {
            "features/features:0": test_batch.features,
        }
        logits_which_keyword, features_which_keyword = inputs.sess.run(["classifiers/which_keyword/logits:0", "features/x:0"], feed_dict=test_feed_dict)

    data = dict(logits_is_keyword=logits_is_keyword.tolist(),
                logits_which_keyword=logits_which_keyword.tolist(),
                features_is_keyword=features_is_keyword.tolist(),
                features_which_keyword=features_which_keyword.tolist(),
                features=[r.features.__dict__ for r in filtered_records],
                is_keyword=[r.is_keyword for r in filtered_records],
                which_keyword=[r.keyword_cat for r in filtered_records],
                batch_features=all_vectors)
    with open("/data/kite/keywords_model/result_python.json", "w") as outfile:
        json.dump(data, outfile)

if __name__ == "__main__":
    main()
