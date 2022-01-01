from typing import Dict, Any, List

import tensorflow as tf

import numpy as np

from ..model.model import Model as BaseModel
from ..model.config import LossOpt

from ..graph_encoder.embeddings import NodeEmbeddings

from ..name_encoder.scope_encoder import Encoder as ScopeEncoder

from ..utils.segment import segment_softmax, segment_maxmargin_loss, segment_accuracy
from ..utils.segmented_data import SegmentedIndices
from ..utils.aggregator import SummaryInfo
from ..utils.loss import safe_cross_entropy
from ..utils.initializers import glorot_init
from ..utils.embeddings import CodeBookEmbedding

from .feed import Feed
from .config import Config


class _Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            # shape [num graphs in batch associated with a production task]
            self.prediction_nodes: tf.Tensor = tf.placeholder(
                dtype=tf.int32, shape=[None], name='prediction_nodes',
            )

            # ids (rows) for each possible production expansion across all production
            # prediction tasks in the batch.
            # shape [num_decoder_targets_batch]
            # e.g let s range over samples in the batch and let num_expansions(s) be
            # the number of possible expansions in the output class for sample s
            # then num_decoder_targets_batch = sum_s(num_expansions(s))
            self.decoder_targets = SegmentedIndices('decoder_targets')

            # [num context tokens in all tasks in batch]
            self.context_tokens = SegmentedIndices('context_tokens')

    def dict(self) -> Dict[str, tf.Tensor]:
        d = {
            self.prediction_nodes.name: self.prediction_nodes,
        }
        d.update(self.decoder_targets.dict())
        d.update(self.context_tokens.dict())
        return d

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        fd = {
            self.prediction_nodes: feed.prediction_nodes,
        }
        fd.update(self.decoder_targets.feed_dict(feed.decoder_targets))
        fd.update(self.context_tokens.feed_dict(feed.context_tokens))
        return fd


class _TrainPlaceholders(object):
    def __init__(self):
        with tf.name_scope('labels'):
            # shape [num graphs in batch associated with a production task]
            self.labels: tf.Tensor = tf.placeholder(
                dtype=tf.int32, shape=[None], name='labels',
            )

            # shape [num corrupted labels]
            self.corrupted = SegmentedIndices('corrupted')

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        fd = {
            self.labels: feed.labels,
        }
        fd.update(self.corrupted.feed_dict(feed.corrupted))
        return fd


