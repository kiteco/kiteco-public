import tensorflow as tf

import argparse
import json
import os

from model.config import Config, SearchConfig
from model.model import LexicalModel
from model.prefix_suffix_lm import Model as PrefixSuffixLM
from kite.utils.save import save_frozen_model


def get_model(config, search_config, training, cpu):
    models = {
        'lexical': LexicalModel,
        'prefix_suffix': PrefixSuffixLM,
    }
    Model = models[config.model_type]
    return Model(config=config, search_config=search_config, training=training, cpu=cpu)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--in_saved_model', type=str, default='')
    parser.add_argument('--config', type=str, default='')
    parser.add_argument('--search_config', type=str, default='')
    parser.add_argument('--out_saved_model', type=str, default='')
    args = parser.parse_args()

    config = Config.from_json(json.load(open(args.config, 'r')))
    search_config = SearchConfig.from_json(json.load(open(args.search_config, 'r')))

    print("using config:", config)
    print("using searchconfig:", search_config)

    # initialize new model
    model = get_model(config, search_config=search_config, training=False, cpu=False)

    # load values of variables from saved model
    saver = tf.compat.v1.train.Saver(var_list=tf.compat.v1.global_variables())
    builder = tf.compat.v1.saved_model.builder.SavedModelBuilder(
        args.out_saved_model)

    with tf.compat.v1.Session() as sess:
        print('importing model from', args.in_saved_model)
        saver.restore(sess, os.path.join(
            args.in_saved_model, "variables/variables"))

        inputs = model.tfserving_inputs_dict()
        inputs = {k: tf.compat.v1.saved_model.utils.build_tensor_info(
            v) for k, v in inputs.items()}

        outputs = model.tfserving_outputs_dict()
        outputs = {k: tf.compat.v1.saved_model.utils.build_tensor_info(
            v) for k, v in outputs.items()}

        query_signature = tf.compat.v1.saved_model.build_signature_def(
            inputs=inputs,
            outputs=outputs,
            method_name=tf.compat.v1.saved_model.signature_constants.PREDICT_METHOD_NAME,
        )

        print('exporting trained model to', args.out_saved_model)
        builder.add_meta_graph_and_variables(
            sess,
            [tf.compat.v1.saved_model.tag_constants.SERVING],
            signature_def_map={
                tf.compat.v1.saved_model.signature_constants.DEFAULT_SERVING_SIGNATURE_DEF_KEY: query_signature,
            },
        )

        builder.save(as_text=True)


if __name__ == "__main__":
    main()
