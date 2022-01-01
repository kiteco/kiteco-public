from typing import Dict

import argparse
import json
import tensorflow as tf

from kite.infer_expr.config import MetaInfo, Config
from kite.infer_production.index import Production
from kite.infer_expr.model import Model
from kite.utils.save import save_frozen_model



def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--meta_info", type=str, required=True)
    parser.add_argument("--out_frozen_model", type=str, required=True)
    parser.add_argument("--node_depth", type=int, default=0)
    parser.add_argument("--vocab_scale", type=float, default=1.0)
    args = parser.parse_args()

    meta_info = MetaInfo.from_json(json.load(open(args.meta_info, 'r')))

    # mutate the meta-info to change the vocabulary size
    type_subtoken_size = int(len(meta_info.type_subtoken_index) * args.vocab_scale)
    name_subtoken_size = int(len(meta_info.name_subtoken_index) * args.vocab_scale)

    type_size = int(len(meta_info.call.arg_indices) * args.vocab_scale)
    new_call = meta_info.call._replace(arg_indices={str(i): i for i in range(type_size)})

    production_size = int(meta_info.production.vocab() * args.vocab_scale)
    new_production = meta_info.production._replace(
        productions={str(i): Production(id=str(i), children=[]) for i in range(production_size)},
        indices={str(i): i for i in range(production_size)},
    )

    meta_info = meta_info._replace(
        name_subtoken_index={str(i): i for i in range(name_subtoken_size)},
        type_subtoken_index={str(i): i for i in range(type_subtoken_size)},
        production=new_production,
        call=new_call,
    )

    config = Config()

    gc = config.graph
    if args.node_depth != 0:
        gc = gc._replace(type_depth=args.node_depth, subtoken_depth=args.node_depth)
    config = config._replace(graph=gc)

    model = Model(config, meta_info, compressed=True)

    with tf.Session() as sess:
        sess.run(tf.global_variables_initializer())

        print(f"saving frozen model to {args.out_frozen_model}")
        outputs: Dict[str, tf.Tensor] = {
            model.production_model().pred().name: model.production_model().pred(),
            model.name_model().pred().name: model.name_model().pred(),
        }

        save_frozen_model(sess, args.out_frozen_model, list(outputs.keys()))


if __name__ == "__main__":
    main()
