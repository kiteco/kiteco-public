from typing import Dict, List, NamedTuple, Tuple

import argparse
import json
import numpy as np
import os
import tensorflow as tf

from kite.infer_expr.config import MetaInfo, Config
from kite.infer_expr.model import Model
from kite.utils.save import save_frozen_model

from kite.compress_embeddings.codebook.embed_compress import EmbeddingCompressor


# We scale the embeddings by this factor before training the embeddings; seems to provide better performance
SCALE_FACTOR = 100.


class CBParams(NamedTuple):
    n_codebooks: int
    n_entries: int
    shape: Tuple[int, int]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--meta_info', type=str, required=True)
    parser.add_argument('--in_saved_model', type=str, required=True)
    parser.add_argument('--out_frozen_model', type=str, required=True)
    parser.add_argument('--models_path', type=str, required=True,
                        help="the path to which the codebook model checkpoints are saved")
    parser.add_argument('--train', type=bool, default=False,
                        help="if True, the codebook models are trained; otherwise, they are recovered from models_path")
    parser.add_argument('--max_epochs', type=int, default=1500)

    args = parser.parse_args()

    checkpoint_path = os.path.join(args.in_saved_model, "variables/variables")

    meta_info = MetaInfo.from_json(json.load(open(args.meta_info, 'r')))
    config = Config()

    # First, determine which variables are compressible
    Model(config, meta_info, compressed=True)
    # (names of the codebook variable) => (params of that variable)
    cb_params = {}
    vars = tf.global_variables()
    for v in vars:
        if v.name.endswith("/codebooks:0"):
            cb_var_name = v.name.replace("/codebooks:0", "")
            # figure out the codebook params from the shape of the codebooks/codes vars
            # cbs_entries_prod = n_codebooks * n_entries
            cbs_entries_prod, depth = map(int, v.shape)
            codes_var = [v for v in vars if v.name == cb_var_name + "/codes:0"][0]
            vocab, n_codebooks = map(int, codes_var.shape)
            n_entries = cbs_entries_prod // n_codebooks
            cb_params[cb_var_name] = CBParams(n_codebooks=n_codebooks, n_entries=n_entries, shape=(vocab, depth))

    # names of the regular (non-codebook) variables
    reg_vars = {v.name: v for v in vars if not any([v.name.startswith(cb) for cb in cb_params.keys()])}

    for var_name, var in reg_vars.items():
        print("\nWILL NOT compress:", var_name)
        print("  shape:", var.shape)
        print("  bytes (original float32):", int(np.prod(var.shape) * 4))

    print("\n============================")

    for var_name, params in cb_params.items():
        print("\nWILL compress:", var_name)
        print("  shape:", params.shape)
        print("  num codebooks:", params.n_codebooks)
        print("  num entries  :", params.n_entries)
        original_bytes = int(np.prod(params.shape) * 4)
        vocab, depth = params.shape
        comp_bytes = (params.n_entries * params.n_codebooks * depth * 4) + (vocab * params.n_codebooks)
        print("  bytes (original float32):", original_bytes)
        print("  bytes (compressed)      :", comp_bytes)
        print("  compression ratio       :", original_bytes / comp_bytes)

    # we get the values of each variable
    tf.reset_default_graph()
    Model(config, meta_info, compressed=False)
    saver = tf.train.Saver()

    with tf.Session() as sess:
        saver.restore(sess, checkpoint_path)
        original_values = sess.run({var: var + ":0" for var in cb_params.keys()})

    # we learn the codebooks and codes for each variable
    cb_books: Dict[str, np.ndarray] = {}
    cb_codes: Dict[str, np.ndarray] = {}

    for var_name, emb_matrix in original_values.items():
        params = cb_params[var_name]
        if args.train:
            train_codebooks(args.models_path, var_name, emb_matrix, params.n_codebooks, params.n_entries, args.max_epochs)
        codebooks, codes = load_codebooks(
            args.models_path, var_name, emb_matrix, params.n_codebooks, params.n_entries)
        cb_books[var_name] = codebooks
        cb_codes[var_name] = codes

    # finally, we dump the learned codebooks/codes into the variables of the compressed models
    tf.reset_default_graph()
    model = Model(config, meta_info, compressed=True)

    reg_saver = tf.train.Saver(var_list=[v for v in tf.global_variables() if v.name in reg_vars.keys()])

    with tf.Session() as sess:
        # set the appropriate variables for the codebook embeddings
        for var_name in cb_books.keys():
            print(f"setting codebook/codes for {var_name}")
            sess.run([
                tf.assign(sess.graph.get_tensor_by_name(var_name + "/codebooks:0"), cb_books[var_name]),
                tf.assign(sess.graph.get_tensor_by_name(var_name + "/codes:0"), cb_codes[var_name]),
            ])

        # set the regular variables
        reg_saver.restore(sess, checkpoint_path)

        print(f"saving frozen model to {args.out_frozen_model}")
        outputs = model.outputs_dict()


        save_frozen_model(sess, args.out_frozen_model, list(outputs.keys()))


def train_codebooks(models_path: str, var_name: str, original_embeddings: np.ndarray,
                    n_codebooks: int, n_entries: int, max_epochs: int):
    tf.reset_default_graph()
    print("training codebook model on:", var_name)
    model_path = cb_model_path(models_path, var_name)
    tb_path = cb_tensorboard_path(models_path, var_name)
    print("tensorboard will be saved to", tb_path)
    compressor = EmbeddingCompressor(n_codebooks, n_entries, model_path)
    compressor.train(original_embeddings * SCALE_FACTOR, tb_path, max_epochs=max_epochs)
    compressor.export(original_embeddings * SCALE_FACTOR, model_path)


def load_codebooks(models_path: str, var_name: str, original_embeddings: np.ndarray,
                   n_codebooks: int, n_entries: int) -> (np.ndarray, np.ndarray):
    model_path = cb_model_path(models_path, var_name)
    codebook = np.load(f"{model_path}.codebook.npy") / SCALE_FACTOR
    assert codebook.shape == (n_codebooks * n_entries, original_embeddings.shape[1])
    codes = np.load(f"{model_path}.codes.npy")
    assert codes.shape == (original_embeddings.shape[0], n_codebooks)
    return codebook, codes


def cb_model_path(models_path: str, var_name: str) -> str:
    return os.path.join(models_path, var_name.replace("/", "_"), "model")


def cb_tensorboard_path(models_path: str, var_name: str) -> str:
    return os.path.join(models_path, "tensorboard", var_name.replace("/", "_"))


if __name__ == "__main__":
    main()

