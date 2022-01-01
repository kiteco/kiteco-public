from typing import Any, Dict, List, NamedTuple

import numpy as np
import tensorflow as tf

from ..graph_data.graph_feed import Edge
from ..graph_data.graph import EdgeType

from ..utils.segmented_data import SegmentedIndices, SegmentedIndicesFeed

from ..asserts.asserts import FieldValidator


class EdgePlaceholders(object):
    def __init__(self, edge_set: List[EdgeType]):
        self._edge_set = edge_set
        self._edge_keys = self._get_edge_keys()
        self.edges: Dict[str, tf.Tensor] = {}
        for key in self._edge_keys:
            # Normally we avoid having placeholders with defaults, but with the tensorflow go bindings
            # there's no way to pass in a 0-by-2 array as a feed
            self.edges[key] = tf.placeholder_with_default(
                np.empty((0, 2), dtype=np.int32), shape=[None, 2], name=key)

    def _get_edge_keys(self) -> List[str]:
        keys = []
        for edge_type in self._edge_set:
            for forward in [True, False]:
                keys.append(edge_type.edge_key(forward))
        return keys

    def dict(self) -> Dict[str, tf.Tensor]:
        d = {}
        for adj_list in self.edges.values():
            d[adj_list.name] = adj_list
        return d

    def feed_dict(self, feed: Dict[str, List[Edge]]) -> Dict[tf.Tensor, Any]:
        d = {}
        for key, adj_list in feed.items():
            d[self.edges[key]] = adj_list
        return d


class NodeFeed(NamedTuple):
    types: SegmentedIndicesFeed
    subtokens: SegmentedIndicesFeed

    @classmethod
    def from_json(cls, d: dict) -> 'NodeFeed':
        v = FieldValidator(cls, d)
        return NodeFeed(
            types=v.get('types', dict, build=SegmentedIndicesFeed.from_json),
            subtokens=v.get('subtokens', dict, build=SegmentedIndicesFeed.from_json),
        )

    def assert_valid(self, max_type_sym: int, max_subtoken_sym: int):
        batch_size = len(set(self.types.sample_ids))
        self.types.assert_valid(batch_size, max_type_sym)
        self.subtokens.assert_valid(batch_size, max_subtoken_sym)

    def num_nodes(self) -> int:
        return len(set(self.types.sample_ids))


class NodePlaceholders(object):
    def __init__(self):
        # [number of types in all nodes]
        self.types = SegmentedIndices('types')

        # [number of sub-tokens in all nodes]
        self.subtokens = SegmentedIndices('subtokens')

    def dict(self) -> Dict[str, tf.Tensor]:
        d = self.subtokens.dict()
        d.update(self.types.dict())
        return d

    def feed_dict(self, feed: NodeFeed) -> Dict[tf.Tensor, Any]:
        d = self.subtokens.feed_dict(feed.subtokens)
        d.update(self.types.feed_dict(feed.types))
        return d
