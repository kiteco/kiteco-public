from typing import NamedTuple

import tensorflow as tf

from ..utils.segmented_data import SegmentedIndices

from ..utils.embeddings import CodeBookEmbedding, CodebookConfig
from ..utils.initializers import glorot_init

from ..model.config import PoolingOpt


class NodeEmbeddings(NamedTuple):
    embeddings: tf.Tensor
    depth: int


class Config(NamedTuple):
    type_depth: int
    subtoken_depth: int
    type_codebook: CodebookConfig
    subtoken_codebook: CodebookConfig

    subtoken_pooling: PoolingOpt
    type_pooling: PoolingOpt


class Embeddings(object):
    def __init__(self, type_vocab: int, subtoken_vocab: int, compressed: bool, reuse: bool, config: Config):
        self._type_vocab = type_vocab
        self._subtoken_vocab = subtoken_vocab
        self._reuse = reuse
        self._compressed = compressed
        self._config = config
        self._build_embeddings()

    def _build_embeddings(self):
        with tf.name_scope("embeddings"):
            # [total number of type symbols, type embed depth]
            self._type_embedding = CodeBookEmbedding(
                glorot_init(self._type_vocab, self._config.type_depth),
                name="type",
                compressed=self._compressed,
                reuse=self._reuse,
                config=self._config.type_codebook,
            )

            self._subtoken_embedding = CodeBookEmbedding(
                glorot_init(self._subtoken_vocab, self._config.subtoken_depth),
                name="subtoken",
                compressed=self._compressed,
                reuse=self._reuse,
                config=self._config.subtoken_codebook,
            )

    def depth(self) -> int:
        return self._config.type_depth + self._config.subtoken_depth

    def embed(self, types: SegmentedIndices, subtokens: SegmentedIndices):
        """
        embeddings for the specified types and subtokens,
        must have len(unique(types.sample_ids)) == len(unique(subtokens.sample_ids))
        :param types: ids for the type embeddings, sample_ids[i] = j implies type i is for node j
        :param subtokens: ids for subtoken embeddings, sample_ids[i] = j implies sub-token i is for node j
        :return: node embeddings, shape [num nodes, depth]
        """
        with tf.name_scope("lookup_embedding"):
            # [num nodes, num types, type embed depth]
            all_node_types = self._type_embedding.lookup(types.indices)

            if self._config.type_pooling is PoolingOpt.SUM:
                type_pool_op = tf.segment_sum
            elif self._config.type_pooling is PoolingOpt.AVG:
                type_pool_op = tf.segment_mean
            elif self._config.type_pooling is PoolingOpt.MAX:
                type_pool_op = tf.segment_max
            else:
                raise AssertionError("unrecognized type pooling type {}".format(self._config.type_pooling))

            # [num nodes, type embed depth]
            node_types = type_pool_op(all_node_types, types.sample_ids, name='node_types')

            # [num nodes, num sub-tokens, sub-token embed depth]
            all_node_subtokens = self._subtoken_embedding.lookup(subtokens.indices)

            if self._config.subtoken_pooling is PoolingOpt.SUM:
                subtoken_pool_op = tf.segment_sum
            elif self._config.subtoken_pooling is PoolingOpt.AVG:
                subtoken_pool_op = tf.segment_mean
            elif self._config.subtoken_pooling is PoolingOpt.MAX:
                subtoken_pool_op = tf.segment_max
            else:
                raise AssertionError("unrecognized sub-token pooling type {}".format(self._config.type_pooling))

            # [num nodes, sub token embed depth]
            node_subtokens = subtoken_pool_op(all_node_subtokens, subtokens.sample_ids, name='node_subtokens')

            # [num nodes, depth]
            return tf.concat([node_types, node_subtokens], axis=1, name='node_embeddings')
