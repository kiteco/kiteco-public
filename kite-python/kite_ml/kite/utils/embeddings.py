from typing import NamedTuple, Optional

import numpy as np
import tensorflow as tf

QUANTIZE_DTYPE = tf.qint8
QUANTIZE_MODE = 'MIN_COMBINED'


class QuantizableEmbedding(object):
    def __init__(self, init_value: np.ndarray, name: str, compressed: bool):
        self._quantized = compressed
        self._name = name

        if self._quantized:
            with tf.name_scope(name):
                self._quantized = tf.Variable(np.zeros_like(init_value), dtype=QUANTIZE_DTYPE, name="quantized")
                self._min = tf.Variable(0, dtype=tf.float32, name="min")
                self._max = tf.Variable(0, dtype=tf.float32, name="max")
        else:
            self._value = tf.Variable(init_value, dtype=np.float32, name=name)

    def lookup(self, indices: tf.Tensor, name: Optional[str] = None) -> tf.Tensor:
        name = name or self._name + "/lookup"

        if self._quantized:
            return tf.contrib.quantization.dequantize(
                tf.nn.embedding_lookup(self._quantized, indices),
                min_range=self._min, max_range=self._max, mode=QUANTIZE_MODE, name=name)
        else:
            return tf.nn.embedding_lookup(self._value, indices, name=name)


class CodebookConfig(NamedTuple):
    enabled: bool
    n_codebooks: int
    n_entries: int


class CodeBookEmbedding(object):
    """Based on https://arxiv.org/abs/1711.01068"""
    def __init__(self,
                 init_value: np.ndarray,
                 name: str,
                 compressed: bool,
                 reuse: bool,
                 config: CodebookConfig):
        assert len(init_value.shape) == 2, "only 2D embeddings supported"
        assert config.n_entries <= 256, "number of entries must be <= 256 in order to use 8-bit codes"

        self._name = name
        self._compressed = compressed and config.enabled
        self._n_codebooks = config.n_codebooks
        self._n_entries = config.n_entries
        self._vocab, self._depth = init_value.shape

        # dimensions:
        # D = embedding depth
        # V = vocab size
        # M = number of codebooks
        # K = number of codes per codebook

        if self._compressed:
            with tf.variable_scope(name, reuse=reuse):
                # [M * K, D]
                self._codebooks = tf.get_variable(
                    name='codebooks', shape=[self._n_codebooks * self._n_entries, self._depth],
                    dtype=tf.float32, initializer=tf.zeros_initializer(),
                )
                # [V, M]
                self._codes = tf.get_variable(
                    name='codes', shape=[self._vocab, self._n_codebooks],
                    dtype=tf.uint8, initializer=tf.zeros_initializer(),
                )
        else:
            # TODO: just send in initializer and shape
            self._value = tf.get_variable(
                shape=init_value.shape,
                name=name, initializer=tf.constant_initializer(init_value),
            )

    def lookup(self, indices: tf.Tensor, name: Optional[str] = None) -> tf.Tensor:
        name = name or self._name + "_lookup"
        if not self._compressed:
            return tf.nn.embedding_lookup(self._value, indices, name=name)

        with tf.name_scope(name):
            if len(indices.shape) == 1:
                return self._lookup_1d(indices)
            elif len(indices.shape) == 2:
                return self._lookup_2d(indices)
            else:
                raise Exception("only 1D and 2D lookup supported")

    def _lookup_2d(self, indices: tf.Tensor) -> tf.Tensor:
        assert len(indices.shape) == 2
        # N = indices shape 0
        # P = indices shape 1

        # [N * P]
        flattened = tf.reshape(indices, (-1,))

        # [N * P, D]
        lookup = self._lookup_1d(flattened, lookup_op_name=None)

        # [N, P, D]
        return tf.reshape(lookup, (-1, int(indices.shape[1]), self._depth), name="lookup")

    def _lookup_1d(self, indices: tf.Tensor, lookup_op_name: Optional[str] = "lookup") -> tf.Tensor:
        assert len(indices.shape) == 1
        # N = length of indices
        n = tf.cast(tf.shape(indices)[0], tf.int64)

        # [N, M]
        idx_codes = tf.cast(tf.nn.embedding_lookup(self._codes, indices), dtype=tf.int64)
        # we want the codes to represent the actual indices in the codebook matrix, and not the indices within the
        # respective codebooks
        idx_codes += (tf.range(self._n_codebooks, dtype=tf.int64) * self._n_entries)

        # [N * M, 1]
        dim2 = tf.reshape(idx_codes, (-1, 1))

        # [N, 1]
        ascending = tf.expand_dims(
            tf.range(n, dtype=tf.int64),
            axis=1)
        # [N, M]
        tiled = tf.tile(ascending, (1, self._n_codebooks))
        # [N * M, 1]
        dim1 = tf.reshape(tiled, (-1, 1))

        # [N * M, 2]
        sparse_idxs = tf.concat((dim1, dim2), axis=1)
        # [N * M]
        sparse_values = tf.ones(n * self._n_codebooks, dtype=tf.float32)

        # [N, M * K] sparse tensor with N * M entries
        sparse_one_hot = tf.SparseTensor(
            sparse_idxs,
            sparse_values,
            (n, self._n_codebooks * self._n_entries))

        # [N, D] dense tensor
        return tf.sparse_tensor_dense_matmul(sparse_one_hot, self._codebooks, name=lookup_op_name)
