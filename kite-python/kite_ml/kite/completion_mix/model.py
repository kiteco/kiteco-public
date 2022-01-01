from typing import Any, Dict, List

from .config import Config
from .feed import Feed
from .raw_sample import RawSample

from ..model.model import Model as BaseModel
from ..utils.aggregator import SummaryInfo
from ..utils.initializers import glorot_init
from ..utils.segment import segment_softmax

import tensorflow as tf


class Placeholders(object):
    def __init__(self):
        context_feature_depth = Feed.context_feature_depth()
        comp_feature_depth = Feed.comp_feature_depth()

        with tf.name_scope('placeholders'):
            # size: [batch size, context feature depth]
            self.contextual_features: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[None, context_feature_depth], name='contextual_features')

            # size: [total number of completions in batch, completion feature depth]
            self.completion_features: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[None, comp_feature_depth], name='completion_features')

            # size: [total number of batch completions]

            self.sample_ids: tf.Tensor = tf.placeholder(dtype=tf.int32, shape=[None], name='sample_ids')

            # size: [batch size]
            self.labels: tf.Tensor = tf.placeholder(dtype=tf.int32, shape=[None], name='labels')

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.contextual_features: feed.context_features,
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


class Model(BaseModel):
    def __init__(self, config: Config):
        self._config = config
        self._placeholders = Placeholders()
        self._build()

    def _build(self):
        # size: [total completion count, contextual feature depth]
        expanded_contextual = tf.gather(
            self._placeholders.contextual_features, self._placeholders.sample_ids)

        # size: [total completion count, total feature depth]
        # where total_feature_depth = contextual_feature_depth + completion_feature_depth = 62+9 = 71
        concatenated_features = tf.concat([expanded_contextual, self._placeholders.completion_features], axis=1)

        total_depth = Feed.context_feature_depth() + Feed.comp_feature_depth()
        self._weights = tf.Variable(glorot_init(total_depth, 1), dtype=tf.float32, name='weights')

        # size: [total completion count] <- [total completion count, 1]
        self._logits = tf.squeeze(tf.matmul(concatenated_features, self._weights))

        self._pred: tf.Tensor = self._build_pred()
        self._loss: tf.Tensor = self._build_loss()

    def _build_pred(self) -> tf.Tensor:
        with tf.name_scope('pred'):
            # size: [total completion count]
            return segment_softmax(self._logits, self._placeholders.sample_ids, name='pred')

    def _build_loss(self) -> tf.Tensor:
        with tf.name_scope('loss'):
            # size: total completion count x 1
            # note we add a small constant to prevent -inf logs with 0 softmax
            # TODO: revisit, perhaps normalization?
            logs: tf.Tensor = -tf.log(tf.gather(self._pred + 1e-6, self._placeholders.labels,
                                                validate_indices=True, axis=0), name='logs')
            # size: scalar
            return tf.reduce_mean(logs, name='loss')

    def pred(self) -> tf.Tensor:
        return self._pred

    def loss(self) -> tf.Tensor:
        return self._loss

    def placeholders(self) -> Placeholders:
        return self._placeholders

    def feed_dict(self, batch_samples: List[RawSample], train: bool) -> Dict[tf.Tensor, Any]:
        return self._placeholders.feed_dict(Feed.from_samples(batch_samples))

    def summary_infos(self) -> List[SummaryInfo]:
        return []

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return {}

    def load_checkpoint(self, sess: tf.Session, checkpoint_path: str):
        saver = tf.train.Saver([self._weights])
        saver.restore(sess, checkpoint_path)


