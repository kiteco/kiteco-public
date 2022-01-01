from typing import NamedTuple, List

from ..graph_data.graph import VariableID

from ..utils.segmented_data import SegmentedIndicesFeed

from ..name_encoder.usage_feed import Feed as NameEncoderFeed

from ..asserts.asserts import Assert, FieldValidator


class Feed(NamedTuple):
    prediction_nodes: List[int]
    corrupted: SegmentedIndicesFeed
    labels: List[VariableID]
    types: SegmentedIndicesFeed
    subtokens: SegmentedIndicesFeed
    names: NameEncoderFeed

    @classmethod
    def from_json(cls, d: dict) -> 'Feed':
        v = FieldValidator(cls, d)
        return Feed(
            prediction_nodes=v.get_list('prediction_nodes', int),
            corrupted=v.get('corrupted', dict, build=SegmentedIndicesFeed.from_json),
            labels=v.get_list('labels', int),
            types=v.get('types', dict, build=SegmentedIndicesFeed.from_json),
            subtokens=v.get('subtokens', dict, build=SegmentedIndicesFeed.from_json),
            names=v.get('names', dict, build=NameEncoderFeed.from_json),
        )

    def assert_valid(self):
        batch_size = self.batch_size()

        assert_batch_size = Assert.has_len(batch_size)
        assert_batch_size('labels', self.labels)
        assert_batch_size('prediction_nodes', self.prediction_nodes)

        self.corrupted.assert_valid(batch_size, self.names.num_vars())
        self.types.assert_valid(batch_size)
        self.subtokens.assert_valid(batch_size)
        self.names.assert_valid(batch_size)

    def batch_size(self) -> int:
        return len(self.labels)
