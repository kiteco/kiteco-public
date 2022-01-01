
import unittest

import tensorflow as tf

import numpy as np

from .segment import segment_topk


class TestSegmentTopK(unittest.TestCase):
    def test_top1(self):
        sess = tf.InteractiveSession()
        try:
            preds = tf.constant(np.array([
                .3,
                .7,
                .33,
                .33,
                .66,
                1.,
            ]))

            sample_ids = tf.constant(np.array([
                0,
                0,
                1,
                1,
                1,
                2,
            ]))

            new_preds, idxs, new_ids = segment_topk(preds, sample_ids, 1, 'top1')

            fetches = [new_preds, idxs, new_ids]
            fetches = sess.run(fetches=fetches)

            actual_preds = fetches[0]
            actual_idxs = fetches[1]
            actual_new_ids = fetches[2]

            expected_preds = np.array([.7, .66, 1.])
            self.assertTrue(np.allclose(actual_preds, expected_preds),
                            'expected {} got {}'.format(expected_preds, actual_preds))

            expected_idxs = np.array([1, 4, 5])
            self.assertTrue(np.all(actual_idxs == expected_idxs),
                            'expected {} got {}'.format(expected_idxs, actual_idxs))

            expected_ids = np.array([0, 1, 2])
            self.assertTrue(np.all(expected_ids == actual_new_ids),
                            'expected {} got {}'.format(expected_ids, actual_new_ids))
        finally:
            sess.close()

    def test_top2(self):
        sess = tf.InteractiveSession()
        try:
            preds = tf.constant(np.array([
                .3,
                .7,
                .33,
                .33,
                .66,
                1.,
            ]))

            sample_ids = tf.constant(np.array([
                0,
                0,
                1,
                1,
                1,
                2,
            ]))

            new_preds, idxs, new_ids = segment_topk(preds, sample_ids, 2, 'top2')

            fetches = [new_preds, idxs, new_ids]
            fetches = sess.run(fetches=fetches)

            actual_preds = fetches[0]
            actual_idxs = fetches[1]
            actual_new_ids = fetches[2]

            expected_preds = np.array([.7, .3, .66, .33, 1.])
            self.assertTrue(np.allclose(actual_preds, expected_preds),
                            'expected {} got {}'.format(expected_preds, actual_preds))

            expected_idxs = np.array([1, 0, 4, 2, 5])
            self.assertTrue(np.all(actual_idxs == expected_idxs),
                            'expected {} got {}'.format(expected_idxs, actual_idxs))

            expected_ids = np.array([0, 0, 1, 1, 2])
            self.assertTrue(np.all(expected_ids == actual_new_ids),
                            'expected {} got {}'.format(expected_ids, actual_new_ids))
        finally:
            sess.close()

    def test_topEmpty(self):
        sess = tf.InteractiveSession()
        try:
            preds = tf.placeholder_with_default(
                np.empty((0,), dtype=np.float64), shape=[None],
            )

            sample_ids = tf.placeholder_with_default(
                np.empty((0,), dtype=np.int64), shape=[None]
            )

            new_preds, idxs, new_ids = segment_topk(preds, sample_ids, 2, 'top2')

            fetches = [new_preds, idxs, new_ids]
            fetches = sess.run(fetches=fetches)

            actual_preds = fetches[0]
            actual_idxs = fetches[1]
            actual_new_ids = fetches[2]

            expected_preds = np.empty((0,), dtype=np.float64)
            self.assertTrue(np.allclose(actual_preds, expected_preds),
                            'expected {} got {}'.format(expected_preds, actual_preds))

            expected_idxs = np.empty((0,), dtype=np.int64)
            self.assertTrue(np.all(actual_idxs == expected_idxs),
                            'expected {} got {}'.format(expected_idxs, actual_idxs))

            expected_ids = np.empty((0,), dtype=np.int64)
            self.assertTrue(np.all(expected_ids == actual_new_ids),
                            'expected {} got {}'.format(expected_ids, actual_new_ids))
        finally:
            sess.close()
