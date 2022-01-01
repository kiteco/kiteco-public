from typing import Dict, Any, Tuple, List

import tensorflow as tf

from ..model.model import Model as BaseModel
from ..model.config import LossOpt

from ..graph_encoder.embeddings import NodeEmbeddings, Embeddings

from ..name_encoder.usage_encoder import Encoder as NameEncoder

from ..utils.segment import segment_softmax, segment_maxmargin_loss, segment_topk, segment_accuracy
from ..utils.aggregator import SummaryInfo
from ..utils.reduce import safe_reduce_mean
from ..utils.loss import safe_cross_entropy
from ..utils.segmented_data import SegmentedIndices

from .feed import Feed


class _Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            # shape [num graphs in batch]
            self.prediction_nodes: tf.Tensor = tf.placeholder(
                dtype=tf.int32, shape=[None], name='prediction_nodes',
            )

            # shape [num types in all decoders in batch]
            self.types = SegmentedIndices('types')

            # shape [num tokens in all decoders in batch]
            self.subtokens = SegmentedIndices('subtokens')

    def dict(self) -> Dict[str, tf.Tensor]:
        d = self.types.dict()
        d.update(self.subtokens.dict())
        d[self.prediction_nodes.name] = self.prediction_nodes
        return d

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        d = self.types.feed_dict(feed.types)
        d.update(self.subtokens.feed_dict(feed.subtokens))
        d[self.prediction_nodes] = feed.prediction_nodes
        return d


class _TrainPlaceholders(object):
    def __init__(self):
        with tf.name_scope('labels'):
            # shape [num graphs in batch]
            # labels for the true variable for each graph in the batch
            self.labels: tf.Tensor = tf.placeholder(
               dtype=tf.int32, shape=[None], name='labels',
            )

            # shape [num corrupted labels in batch]
            self.corrupted = SegmentedIndices('corrupted')

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        d = {
            self.labels: feed.labels,
        }
        d.update(self.corrupted.feed_dict(feed.corrupted))
        return d


