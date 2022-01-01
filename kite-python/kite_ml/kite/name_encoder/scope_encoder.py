from typing import Dict, Any

import tensorflow as tf

from ..utils.segmented_data import SegmentedIndices, SegmentedIndicesFeed

from ..graph_encoder.embeddings import NodeEmbeddings


class Encoder(object):
    def __init__(self, nodes: NodeEmbeddings):
        self._nodes = nodes
        self._build()

    def _build(self):
        self._build_placeholders()
        self._build_scope_state()

    def _build_placeholders(self):
        with tf.name_scope('placeholders'):
            # shape [number of variables in batch]
            # sample_ids[i] = s means that means that variable i is part of sample s in the batch
            self._variable_node_ids = SegmentedIndices('variable_node_ids')

    def _build_scope_state(self):
        with tf.name_scope('build_scope_state'):
            # [num variable nodes in batch]
            self._variable_nodes_embedded: tf.Tensor = tf.gather(
                self._nodes.embeddings,
                self._variable_node_ids.indices,
                name='scope_nodes_embedded',
            )

            # reduce across variable nodes in each graph in the batch
            # shape [batch size, graph embedding depth]
            self._scope_state: tf.Tensor = tf.segment_max(
                self._variable_nodes_embedded,
                self._variable_node_ids.sample_ids,
                name='scope_state',
            )

    def feed_dict(self, feed: SegmentedIndicesFeed) -> Dict[tf.Tensor, Any]:
        return self._variable_node_ids.feed_dict(feed)

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        return self._variable_node_ids.dict()

    def scope_state(self) -> tf.Tensor:
        """
        :return: representation of all the variables in scope, shape [batch size, graph embedding depth]
        """
        return self._scope_state
