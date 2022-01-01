import unittest

import numpy as np
import tensorflow as tf

from .embeddings import CodeBookEmbedding, CodebookConfig


class TestCodebookEmbedding(unittest.TestCase):
    VOCAB = 6
    DEPTH = 4
    N_CODEBOOKS = 3
    N_ENTRIES = 2

    CODEBOOKS = np.array([
        [1., 2., 3., 4.],
        [5., 6., 7., 8.],

        [0., 1., 1., 0],
        [5., 5., 4., 2.],

        [2., 2., 2., 2.],
        [4., 3., 2., 1.],
    ], dtype=np.float32)

    CODES = np.array([
        [0, 0, 0],
        [0, 1, 1],
        [1, 0, 0],
        [1, 1, 1],
        [0, 1, 0],
        [0, 1, 0],
    ], dtype=np.uint8)

    assert CODEBOOKS.shape == (N_CODEBOOKS * N_ENTRIES, DEPTH)
    assert CODES.shape == (VOCAB, N_CODEBOOKS)

    def test_lookup_1d(self):
        tf.reset_default_graph()
        sess = tf.InteractiveSession()
        try:
            emb = CodeBookEmbedding(
                np.zeros((self.VOCAB, self.DEPTH)), "embedding", True, False,
                CodebookConfig(enabled=True, n_codebooks=self.N_CODEBOOKS, n_entries=self.N_ENTRIES))
            indices = tf.placeholder(tf.int32, shape=[None], name="indices")
            lookup = emb.lookup(indices)

            sess.run(tf.assign(sess.graph.get_tensor_by_name("embedding/codebooks:0"), self.CODEBOOKS))
            sess.run(tf.assign(sess.graph.get_tensor_by_name("embedding/codes:0"), self.CODES))

            res = sess.run(lookup, feed_dict={indices: [2, 0, 3]})
            expected = np.array([
                [7.,   9., 10., 10.],
                [3.,   5.,  6.,  6.],
                [14., 14., 13., 11.],
            ], dtype=np.float32)
            np.testing.assert_almost_equal(res, expected)

        finally:
            sess.close()

    def test_lookup_2d(self):
        tf.reset_default_graph()
        sess = tf.InteractiveSession()
        try:
            emb = CodeBookEmbedding(
                np.zeros((self.VOCAB, self.DEPTH)), "embedding", True, False,
                CodebookConfig(enabled=True, n_codebooks=self.N_CODEBOOKS, n_entries=self.N_ENTRIES))
            indices = tf.placeholder(tf.int32, shape=[None, 2], name="indices")
            lookup = emb.lookup(indices)

            sess.run(tf.assign(sess.graph.get_tensor_by_name("embedding/codebooks:0"), self.CODEBOOKS))
            sess.run(tf.assign(sess.graph.get_tensor_by_name("embedding/codes:0"), self.CODES))

            res = sess.run(lookup, feed_dict={indices: [[2, 1], [0, 3]]})
            expected = np.array([
                [[7.,   9., 10., 10.],
                 [10., 10.,  9.,  7.]],
                [[3.,   5.,  6.,  6.],
                 [14., 14., 13., 11.]],
            ])
            np.testing.assert_almost_equal(res, expected)

        finally:
            sess.close()
