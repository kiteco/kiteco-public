import tensorflow as tf

import argparse
import json
import os

from model.config import Config
from model.model import LexicalModel
from model.prefix_suffix_lm import Model as PrefixSuffixLM
from kite.utils.save import save_frozen_model

def get_model(config, training, cpu):
    models = {
        'lexical': LexicalModel,
        'prefix_suffix': PrefixSuffixLM,
    }
    Model = models[config.model_type]
    return Model(config, training=training, cpu=cpu)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--in_saved_model', type=str, default='out/saved_model')
    parser.add_argument('--out_frozen_model', type=str, default='out/lexical_model.frozen.pb')
    parser.add_argument('--config', type=str, default='out/config.json')
    args = parser.parse_args()

    config = Config.from_json(json.load(open(args.config, 'r')))
    print("using config:", config)

    model = get_model(config, training=False, cpu=True)

    # load values of variables from saved model
    saver = tf.compat.v1.train.Saver(var_list=tf.compat.v1.global_variables())

    with tf.compat.v1.Session() as sess:
        # set the values of the variables to their saved values
        saver.restore(sess, os.path.join(args.in_saved_model, "variables/variables"))

        # save the frozen model
        save_frozen_model(
            sess,
            args.out_frozen_model,
            list(model.outputs_dict().keys()),
        )


if __name__ == "__main__":
    main()