class Model(BaseModel):
    def __init__(self, config: Config, vocab: int, nodes: NodeEmbeddings, compressed: bool, train: bool):
        self._config = config
        self._vocab = vocab
        self._nodes = nodes
        self._compressed = compressed
        self._reuse = not train
        self._train = train

        with tf.name_scope('scope_encoder'):
            self._scope = ScopeEncoder(self._nodes)
        self._placeholders = _Placeholders()

        if self._train:
            self._train_placeholders = _TrainPlaceholders()
        self._build()

    def _token_state(self) -> tf.Tensor:
        # [num infer production tasks in batch, graph embedding depth]
        site_state: tf.Tensor = tf.gather(
            self._nodes.embeddings, self._placeholders.prediction_nodes, name='site_state',
        )

        # distribute prediction site state up to number of context tokens
        # [num tokens in batch, graph embedding depth]
        site_state = tf.gather(
            site_state, self._placeholders.context_tokens.sample_ids, name='site_state_tokens',
        )

        # [num tokens in batch, graph embedding depth]
        token_state = tf.gather(
            self._nodes.embeddings, self._placeholders.context_tokens.indices, name='token_state_init',
        )

        # [num tokens in batch]
        token_attn = tf.reduce_sum(token_state * site_state, axis=1, name='token_attn_logits')

        # [num tokens in batch]
        token_attn = segment_softmax(token_attn, self._placeholders.context_tokens.sample_ids, name='token_attn')

        # [num tokens in batch, 1] * [num tokens in batch, graph embedding depth]
        # =>
        # [num samples in batch, graph embedding depth]
        token_state = tf.segment_sum(
            tf.expand_dims(token_attn, -1) * token_state, self._placeholders.context_tokens.sample_ids,
            name='token_state_final',
        )

        # [num decoder targets in all tasks in batch, graph embedding depth]
        return tf.gather(
            token_state, self._placeholders.decoder_targets.sample_ids, name='token_state_final_expanded',
        )

    def _site_state(self) -> tf.Tensor:
        # [num infer production tasks in batch, graph embedding depth]
        site_state: tf.Tensor = tf.gather(
            self._nodes.embeddings, self._placeholders.prediction_nodes, name='site_state',
        )

        # [num infer production tasks in batch, graph embedding depth]
        # =>
        # [num decoder targets in all infer production tasks in batch, graph embedding depth]
        return tf.gather(
            site_state, self._placeholders.decoder_targets.sample_ids, name='site_state_expanded',
        )

    def _build_prediction_op(self):
        decoder_depth = 3 * self._nodes.depth
        if self._config.decouple_decoder_dim:
            decoder_depth = self._config.depth

        context_depth = 3 * self._nodes.depth
        if not self._config.concat_context:
            context_depth = self._nodes.depth
            if not self._config.decouple_decoder_dim:
                decoder_depth = context_depth

        with tf.variable_scope('embed_decoder_targets', reuse=self._reuse):
            decoder_embeddings = CodeBookEmbedding(
                glorot_init(self._vocab, decoder_depth),
                name='production_decoder_embeddings',
                compressed=self._compressed,
                reuse=self._reuse,
                config=self._config.codebook,
            )

            # [num decoder targets in all infer production tasks in batch, decoder depth]
            targets_embedded: tf.Tensor = decoder_embeddings.lookup(
                self._placeholders.decoder_targets.indices,
                name='targets_embedded',
            )

        if self._config.decouple_decoder_dim:
            with tf.variable_scope('decoder_to_context', reuse=self._reuse):
                # [decoder depth, graph context depth]
                decoder_to_context: tf.Tensor = tf.get_variable(
                    name='decoder_to_context', shape=[decoder_depth, context_depth],
                    initializer=tf.glorot_uniform_initializer(), dtype=tf.float32,
                )
                # [num decoder targets, decoder depth] x [decoder depth, context depth]
                # =>
                # [num decoder targets, context depth]
                targets_embedded = tf.matmul(targets_embedded, decoder_to_context,
                                             name='targets_embedded_in_context_space')

        with tf.name_scope('site_state'):
            # [num decoder targets in batch, graph embedding depth]
            site_state = self._site_state()

        with tf.name_scope('token_state'):
            # [num decoder targets in batch, graph embedding depth]
            token_state = self._token_state()

        with tf.variable_scope('build_context', reuse=self._reuse):
            # [num prod tasks, graph embedding depth]
            # =>
            # [num decoder targets all tasks, graph embedding depth]
            scope_state = tf.gather(
                self._scope.scope_state(), self._placeholders.decoder_targets.sample_ids, name='scope_state_expanded',
            )

            if self._config.concat_context:
                # [num decoder targets in batch, 3 * graph embedding depth = context depth]
                context = tf.concat([site_state, scope_state, token_state], axis=1, name='context')
            else:
                # [vocab, 3]
                all_context_weights: tf.Variable = tf.get_variable(
                    name='all_scope_weights',
                    initializer=tf.ones_initializer(dtype=np.float32)(shape=(self._vocab, 3)),
                )

                # [num decoder targets in batch, 3]
                context_weights: tf.Tensor = tf.gather(
                    all_context_weights, self._placeholders.decoder_targets.indices,
                    name='scope_weight',
                )

                # [num decoder targets in batch, 1]
                scope_weight = tf.expand_dims(context_weights[:, 0], -1, name='scope_weight')
                site_weight = tf.expand_dims(context_weights[:, 1], -1, name='site_weight')
                token_weight = tf.expand_dims(context_weights[:, 2], -1, name='token_weight')

                # [num decoder targets in batch, context depth]
                context = tf.identity(
                    scope_weight * scope_state + site_weight * site_state + token_weight * token_state,
                    name='context',
                )

        with tf.name_scope('prediction'):
            # inner product context and target decoder embedding
            # [num decoder targets in all infer production tasks in batch]
            self._logits = tf.reduce_sum(targets_embedded * context, axis=1, name='logits')

            self._pred = segment_softmax(
                self._logits, self._placeholders.decoder_targets.sample_ids, name='pred',
            )

    def _build_loss_op(self) -> tf.Tensor:
        with tf.name_scope('loss'):
            if self._config.loss is LossOpt.CROSS_ENTROPY:
                return safe_cross_entropy(self._pred, self._train_placeholders.labels, 'loss')

            return segment_maxmargin_loss(
                self._logits, self._train_placeholders.labels,
                self._train_placeholders.corrupted.sample_ids,
                self._train_placeholders.corrupted.indices, name='loss')

    def _build_accuracy_op(self, topk=5):
        self._acc = segment_accuracy(
            self._pred, self._train_placeholders.labels, self._placeholders.decoder_targets.sample_ids, topk=1,
        )

        self._acck = segment_accuracy(
            self._pred, self._train_placeholders.labels, self._placeholders.decoder_targets.sample_ids, topk=topk,
        )

    def _build_scalars(self) -> Dict[str, tf.Tensor]:
        with tf.name_scope('accuracy'):
            self._build_accuracy_op(3)
            return{
                'accuracy': self._acc,
                'accuracy_atk': self._acck,
            }

    def _build(self):
        self._build_prediction_op()
        if self._train:
            self._loss = self._build_loss_op()
            self._scalars = self._build_scalars()
            self._summaries = {k: v for k, v in self._scalars.items()}

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._scope.placeholders_dict()
        d.update(self._placeholders.dict())
        return d

    def pred(self) -> tf.Tensor:
        return self._pred

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        feed_dict = self._scope.feed_dict(feed.scope_encoder)
        if self._train:
            feed_dict.update(self._train_placeholders.feed_dict(feed))
        feed_dict.update(self._placeholders.feed_dict(feed))
        return feed_dict

    def loss(self) -> tf.Tensor:
        return self._loss

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries
