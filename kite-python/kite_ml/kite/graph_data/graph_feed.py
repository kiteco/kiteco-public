from typing import Dict, List, Tuple, NamedTuple

from .graph import EdgeType, NodeID

from ..utils.segmented_data import SegmentedIndicesFeed

from ..asserts.asserts import Assert, FieldValidator

# edges are represented as (from node, to node) tuples
Edge = Tuple[NodeID, NodeID]


class GraphFeedConfig(NamedTuple):
    edge_set: List[EdgeType]

    def to_json(self) -> dict:
        return {
            'edge_set': [et.value for et in self.edge_set],
        }


def build_edges(lst: List[List[NodeID]]) -> List[Edge]:
    return [(e[0], e[1]) for e in lst]


def assert_valid_edges(edge_set: List[EdgeType], edges: Dict[str, List[Edge]], num_nodes: int):
    edge_keys = edges.keys()
    num_edge_types = len(edge_keys)
    assert num_edge_types <= 2*len(edge_set), \
        'expected at most {} edge types, got {}'.format(2*len(edge_set), num_edge_types)

    for edge_key, edges in edges.items():
        edge_type = EdgeType.from_edge_key(edge_key)
        if edge_type not in edge_set:
            # TODO: should we raise an exception here?
            continue

        for edge in edges:
            assert len(edge) == 2 and \
                   isinstance(edge[0], int) and isinstance(edge[1], int), 'invalid edge format {}'.format(edge)
            assert 0 <= edge[0] < num_nodes, \
                'invalid from node id {} for edge of type {}'.format(edge[0], edge_key)
            assert 0 <= edge[1] < num_nodes, \
                'invalid to node id {} for edge of type {}'.format(edge[1], edge_key)


class GraphFeed(NamedTuple):
    node_types: SegmentedIndicesFeed
    node_subtokens: SegmentedIndicesFeed
    edges: Dict[str, List[Edge]]

    @classmethod
    def from_json(cls, d: dict) -> 'GraphFeed':
        v = FieldValidator(cls, d)

        return GraphFeed(
            node_types=v.get('node_types', dict, build=SegmentedIndicesFeed.from_json),
            node_subtokens=v.get('node_subtokens', dict, build=SegmentedIndicesFeed.from_json),
            edges=v.get_map('edges', str, list, val_build=build_edges),
        )

    def assert_valid(self, edge_set: List[EdgeType], max_type: int, max_subtoken: int):
        Assert.unique()('GraphFeed.edge_set', edge_set)
        self.node_types.assert_valid(-1, max_type)
        self.node_subtokens.assert_valid(-1, max_subtoken)

        nodes_types = set(self.node_types.sample_ids)
        nodes_subtokens = set(self.node_subtokens.sample_ids)
        for i in nodes_types:
            assert i in nodes_subtokens, 'missing node {} in subtokens but is in types'.format(i)
        for i in nodes_subtokens:
            assert i in nodes_types, 'missing node {} in types but is in subtokens'.format(i)

        num_nodes = len(nodes_types)

        assert_valid_edges(edge_set, self.edges, num_nodes)
