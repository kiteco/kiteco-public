from typing import Dict, Any, Optional

import tensorflow as tf

import numpy as np

from .usage_feed import Feed

from ..graph_encoder.embeddings import NodeEmbeddings

from ..utils.segmented_data import SegmentedIndices


class Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            # node ids for the candidate variable usage nodes for each graph in the batch
            # shape [number of variables in all graphs in batch]
            # e.g let s index over samples in the batch then
            # num_vars_batch = number of variables in all graphs in batch = sum_s(num vars in graph s)
            # sample_ids[i] = s means that means that variable i is part of sample s in the batch
            self.usages = SegmentedIndices('usages')

    def dict(self) -> Dict[str, tf.Tensor]:
        return self.usages.dict()

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return self.usages.feed_dict(feed.usages)


class _TrainPlaceholders(object):
    def __init__(self):
        with tf.name_scope('extras'):
            # types for each variable for debugging,
            # [num vars in batch]
            self.types: tf.Tensor = tf.placeholder_with_default(
                np.empty((0,), dtype=np.string_),
                shape=[None], name='variable_types',
            )

            # names for each variable for debugging,
            # [num vars in batch]
            self.names: tf.Tensor = tf.placeholder_with_default(
                np.empty((0,), dtype=np.string_),
                shape=[None], name='variable_names',
            )

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.types: feed.types,
            self.names: feed.names,
        }


class Encoder(object):
    def __init__(self, nodes: NodeEmbeddings):
        self._nodes = nodes
        self._build()

    def _build(self):
        self._build_placeholders()
        self._build_usage_states()

    def _build_placeholders(self):
        self._placeholders = Placeholders()

        self._train_placeholders = _TrainPlaceholders()

    def _build_usage_states(self):
        with tf.name_scope('lookup_usage_embeddings'):
            # shape [num vars in batch, node embedding depth]
            self._usage_node_states: tf.Tensor = tf.gather(
                self._nodes.embeddings,
                self._placeholders.usages.indices,
                name='usage_node_states'
            )

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        fd = self._placeholders.feed_dict(feed)
        fd.update(self._train_placeholders.feed_dict(feed))
        return fd

    def sample_ids(self) -> tf.Tensor:
        """
        sample ids for the usage node states
        :return: shape [num variables in batch]
        """
        return self._placeholders.usages.sample_ids

    def placeholders(self) -> Placeholders:
        return self._placeholders

    def usage_node_states(self) -> tf.Tensor:
        """
        :return: usage node states, shape [num variables in batch, node embedding depth]
        """
        return self._usage_node_states

    def types(self, idxs: tf.Tensor, name: Optional[str]=None) -> tf.Tensor:
        """
        return types of variables specified by idxs
        :param idxs: ids of variables to retrieve types for
        :param name: name of the resulting tensor
        :return: a tensor of strings with the same shape as idxs
        """
        return tf.gather(self._train_placeholders.types, idxs, name=name)

    def names(self, idxs: tf.Tensor, name: Optional[str]=None) -> tf.Tensor:
        """
        SEE: self.types(...)
        """
        return tf.gather(self._train_placeholders.names, idxs, name=name)

    def names_and_types(self, idxs: tf.Tensor, name: Optional[str]=None) -> tf.Tensor:
        """
        SEE: self.var_type(...)
        """

        if name:
            name = name+'_names'
        names = self.names(idxs, name=name)

        if name:
            name = name+'_types'

        types = self.types(idxs, name=name)

        return tf.string_join([names, types], separator="::", name=name)
