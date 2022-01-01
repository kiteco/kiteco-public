import argparse
import json
import os
import tensorflow as tf

from model.config import Config, update
from model.model import LexicalModel
from kite.utils.save import save_model, save_frozen_model
from kite.model.model import Config as BaseConfig, AdamTrainer


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--in_saved_model', type=str, default='original_saved_model')
    parser.add_argument('--extra_vocab_size', type=int, default=100)
    parser.add_argument('--original_config', type=str, default='initial/config.json')
    parser.add_argument('--new_embedding', type=str, default='')
    parser.add_argument('--out_saved_model', type=str, default='updated_saved_model')
    parser.add_argument('--out_config', type=str, default='')
    args = parser.parse_args()

    config = Config.from_json(json.load(open(args.original_config, 'r')))
    new_vocab_size = config.n_vocab + args.extra_vocab_size + 1 # HACK: For SEP token
    config = update(config, {'n_vocab': new_vocab_size})
    model = LexicalModel(config=config)

    with open(args.out_config, 'w') as f:
        json.dump(dict(config._asdict()), f)

    with open(args.new_embedding, 'r') as f:
        updated_wte = json.load(f)

    print(len(updated_wte), len(updated_wte[0]))
    with tf.compat.v1.Session() as sess:
        # Load all the other variables from the original model
        variables_to_restore = [var for var in tf.compat.v1.global_variables() if 'wte' not in var.name]

        saver = tf.compat.v1.train.Saver(var_list=variables_to_restore)
        saver.restore(sess, os.path.join(args.in_saved_model, "variables/variables"))
        graph = tf.compat.v1.get_default_graph()
        wte = graph.get_tensor_by_name('wte:0')

        # Update the wte matrix with our updated embedding
        sess.run(tf.compat.v1.assign(wte, updated_wte))

        save_model(
            sess,
            args.out_saved_model,
            inputs=model.placeholders_dict(),
            outputs=model.outputs_dict(),
        )


if __name__ == '__main__':
    main()
