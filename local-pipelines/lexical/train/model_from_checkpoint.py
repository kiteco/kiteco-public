import logging
import argparse
import json

import tensorflow as tf

from kite.model.model import Model
from kite.utils.save import save_model, save_frozen_model

from model.model import LexicalModel
from model.prefix_suffix_lm import Model as PrefixSuffixLM
from model.config import Config


logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


def get_model(config, training, cpu):
    models = {
        'lexical': LexicalModel,
        'prefix_suffix': PrefixSuffixLM,
    }
    Model = models[config.model_type]
    return Model(config, training=training, cpu=cpu)

def save(model: Model, sess: tf.compat.v1.Session, outdir: str):
    inputs = model.placeholders_dict()
    outputs = model.outputs_dict()

    save_model(
        sess,
        outdir,
        inputs=inputs,
        outputs=outputs,
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--load_checkpoint', type=str, default='./tmp/')
    parser.add_argument('--out_dir', type=str, default='./out/saved_model')
    parser.add_argument('--config', type=str, default='./out/config.json')

    args = parser.parse_args()

    config = Config()
    if args.config != '':
        config = Config.from_json(json.load(open(args.config, 'r')))

    # initialize new model
    model = get_model(config, training=False, cpu=True)

    with tf.compat.v1.Session() as sess:
        saver = tf.compat.v1.train.Saver()
        # Load the most recent model from a checkpoint path
        ckpt = tf.train.get_checkpoint_state(args.load_checkpoint)

        assert ckpt, f'unable to get checkpoint state from {args.load_checkpoint}'
        assert ckpt.model_checkpoint_path, f'no model checkpoint path for {ckpt}'

        saver.restore(sess, ckpt.model_checkpoint_path)
        save(model, sess, args.out_dir)


if __name__ == '__main__':
    main()
