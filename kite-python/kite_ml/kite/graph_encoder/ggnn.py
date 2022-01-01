from typing import NamedTuple, List, Dict, Optional

import tensorflow as tf

import numpy as np

from ..graph_data.graph import EdgeType

from ..model.config import PoolingOpt

from ..asserts.asserts import Assert, assert_enum

from ..utils.segment import segment_softmax, normalize_segment_ids


class Config(NamedTuple):
    edge_set: List[EdgeType]
    message_pooling: PoolingOpt
    tie_fwd_bkwd_weights: bool = True
    use_edge_attention: bool = True
    separate_grus_per_step: bool = False

    def assert_valid(self):
        Assert.unique()('ggnn.Config.edges', self.edge_set)
        assert_enum(PoolingOpt, self.message_pooling)


class _Edge(NamedTuple):
    weights: tf.Variable
    attn: Optional[tf.Variable]


class GGNN(object):
    """
    GGNN is a Tensorflow subgraph that given an initial set of embeddings for each node,
    uses a GRU to propagate messages between the nodes in order to calculate a final propagated
    embedding for each node.
    Based on:
      - https://arxiv.org/abs/1511.05493
      - https://arxiv.org/abs/1711.00740
    """
    def __init__(self, config: Config, node_dim: int, reuse: bool = False):
        self._config = config
        self._reuse = reuse
        self._node_dim = node_dim
        with tf.variable_scope('edge_weights', reuse=reuse):
            self._edges: Dict[str, _Edge] = {}
            for edge_type in self._config.edge_set:
                if self._config.tie_fwd_bkwd_weights:
                    keys = [edge_type.value]
                else:
                    keys = [EdgeType.edge_key(edge_type, True), EdgeType.edge_key(edge_type, False)]
                for key in keys:
                    weights = tf.get_variable(
                        name='{}_weights'.format(key), shape=[self._node_dim, self._node_dim],
                        initializer=tf.glorot_uniform_initializer(),
                    )

                    edge_attn = None
                    if self._config.use_edge_attention:
                        edge_attn = tf.Variable(
                            np.ones([1], dtype=np.float32), name='{}_edge_attn_weight'.format(key),
                        )

                    self._edges[key] = _Edge(
                        weights=weights,
                        attn=edge_attn,
                    )

    def propagate(self, state: tf.Tensor, schedule: List[Dict[str, tf.Tensor]]) -> tf.Tensor:
        """
        Notation:
            N = num nodes in graph
            D = embedding depth
            E_types = num edge types
            E_edge_i = num edges for edge type 'edge' at round i
            E_i = sum_edge(E_edge_i) => total number of edges across all types at round i
        :param state: initial node states [N, D]
        :param schedule: message passing schedule, len(schedule) == number of message passing iterations, where
            schedule[i][edge][:, 0] => source nodes for message passing iteration i and edge type edge,
            schedule[i][edge][:, 1] => target nodes for message passing iteration i and edge type edge.
        :return: final node states [N, D]
        """
        state_shape: tf.Tensor = tf.shape(state, name='state_shape')
        for i, edges in enumerate(schedule):
            with tf.name_scope('propagate_{}'.format(i)):
                # iterate over the edge keys in deterministic order
                # and make sure we ALWAYS iterate over sources
                # and targets in this order
                # Note: same logic is applied in kite-go/lang/python/pythongraph/insights.go
                edge_keys = sorted(edges.keys())

                targets_list: List[tf.Tensor] = []  # targets for each edge type [E_edge_i]
                for edge_key in edge_keys:
                    adj_list = edges[edge_key]
                    targets_list.append(adj_list[:, 1])
                targets = tf.concat(targets_list, axis=0, name='targets')  # [E_i]

                # use sorted version of segment_x functions
                # since unsorted unsorted_segment_mean seems to encounter a
                # weird issue where is cannot infer the shape of segment_ids (targets)
                # SEE: SO link: /questions/51827769/tensorflow-unsorted-segment-mean-with-partially-known-segment-ids-shape
                sorting_index = tf.contrib.framework.argsort(targets)
                sorted_targets = tf.gather(targets, sorting_index, axis=0)

                # gather messages for each edge type along with attentions
                all_messages: List[tf.Tensor] = []  # each entry [E_edge_i D]
                edge_attention: List[tf.Tensor] = []  # each entry [E_edge_i]

                for edge_key in edge_keys:
                    # need to get adjacency list based on the directed edge key
                    # since the adjacency list is also directed
                    adj_list = edges[edge_key]
                    with tf.name_scope('edge_' + edge_key):
                        edge = self._edge(edge_key)
                        # [E_edge_i D]
                        source_state = tf.gather(state, adj_list[:, 0], name='source_state')

                        # all messages for edge type
                        # [E_edge_i D] x [D D] = [E_edge_i D]
                        message = tf.matmul(source_state, edge.weights, name='message')
                        all_messages.append(message)

                        if self._config.use_edge_attention:
                            # Edge attention ALA https://arxiv.org/pdf/1508.04025.pdf
                            # Intuition:
                            #  - dot prod between source, target is big => pay more attention to message
                            #  - we softmax normalize over all messages going into a particular node (below)
                            # [E_edge_i D]
                            target_state = tf.gather(state, adj_list[:, 1], name='target_state')

                            # [E_edge_i D] * [E_edge_i D] -> [E_edge_i D]
                            # reduce D axis, multiply by edge attention weight
                            # -> [E_edge_i]
                            attn = edge.attn * tf.reduce_sum(
                                source_state * target_state, axis=1, name='edge_attention',
                            )

                            edge_attention.append(attn)

                # compute incoming messages
                messages: tf.Tensor = tf.concat(all_messages, axis=0, name='messages')  # [E_i D]
                sorted_messages = tf.gather(messages, sorting_index, axis=0, name='sorted_messages')

                if self._config.message_pooling is PoolingOpt.SUM:
                    msg_pool_op = tf.segment_sum
                elif self._config.message_pooling is PoolingOpt.AVG:
                    msg_pool_op = tf.segment_mean
                elif self._config.message_pooling is PoolingOpt.MAX:
                    msg_pool_op = tf.segment_max
                else:
                    raise AssertionError('unrecognized message pooling opt {}'.format(self._config.message_pooling))

                if self._config.use_edge_attention:
                    # [E_i]
                    attentions: tf.Tensor = tf.concat(edge_attention, axis=0, name='attentions')
                    sorted_attentions = tf.gather(attentions, sorting_index, axis=0, name='sorted_attentions')

                    # softmax normalize attentions over all messages coming into a particular node
                    sorted_attentions = segment_softmax(sorted_attentions, sorted_targets,
                                                        name='softmaxed_attentions')

                    # expand to [E_i 1] to make broadcasting work properly
                    # since messages shape is [E_i D] and we want to broadcast attention weight
                    # along D axis
                    # [E_i D] * [E_i 1] => [E_i D]
                    sorted_messages = sorted_messages * tf.expand_dims(sorted_attentions, -1)

                # TODO: this is pretty gnarly, the other option is to split the edge sources and targets,
                # and then require that the targets at each round be normalized such that
                # they are contiguous in the range 0 ... num targets for the round

                sorted_targets_unique, sorted_targets_unique_idxs = tf.unique(sorted_targets,
                                                                              name='sorted_targets_unique')

                old_target_states = tf.gather(state, sorted_targets_unique, name='old_target_states')

                # normalize sorted targets so that they are in range 0 ... num targets for the round
                # before doing the segment op. We need to do this because we cannot guarantee that
                # the first target node is 0 or that last one is num_nodes, or that they are contiguous
                # this is important because the segment ops will return a tensor with max(segment_ids) rows.
                # [num messages]
                normalized_sorted_targets = normalize_segment_ids(
                    sorted_targets, sorted_targets_unique_idxs, name='sorted_targets_unique'
                )

                # [num target nodes, embedding depth]
                incoming_messages = msg_pool_op(sorted_messages, normalized_sorted_targets, name='pooled_messages')

                gru_name = 'gru'
                if self._config.separate_grus_per_step:
                    gru_name = 'gru_{}'.format(i)

                gru = tf.contrib.rnn.GRUCell(
                    num_units=self._node_dim,
                    reuse=tf.AUTO_REUSE,
                    activation=tf.nn.tanh,
                    name=gru_name,
                )

                _, new_target_states = gru.apply(incoming_messages, old_target_states)

                # scatter new states as needed
                # TODO: once we upgrade our version of TF we should use the tensor_scatter_nd_* ops here.
                # need to expand dims so tf.scatter_nd works properly
                sorted_targets_unique = tf.expand_dims(sorted_targets_unique, -1)

                # [N, D]
                update_mask = tf.scatter_nd(
                    indices=sorted_targets_unique,
                    shape=state_shape,
                    updates=tf.ones_like(new_target_states),
                    name='update_mask',
                )

                # [N, D]
                keep_state = (tf.constant(1.) - update_mask) * state

                # [N, D]
                new_target_states = tf.scatter_nd(
                    indices=sorted_targets_unique,
                    shape=state_shape,
                    updates=new_target_states,
                    name='new_state',
                )

                state = tf.identity(
                    keep_state + new_target_states,
                    name='updated_states_{}'.format(i),
                )
                # TODO: once we upgrade our version of TF we should use the tensor_scatter_nd_* ops here.
                # state = tf.tensor_scatter_nd_update(
                #     state, states_to_update, new_state, name='updated_states_{}'.format(i),
                # )
        return state

    def edge_set(self) -> List[EdgeType]:
        return [t for t in self._config.edge_set]

    def _edge(self, edge_key: str) -> _Edge:
        if self._config.tie_fwd_bkwd_weights:
            # update the edge key to no longer have a direction, since the weights
            # are shared in each direction
            edge_type = EdgeType.from_edge_key(edge_key)
            edge_key = str(edge_type.value)
        return self._edges[edge_key]
