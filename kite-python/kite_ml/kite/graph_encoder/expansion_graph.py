from typing import List, Dict, NamedTuple, Any, Optional

import tensorflow as tf

from ..graph_data.graph_feed import EdgeType, Edge, build_edges, assert_valid_edges

from .ggnn import GGNN
from .graph import EdgePlaceholders, NodePlaceholders, NodeFeed
from .embeddings import NodeEmbeddings, Embeddings

from ..asserts.asserts import FieldValidator


class TrainFeed(NamedTuple):
    edges: Dict[str, List[Edge]]
    context_to_expansion: List[int]
    lookup_nodes: NodeFeed
    lookup_to_expansion: List[int]
    context_graph_nodes: List[int]

    @classmethod
    def from_json(cls, d: dict) -> 'TrainFeed':
        v = FieldValidator(cls, d)
        return TrainFeed(
            edges=v.get_map('edges', str, list, val_build=build_edges),
            context_to_expansion=v.get_list('context_to_expansion', int),
            lookup_nodes=v.get('lookup_nodes', dict, build=NodeFeed.from_json),
            lookup_to_expansion=v.get_list('lookup_to_expansion', int),
            context_graph_nodes=v.get_list('context_graph_nodes', int),
        )

    def assert_valid(self, edge_set: List[EdgeType], max_type: int, max_subtoken: int):
        self.lookup_nodes.assert_valid(max_type, max_subtoken)
        num_lookup_nodes = self.lookup_nodes.num_nodes()
        assert len(self.lookup_to_expansion) == num_lookup_nodes,\
            'len(lookup_to_expansion) {} != {} num lookup nodes'.format(
                len(self.lookup_to_expansion), num_lookup_nodes)

        num_context_nodes = len(self.context_graph_nodes)
        assert num_context_nodes == len(self.context_to_expansion), \
            'context graph nodes {} != context to expansion {}'.format(
                num_context_nodes, len(self.context_to_expansion))

        num_nodes = num_context_nodes + num_lookup_nodes
        assert_valid_edges(edge_set, self.edges, num_nodes)

        for egid in self.lookup_to_expansion:
            assert 0 <= egid < num_nodes, \
                'lookup to expansion out of range {} not in [0,...,{}]'.format(egid, num_nodes-1)
        for egid in self.context_to_expansion:
            assert 0 <= egid < num_nodes, \
                'context to expansion out of range {} not in [0,...,{}]'.format(egid, num_nodes-1)

    def num_nodes(self) -> int:
        return len(self.lookup_to_expansion) + len(self.context_to_expansion)


class _Placeholders(object):
    def __init__(self, edge_set: List[EdgeType]):
        with tf.name_scope('placeholders'):
            self.edges = EdgePlaceholders(edge_set)

            # maps context graph nodes to their ID in the expansion graph
            # [N_C]
            self.context_to_expansion: tf.Tensor = tf.placeholder(
                dtype=tf.int32, shape=[None],
                name='context_to_expansion',
            )

            # the types and tokens for (some of) the nodes in the expansion graph
            # [N_L]
            self.lookup_nodes = NodePlaceholders()

            # maps lookup nodes to their ID in the expansion graph
            # [N_L]
            self.lookup_to_expansion: tf.Tensor = tf.placeholder(
                dtype=tf.int32, shape=[None],
                name='lookup_to_expansion',
            )

    def dict(self) -> Dict[str, tf.Tensor]:
        d = {
            self.context_to_expansion.name: self.context_to_expansion,
            self.lookup_to_expansion.name: self.lookup_to_expansion,
        }
        d.update(self.edges.dict())
        d.update(self.lookup_nodes.dict())
        return d

    def feed_dict(self, feed: TrainFeed) -> Dict[tf.Tensor, Any]:
        d = {
            self.context_to_expansion: feed.context_to_expansion,
            self.lookup_to_expansion: feed.lookup_to_expansion,
        }
        d.update(self.edges.feed_dict(feed.edges))
        d.update(self.lookup_nodes.feed_dict(feed.lookup_nodes))
        return d


class Graph(object):
    def __init__(self, ggnn: GGNN, context_graph: Optional[NodeEmbeddings], embeddings: Embeddings, train: bool):
        """
        Construct an expansion graph for prediction.
        :param ggnn: used for propagation
        :param context_graph: represents node embeddings to pull from the context graph, can be None when
        train is False, will be ignored when train is false
        :param embeddings: type and sub-token embeddings.
        :param train: True if we are in training mode
        """
        self._ggnn = ggnn
        self._embeddings = embeddings
        self._train = train
        self._placeholders = _Placeholders(ggnn.edge_set())
        if self._train:
            self._context_graph = context_graph
            with tf.name_scope('train_placeholders'):
                # the ids for the nodes in the context graph
                # that are fed into the expansion graph
                # [N_C]
                self._train_context_graph_node_ids: tf.Tensor = tf.placeholder(
                    dtype=tf.int32, shape=[None],
                    name='context_graph_node_ids',
                )
        else:
            self._context_graph = None
            with tf.name_scope('test_placeholders'):
                # the actual embeddings for the context nodes to place in the expansion graph
                # [N_C, node embedding depth]
                self._test_context_node_embeddings: tf.Tensor = tf.placeholder(
                    dtype=tf.float32, shape=[None, embeddings.depth()],
                    name='context_node_embeddings',
                )

        self._build()

    def _build(self):
        with tf.name_scope('context_nodes'):
            if self._train:
                context_nodes = tf.gather(
                    self._context_graph.embeddings,
                    self._train_context_graph_node_ids,
                    name='context_nodes',
                )
            else:
                context_nodes = self._test_context_node_embeddings

        with tf.name_scope('lookup_nodes'):
            lookup_nodes = self._embeddings.embed(
                self._placeholders.lookup_nodes.types,
                self._placeholders.lookup_nodes.subtokens,
            )

        with tf.name_scope('node_state_shape'):
            shape = tf.stack([
                tf.shape(context_nodes)[0] + tf.shape(lookup_nodes)[0],
                self._embeddings.depth(),
            ], name='shape')

        with tf.name_scope('build_state'):
            # TODO: once we upgrade our version of TF we should use the tensor_scatter_nd_* ops here.

            # NOTE: we add a 1 to the placeholder's last dimension
            # to make it [num lookup nodes, 1] to make tf.scatter_nd work properly

            # scatter lookup nodes into the full expansion graph node state matrix
            # [num nodes expansion graph, embedding depth]
            lookup_nodes = tf.scatter_nd(
                indices=self._placeholders.lookup_to_expansion[:, tf.newaxis],
                updates=lookup_nodes, shape=shape,
                name='lookup_nodes_eg',
            )

            # scatter context nodes up to the full expansion graph node state matrix
            # [num nodes expansion graph, embedding depth]
            context_nodes = tf.scatter_nd(
                indices=self._placeholders.context_to_expansion[:, tf.newaxis],
                updates=context_nodes, shape=shape,
                name='context_nodes_eg',
            )

            # the node state matrices above are "complementary" in the sense
            # that when a row is non zero in one it is zero in the other
            # so we can construct the full node state matrix by element-wise addition
            init_node_states = tf.identity(
                lookup_nodes + context_nodes,
                name='init_node_states',
            )

        with tf.name_scope('graph'):
            self._node_states = self._ggnn.propagate(
                init_node_states,
                [self._placeholders.edges.edges],
            )

            self._node_states: tf.Tensor = tf.identity(self._node_states, name='node_states')

    def node_states(self) -> NodeEmbeddings:
        return NodeEmbeddings(
            embeddings=self._node_states,
            depth=self._embeddings.depth(),
        )

    def feed_dict(self, feed: TrainFeed) -> Dict[tf.Tensor, Any]:
        d = self._placeholders.feed_dict(feed)
        d[self._train_context_graph_node_ids] = feed.context_graph_nodes
        return d

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._placeholders.dict()
        d[self._test_context_node_embeddings.name] = self._test_context_node_embeddings
        return d

    def outputs_dict(self) -> Dict[str, tf.Tensor]:
        return {
            self._node_states.name: self._node_states,
        }
