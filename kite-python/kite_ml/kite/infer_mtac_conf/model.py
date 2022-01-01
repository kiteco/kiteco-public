from typing import Any, Dict

from .config import Config
from .feed import Feed

from ..utils.initializers import glorot_init

import numpy as np
import tensorflow as tf


class Placeholders(object):
    def __init__(self):
        contextual_depth = Feed.contextual_feature_depth()
        comp_depth = Feed.comp_feature_depth()

        with tf.name_scope('placeholders'):
            # size: [batch size, contextual feature depth]
            self.contextual_features: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[None, contextual_depth], name='contextual_features')

            # size: [total completion count, completion feature depth]
            # where total completion count = sum of completion counts for all samples in batch
            self.completion_features: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[None, comp_depth], name='completion_features')

            # size: [total completion count]
            self.sample_ids: tf.Tensor = tf.placeholder(dtype=tf.int32, shape=[None], name='sample_ids')

            # size: [total completion size]
            self.labels: tf.Tensor = tf.placeholder(dtype=tf.int32, shape=[None], name='labels')

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.contextual_features: feed.contextual_features,
            self.completion_features: feed.comp_features,
            self.sample_ids: feed.sample_ids,
            self.labels: feed.labels,
        }

    def dict(self) -> Dict[str, tf.Tensor]:
        return {
            self.contextual_features.name: self.contextual_features,
            self.completion_features.name: self.completion_features,
            self.sample_ids.name: self.sample_ids,
        }


class Model(object):
    """Predicts the probability that a given call completion is the correct one."""
    def __init__(self, config: Config):
        self._config = config
        self._placeholders = Placeholders()
        self._build()

    def _build(self):
        # size: [total completion count, contextual feature depth]
        expanded_contextual = tf.gather(
            self._placeholders.contextual_features, self._placeholders.sample_ids)

        # size: [total completion count, total feature depth]
        # where total_feature_depth = contextual_feature_depth + completion_feature_depth
        concatenated_features = tf.concat([expanded_contextual, self._placeholders.completion_features], axis=1)

        self.total_depth = Feed.contextual_feature_depth() + Feed.comp_feature_depth()

        self._weights = tf.Variable(glorot_init(self.total_depth, 1), dtype=tf.float32, name='weights')

        self._logits = tf.matmul(concatenated_features, self._weights)

        self._pred: tf.Tensor = self._build_pred()

    def _build_pred(self) -> tf.Tensor:
        with tf.name_scope('pred'):
            # size: [total completion count, 1]
            return tf.nn.sigmoid(self._logits, name='pred')

    def pred(self) -> tf.Tensor:
        return self._pred

    def weights(self) -> tf.Variable:
        return self._weights

    def set_weights(self, sess: tf.Session, weights: np.ndarray):
        """
        :param sess: tensorflow session
        :param weights: [<self.total_depth>, 1 array]
        """
        sess.run(tf.assign(self._weights, tf.constant(weights, dtype=tf.float32)))

    def placeholders(self) -> Placeholders:
        return self._placeholders

    def load_checkpoint(self, sess: tf.Session, checkpoint_path: str):
        saver = tf.train.Saver([self._weights])
        saver.restore(sess, checkpoint_path)


