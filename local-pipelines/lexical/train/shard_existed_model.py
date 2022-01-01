import tensorflow as tf

horovod_enabled = True
try:
    import horovod.tensorflow as hvd
except ImportError:
    horovod_enabled = False

from tensorflow.python.framework import graph_util
from tensorflow.core.framework import graph_pb2
import copy
import argparse


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--in_saved_model', type=str, default='out')
    parser.add_argument('--out_frozen_model', type=str, default='lexical_model.frozen.pb')
    args = parser.parse_args()

    # This part is to add the `prediction/last` op for the old models that don't have it
    # and the `prediction/last_logits` op
    with tf.Session() as sess:
        tf.saved_model.loader.load(sess, ['serve'], args.in_saved_model)
        graph = tf.get_default_graph()

        # Add the new operation `prediction/last`
        h = graph.get_tensor_by_name('transformer/ln_f/add_1:0')

        wte = graph.get_tensor_by_name('wte:0')
        last_logits = tf.matmul(h[:, -1, :], wte, transpose_b=True)
        with tf.name_scope('prediction'):
            _ = tf.nn.softmax(last_logits, name='last', axis=-1)
            _ = tf.identity(last_logits, name='last_logits')

        wte_matrix = wte.eval().tolist()

        output_names = ['prediction/pred', 'prediction/last', 'prediction/last_logits']
        graph_def = graph_util.convert_variables_to_constants(sess, sess.graph_def, output_names)

    # Create a placeholder for wte in a custom_graph so that we can use the same name
    custom_graph = tf.Graph()
    with custom_graph.as_default():
        wte_original = tf.constant(wte_matrix, name='wte/original')
        wte_original_read = tf.identity(wte_original, name='wte/original/read')
        wte_indices = tf.placeholder(dtype=tf.int64, shape=[None], name='wte/indices')
        wte_axis = tf.constant(0, name='wte/axis')
        wte_selected = tf.gather(wte_original, wte_indices, axis=wte_axis, name='wte')
        wte_read = tf.identity(wte_selected, name='wte/read')

    # Create a new graph def (pretty hacky, just replace the tensors associated with the original names)
    new_graph_def = graph_pb2.GraphDef()
    new_graph_def.node.extend([wte_original.op.node_def])
    new_graph_def.node.extend([wte_indices.op.node_def])
    new_graph_def.node.extend([wte_original_read.op.node_def])
    new_graph_def.node.extend([wte_axis.op.node_def])

    for node in graph_def.node:
        if node.name == 'wte':
            new_graph_def.node.extend([wte_selected.op.node_def])
        elif node.name == 'wte/read':
            new_graph_def.node.extend([wte_read.op.node_def])
        else:
            new_graph_def.node.extend([copy.deepcopy(node)])

    # Save the new graph def as a frozen model
    with tf.Session() as sess:
        with tf.Graph().as_default() as graph:
            # Save a new frozen model
            tf.import_graph_def(new_graph_def, name='')

            with tf.gfile.GFile(args.out_frozen_model, 'wb') as f:
                f.write(new_graph_def.SerializeToString())


if __name__ == "__main__":
    main()
