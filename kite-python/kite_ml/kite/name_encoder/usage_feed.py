from typing import NamedTuple, List

from ..asserts.asserts import Assert, FieldValidator

from ..utils.segmented_data import SegmentedIndicesFeed


class Feed(NamedTuple):
    usages: SegmentedIndicesFeed
    names: List[str]
    types: List[str]

    @classmethod
    def from_json(cls, d: dict) -> 'Feed':
        v = FieldValidator(cls, d)
        return Feed(
            usages=v.get('usages', dict, build=SegmentedIndicesFeed.from_json),
            names=v.get_list('names', str),
            types=v.get_list('types', str),
        )

    def num_vars(self) -> int:
        return len(self.usages.indices)

    def assert_valid(self, batch_size: int):
        num_vars = len(self.usages.sample_ids)
        self.usages.assert_valid(batch_size)
        a = Assert.has_len(num_vars)
        a('names', self.names)
        a('types', self.types)
