from typing import Dict, Any, List

import tensorflow as tf

from ..graph_data.graph_feed import GraphFeed, EdgeType

from .embeddings import Embeddings
from .ggnn import GGNN
from .graph import NodeFeed, NodePlaceholders, EdgePlaceholders


class GraphPlaceholders(object):
    def __init__(self, edge_set: List[EdgeType]):
        with tf.name_scope('edges'):
            self.edges = EdgePlaceholders(edge_set)

        with tf.name_scope('nodes'):
            self.nodes = NodePlaceholders()

    def dict(self) -> Dict[str, tf.Tensor]:
        d = self.nodes.dict()
        d.update(self.edges.dict())
        return d

    def feed_dict(self, feed: GraphFeed) -> Dict[tf.Tensor, Any]:
        nf = NodeFeed(
            types=feed.node_types,
            subtokens=feed.node_subtokens,
        )
        d = self.nodes.feed_dict(nf)
        d.update(self.edges.feed_dict(feed.edges))
        return d


class GraphEncoder(object):
    """
    GraphEncoder is a Tensorflow subgraph that, given a graph feed which defines a directed multigraph,
    calculates initial embeddings for each node based on their types and literals (initial_node_states()),
    and uses a GRU to propagate messages between the nodes in order to calculate a final propagated
    embedding for each node (final_node_states()).
    Based on:
      - https://arxiv.org/abs/1511.05493
      - https://arxiv.org/abs/1711.00740
    """
    def __init__(self, ggnn: GGNN, embeddings: Embeddings, max_hops: int):
        self._max_hops = max_hops
        self._ggnn = ggnn
        self._embeddings = embeddings

        with tf.name_scope('placeholders'):
            self._placeholders = GraphPlaceholders(self._ggnn.edge_set())

        self._build()

    def feed_dict(self, feed: GraphFeed) -> Dict[tf.Tensor, Any]:
        return self._placeholders.feed_dict(feed)

    def initial_node_states(self) -> tf.Tensor:
        """
        :return: initial node embeddings, shape [number of nodes, self.embedding_depth()]
        """
        return self._node_embedded

    def final_node_states(self) -> tf.Tensor:
        """
        :return: propagated node embeddings, shape [number of nodes, self.embedding_depth()]
        """
        return self._node_state

    def placeholders(self) -> GraphPlaceholders:
        """
        :return: the placeholders for the graph
        """
        return self._placeholders

    def _build(self):
        self._add_node_embeddings()
        self._add_node_states()

    def _add_node_embeddings(self):
        with tf.name_scope('embed_nodes'):
            # [num nodes, total embed depth]
            self._node_embedded = tf.identity(
                self._embeddings.embed(self._placeholders.nodes.types, self._placeholders.nodes.subtokens),
                name='node_embedded',
            )

    def _add_node_states(self):
        if self._max_hops == 0:
            self._node_state = self._node_embedded
            return

        with tf.name_scope('propagate'):
            schedule = []
            for _ in range(self._max_hops):
                schedule.append(self._placeholders.edges.edges)
            # [num nodes, total embed depth]
            final_state = self._ggnn.propagate(self._node_embedded, schedule)
        
        self._node_state = tf.identity(final_state, name='final_node_states')
