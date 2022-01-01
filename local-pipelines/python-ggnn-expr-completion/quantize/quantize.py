from typing import Dict, List, NamedTuple, Tuple

import argparse
import json
import numpy as np
import os
import tensorflow as tf

from kite.infer_expr.config import MetaInfo, Config
from kite.infer_expr.model import Model
from kite.utils.embeddings import QUANTIZE_DTYPE, QUANTIZE_MODE
from kite.utils.save import save_frozen_model


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--meta_info', type=str, required=True)
    parser.add_argument('--in_saved_model', type=str, required=True)
    parser.add_argument('--out_frozen_model', type=str, required=True)
    args = parser.parse_args()

    checkpoint_path = os.path.join(args.in_saved_model, "variables/variables")

    meta_info = MetaInfo.from_json(json.load(open(args.meta_info, 'r')))
    config = Config()

    # First, determine which variables are quantizable
    Model(config, meta_info, compressed=True)
    # names of the quantizable variables
    q_var_names = []
    vars = tf.global_variables()
    for v in vars:
        if v.name.endswith("/quantized:0"):
            q_var_names.append(v.name.replace("/quantized:0", ""))

    # names of the regular (non-quantizable) variables
    reg_vars = {v.name: v for v in vars if not any([v.name.startswith(q) for q in q_var_names])}
    for var_name, var in reg_vars.items():
        print("\nWILL NOT quantize:", var_name)
        print("  shape:", var.shape)
        print("  bytes (float32):", int(np.prod(var.shape) * 4))
        print("  bytes (int8):", int(np.prod(var.shape)))

    print("\n============================")

    # Now, get the min/max values of each variable and quantize them
    tf.reset_default_graph()
    Model(config, meta_info, compressed=False)

    saver = tf.train.Saver()

    quantized_vals: Dict[str, np.ndarray] = {}
    ranges: Dict[str, Tuple[float, float]] = {}

    with tf.Session() as sess:
        saver.restore(sess, checkpoint_path)

        var_vals = sess.run({var: var + ":0" for var in q_var_names})

        for var_name, val in var_vals.items():
            assert val.dtype == np.float32

            min_range, max_range = np.min(val), np.max(val)

            quantized_val, _, _ = sess.run(
                tf.contrib.quantization.quantize_v2(val, min_range=min_range, max_range=max_range,
                                                    mode=QUANTIZE_MODE, T=QUANTIZE_DTYPE))
            quantized_vals[var_name] = quantized_val
            ranges[var_name] = (min_range, max_range)

            print("\nWILL quantize:", var_name)
            print("  shape:", val.shape)
            print("  range: {} to {}".format(min_range, max_range))
            print("  bytes (float32):", int(np.prod(val.shape) * 4))
            print("  bytes (int8):  ", int(np.prod(val.shape)))

    # Finally, set the quantized values in the quantized model
    tf.reset_default_graph()
    model = Model(config, meta_info, compressed=True)

    reg_saver = tf.train.Saver(var_list=[v for v in tf.global_variables() if v.name in reg_vars.keys()])

    with tf.Session() as sess:
        # set the quantized values along with the min/max ranges
        for var_name in q_var_names:
            print(f"setting quantized values for {var_name}")
            quantized = quantized_vals[var_name]
            min_range, max_range = ranges[var_name]

            sess.run([
                tf.assign(sess.graph.get_tensor_by_name(var_name + "/quantized:0"), quantized),
                tf.assign(sess.graph.get_tensor_by_name(var_name + "/min:0"), min_range),
                tf.assign(sess.graph.get_tensor_by_name(var_name + "/max:0"), max_range),
                ])

        # set the regular variables
        reg_saver.restore(sess, checkpoint_path)

        print(f"saving frozen model to {args.out_frozen_model}")
        outputs = model.outputs_dict()

        save_frozen_model(sess, args.out_frozen_model, list(outputs.keys()))


if __name__ == "__main__":
    main()

