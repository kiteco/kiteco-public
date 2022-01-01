from typing import NamedTuple, List

from ..asserts.asserts import FieldValidator

from ..utils.segmented_data import SegmentedIndicesFeed


class Feed(NamedTuple):
    prediction_nodes: List[int]
    labels: List[int]
    decoder_targets: SegmentedIndicesFeed
    corrupted: SegmentedIndicesFeed
    scope_encoder: SegmentedIndicesFeed
    context_tokens: SegmentedIndicesFeed

    @classmethod
    def from_json(cls, d: dict) -> 'Feed':
        v = FieldValidator(cls, d)

        return Feed(
            prediction_nodes=v.get_list('prediction_nodes', int),
            labels=v.get_list('labels', int),
            decoder_targets=v.get('decoder_targets', dict, build=SegmentedIndicesFeed.from_json),
            corrupted=v.get('corrupted', dict, build=SegmentedIndicesFeed.from_json),
            scope_encoder=v.get('scope_encoder', dict, build=SegmentedIndicesFeed.from_json),
            context_tokens=v.get('context_tokens', dict, build=SegmentedIndicesFeed.from_json),
        )

    def assert_valid(self):
        assert len(self.prediction_nodes) == len(self.labels), \
            'num pred nodes {} != num labels {}'.format(len(self.prediction_nodes), len(self.labels))

        self.decoder_targets.assert_valid(self.batch_size())

        self.corrupted.assert_valid(self.batch_size())

        self.scope_encoder.assert_valid(self.batch_size())

        self.context_tokens.assert_valid(self.batch_size())

    def batch_size(self) -> int:
        return len(self.labels)
