from typing import Dict, List

import logging
import shutil

import tensorflow as tf

def purge_dir(path):
    try:
        shutil.rmtree(path)
    except OSError:
        pass


def save_model(sess: tf.compat.v1.Session, model_dir: str, inputs: Dict[str, tf.Tensor], outputs: Dict[str, tf.Tensor]):
    logging.info("saving model to {}".format(model_dir))
    tf.compat.v1.saved_model.simple_save(sess, model_dir, inputs=inputs, outputs=outputs)


def save_frozen_model(sess: tf.compat.v1.Session, out_file: str, output_names: List[str]):
    logging.info("saving frozen model to {}".format(out_file))
    # Querying a tensor's name in a session produces one with a :0 suffix, but this suffix does not exist in the
    # GraphDef
    output_names = [n.replace(":0", "") for n in output_names]

    graph_def = tf.compat.v1.graph_util.convert_variables_to_constants(sess, sess.graph_def, output_names)
    with tf.io.gfile.GFile(out_file, "wb") as f:
        f.write(graph_def.SerializeToString())


def load_saved_model(sess: tf.compat.v1.Session, model_dir: str):
    tf.compat.v1.saved_model.loader.load(sess, [tf.saved_model.tag_constants.SERVING], model_dir)

def load_frozen_model(frozen_graph_filename: str, prefix:str = ""):
    # We load the protobuf file from the disk and parse it to retrieve the
    # unserialized graph_def
    with tf.io.gfile.GFile(frozen_graph_filename, "rb") as f:
        graph_def = tf.GraphDef()
        graph_def.ParseFromString(f.read())

    # Then, we import the graph_def into a new Graph and returns it
    with tf.Graph().as_default() as graph:
        # The name var will prefix every op/nodes in your graph
        # Since we load everything in a new graph, this is not needed
        tf.import_graph_def(graph_def, name=prefix)
    return graph