class Model(BaseModel):
    def __init__(self, nodes: NodeEmbeddings, embeddings: Embeddings, train: bool, loss: LossOpt = LossOpt.MAX_MARGIN):
        self._nodes = nodes
        self._embeddings = embeddings
        self._reuse = not train
        self._train = train

        with tf.name_scope('name_encoder'):
            self._name = NameEncoder(nodes)
        self._placeholders = _Placeholders()
        if self._train:
            self._train_placeholders = _TrainPlaceholders()
        self._loss_opt = loss
        self._build()

    def _build_prediction_op(self):
        with tf.variable_scope('embed_decoders', reuse=self._reuse):
            # [num samples in batch, graph embedding depth]
            decoders = tf.identity(
                self._embeddings.embed(self._placeholders.types, self._placeholders.subtokens),
                name='decoders',
            )

            # [graph embedding depth, 2 * graph embedding depth]
            decoder_to_context = tf.get_variable(
                name='decoder_to_context', shape=[self._embeddings.depth(), 2 * self._embeddings.depth()],
                initializer=tf.glorot_uniform_initializer(),
            )

            # [num samples in batch, graph depth] x [graph depth, 2 * graph depth]
            # =>
            # [num samples in batch, 2 * graph depth]
            decoders = tf.matmul(decoders, decoder_to_context, name='decoders_in_context')

            # distribute decoders up to
            # [num variables in batch, 2 * graph embedding depth]
            decoders = tf.gather(decoders, self._name.sample_ids(), name='decoders_expanded')

        with tf.name_scope('embed_context'):
            # [num samples in batch, graph embedding depth]
            site_state: tf.Tensor = tf.gather(
                self._nodes.embeddings, self._placeholders.prediction_nodes, name='site_state',
            )

            # distribute site state up to
            # [num variables in batch, graph embedding depth]
            site_state = tf.gather(site_state, self._name.sample_ids(), name='site_state_expanded')

            # [num variables in batch, 2 * graph embedding depth]
            context = tf.concat(
                [self._name.usage_node_states(), site_state], axis=1, name='context',
            )

        with tf.name_scope('prediction'):
            # inner product context and decoder embeddings
            self._logits = tf.reduce_sum(context * decoders, axis=1, name='logits')

            self._pred = segment_softmax(self._logits, self._name.sample_ids(), name='pred')

    def _build_loss_op(self) -> tf.Tensor:
        with tf.name_scope('loss'):
            if self._loss_opt is LossOpt.CROSS_ENTROPY:
                return safe_cross_entropy(self._pred, self._train_placeholders.labels, 'loss')

            return segment_maxmargin_loss(
                self._logits, self._train_placeholders.labels,
                self._train_placeholders.corrupted.sample_ids,
                self._train_placeholders.corrupted.indices, name='loss')

    def _predict(self, topk=5) -> Tuple[tf.Tensor, tf.Tensor, tf.Tensor]:
        """
        compute topk predictions for each segment
        :param topk:
        :return: (probs, idxs, segment_ids): all returned tensors are rank 1,
        and probs.shape == idxs.shape == segment_ids.shape
        """
        return segment_topk(self._pred, self._name.sample_ids(), topk, '_predict')

    def _build_accuracy_op(self, topk=5):
        #
        # accuracy and accuracy at k
        #
        self._acc = segment_accuracy(self._pred, self._train_placeholders.labels, self._name.sample_ids(), 1)
        self._acck = segment_accuracy(self._pred, self._train_placeholders.labels, self._name.sample_ids(), topk)

        #
        # type accuracy and type accuracy at k
        #

        # [num graphs in batch]
        _, idxs, _ = self._predict(1)

        # [num graphs in batch]
        true_types = self._name.types(self._train_placeholders.labels, name='true_types_1')

        # [num graphs in batch]
        top_types = self._name.types(idxs, name='top_types_1')

        self._type_acc: tf.Tensor = safe_reduce_mean(
            tf.cast(tf.equal(true_types, top_types), tf.float32), 0., name='type_accuracy'
        )

        _, idxs, sample_ids = self._predict(topk)

        top_types = self._name.types(idxs, 'top_types_k')

        true_types = self._name.types(self._train_placeholders.labels, name='true_types_k')

        # expand true types to match sample ids
        true_types = tf.gather(true_types, sample_ids, name='true_types_k_expanded')

        # need to be careful here since the true type could match multiple predictions
        # so segment sum then check for greater than 0, use int32 to avoid weird numerical artifacts
        types_equal = tf.cast(tf.equal(true_types, top_types), tf.int32, name='types_equal')
        # [batch size]
        types_summed: tf.Tensor = tf.segment_sum(types_equal, segment_ids=sample_ids, name='types_summed')
        any_matches: tf.Tensor = tf.cast(types_summed > tf.constant(0), tf.float32, name='any_matches')
        # reduce mean across batch
        self._type_acck: tf.Tensor = safe_reduce_mean(any_matches, 0., name='type_accuracy_atk')

    def _build_scalars(self) -> Dict[str, tf.Tensor]:
        with tf.name_scope('accuracy'):
            self._build_accuracy_op(3)

        with tf.name_scope('scalars'):
            scalars = dict()
            scalars['accuracy'] = self._acc
            scalars['accuracy_atk'] = self._acck
            scalars['type_accuracy'] = self._type_acc
            scalars['type_accuracy_atk'] = self._type_acck

            return scalars

    def _build(self):
        self._build_prediction_op()
        if self._train:
            self._loss = self._build_loss_op()
            self._scalars = self._build_scalars()
            self._summaries = {k: v for k, v in self._scalars.items()}

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._name.placeholders().dict()
        d.update(self._placeholders.dict())
        return d

    def pred(self) -> tf.Tensor:
        return self._pred

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        feed_dict = self._placeholders.feed_dict(feed)
        if self._train:
            feed_dict.update(self._train_placeholders.feed_dict(feed))
        feed_dict.update(self._name.feed_dict(feed.names))
        return feed_dict

    def loss(self) -> tf.Tensor:
        return self._loss

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries
