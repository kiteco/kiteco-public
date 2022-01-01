from typing import Dict, Any, List

import tensorflow as tf
import time

from ..graph_encoder.encoder import GraphEncoder
from ..graph_encoder.embeddings import Embeddings, NodeEmbeddings
from ..graph_encoder.expansion_graph import Graph as ExpansionGraph
from ..graph_encoder.ggnn import GGNN

from ..infer_name.model import Model as NameModel
from ..infer_production.model import Model as ProductionModel
from ..model.model import Model as BaseModel

from ..graph_data.session import RawSample

from ..utils.aggregator import SummaryInfo, AggregateOp
from .config import Config, MetaInfo

from .feed import Feed


class _TrainPlaceholders(object):
    def __init__(self):
        with tf.name_scope('debug'):
            self.production_batch: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[], name='production_batch',
            )

            self.name_batch: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[], name='name_batch',
            )

            self.num_nodes_context_graph: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[], name='num_nodes_context_graph',
            )

            self.num_nodes_expansion_graph: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[], name='num_nodes_expansion_graph',
            )

            self.elapsed_time: tf.Tensor = tf.placeholder(
                dtype=tf.float32, shape=[], name='elapsed_time',
            )

    def feed_dict(self, feed: Feed, elapsed_time: float) -> Dict[tf.Tensor, Any]:
        return {
            self.production_batch: feed.num_production_samples,
            self.name_batch: feed.num_name_samples,
            self.num_nodes_context_graph: feed.num_nodes_context_graph,
            self.num_nodes_expansion_graph: feed.num_nodes_expansion_graph,
            self.elapsed_time: elapsed_time,
        }


class Model(BaseModel):
    """
    References:
        1) https://arxiv.org/abs/1805.08490
        2) https://arxiv.org/abs/1711.00740
    """
    def __init__(self, config: Config, meta: MetaInfo, compressed: bool):
        self._config = config
        self._meta = meta
        self._compressed = compressed

        self._start_time = time.time()

        self._train_placeholders = _TrainPlaceholders()

        self._build()

        self._build_summaries()

    def _build(self):
        with tf.variable_scope('embeddings'):
            self._embeddings = Embeddings(
                type_vocab=len(self._meta.type_subtoken_index),
                subtoken_vocab=len(self._meta.name_subtoken_index),
                compressed=self._compressed,
                reuse=False,
                config=self._config.embedding,
            )

        # NOTE: for now we use a shared GGNN object
        # for the context graph and expansion graph
        # in order to share the weights.
        # Once we do the attribute grammar graph structure
        # we can use separate GGNNs.
        with tf.variable_scope('ggnn'):
            self._ggnn = GGNN(self._config.ggnn, self._embeddings.depth(), reuse=False)

        # NOTE: we use name_scope here to ensure that
        # the underlying GGNN transformation parameters
        # are shared between the context graph and
        # the propagation graph.
        with tf.name_scope('context_graph'):
            self._context_graph = GraphEncoder(
                self._ggnn, self._embeddings, self._config.max_hops,
            )

        # NOTE: we use name_scope so that the underlying variables
        # in the expansion graph and prediction ops are shared between test and train
        with tf.name_scope('train'):
            self._build_train_prediction_op()

        with tf.name_scope('test'):
            self._build_test_prediction_op()

    def _build_train_prediction_op(self):
        with tf.name_scope('expansion_graph'):
            cg = NodeEmbeddings(
                embeddings=self._context_graph.final_node_states(),
                depth=self._embeddings.depth(),
            )

            self._train_expansion_graph = ExpansionGraph(self._ggnn, cg, self._embeddings, True)

        with tf.name_scope('infer_name'):
            self._train_infer_name = NameModel(
                self._train_expansion_graph.node_states(), self._embeddings, True, self._config.loss,
            )

        with tf.name_scope('infer_production'):
            self._train_infer_prod = ProductionModel(
                self._config.production,
                self._meta.production.vocab(),
                self._train_expansion_graph.node_states(),
                self._compressed, True,
            )

        with tf.name_scope('loss'):
            self._loss = self._train_infer_name.loss() + self._train_infer_prod.loss()

    def _build_test_prediction_op(self):
        with tf.name_scope('expansion_graph'):
            self._test_expansion_graph = ExpansionGraph(self._ggnn, None, self._embeddings, False)

        with tf.name_scope('infer_name'):
            self._test_infer_name = NameModel(
                self._test_expansion_graph.node_states(), self._embeddings, False, self._config.loss,
            )

        with tf.name_scope('infer_production'):
            self._test_infer_prod = ProductionModel(
                self._config.production,
                self._meta.production.vocab(),
                self._test_expansion_graph.node_states(),
                self._compressed, False,
            )

    def _build_summaries(self):
        infos = [
            SummaryInfo('num_production_samples', agg=AggregateOp.RUNNING_TOTAL),
            SummaryInfo('num_name_samples', agg=AggregateOp.RUNNING_TOTAL),
            SummaryInfo('num_samples_all', agg=AggregateOp.RUNNING_TOTAL),
            SummaryInfo('num_production_samples_batch_avg'),
            SummaryInfo('num_name_samples_batch_avg'),
            SummaryInfo('num_samples_all_batch_avg'),
            SummaryInfo('num_nodes_context_graph'),
            SummaryInfo('num_nodes_expansion_graph'),
            SummaryInfo('name_loss'),
            SummaryInfo('production_loss'),
            SummaryInfo('elapsed_time'),
        ]

        infos.extend(
            [info.with_name('name_{}'.format(info.name)) for info in self._train_infer_name.summary_infos()],
        )
        infos.extend(
            [info.with_name('production_{}'.format(info.name)) for info in self._train_infer_prod.summary_infos()],
        )
        self._infos = infos

        num_samples_all: tf.Tensor = self._train_placeholders.production_batch \
            + self._train_placeholders.name_batch

        summaries = {
            'num_production_samples': self._train_placeholders.production_batch,
            'num_name_samples': self._train_placeholders.name_batch,
            'num_samples_all': num_samples_all,
            'num_production_samples_batch_avg': self._train_placeholders.production_batch,
            'num_name_samples_batch_avg': self._train_placeholders.name_batch,
            'num_samples_all_batch_avg': num_samples_all,
            'num_nodes_context_graph': self._train_placeholders.num_nodes_context_graph,
            'num_nodes_expansion_graph': self._train_placeholders.num_nodes_expansion_graph,
            'elapsed_time': self._train_placeholders.elapsed_time,
            'name_loss': self._train_infer_name.loss(),
            'production_loss': self._train_infer_prod.loss(),
        }
        summaries.update(
            {'name_{}'.format(k): v for k, v in self._train_infer_name.summaries_to_fetch().items()},
        )
        summaries.update(
            {'production_{}'.format(k): v for k, v in self._train_infer_prod.summaries_to_fetch().items()},
        )
        self._summaries = summaries

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._context_graph.placeholders().dict()
        d.update(self._test_expansion_graph.placeholders_dict())
        d.update(self._test_infer_name.placeholders_dict())
        d.update(self._test_infer_prod.placeholders_dict())
        return d

    def outputs_dict(self) -> Dict[str, tf.Tensor]:
        d = {
            self._context_graph.final_node_states().name: self._context_graph.final_node_states(),
            self._test_infer_prod.pred().name: self._test_infer_prod.pred(),
            self._test_infer_name.pred().name: self._test_infer_name.pred(),
        }
        d.update(self._test_expansion_graph.outputs_dict())
        return d

    def loss(self) -> tf.Tensor:
        return self._loss

    def feed_dict(self, sample: RawSample, train: bool) -> Dict[tf.Tensor, Any]:
        elapsed_time = time.time() - self._start_time

        feed = Feed.from_raw(sample.data.expr)
        feed.assert_valid(self._config, self._meta)

        d = self._context_graph.feed_dict(feed.context_graph)
        d.update(self._train_placeholders.feed_dict(feed, elapsed_time))
        d.update(self._train_expansion_graph.feed_dict(feed.expansion_graph))
        d.update(self._train_infer_name.feed_dict(feed.infer_name, train))
        d.update(self._train_infer_prod.feed_dict(feed.infer_production, train))
        return d

    def summary_infos(self) -> List[SummaryInfo]:
        return self._infos

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries
