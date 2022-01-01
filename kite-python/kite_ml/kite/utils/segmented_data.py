from typing import NamedTuple, List, Dict, Any

from ..asserts.asserts import FieldValidator, assert_valid_segmented_dataset


import tensorflow as tf


class SegmentedIndicesFeed(NamedTuple):
    sample_ids: List[int]
    indices: List[int]

    @classmethod
    def from_json(cls, d: dict) -> 'SegmentedIndicesFeed':
        v = FieldValidator(cls, d)
        return SegmentedIndicesFeed(
            sample_ids=v.get_list('sample_ids', int),
            indices=v.get_list('indices', int),
        )

    def assert_valid(self, batch_size: int = -1, max_elem: int = -1):
        assert_valid_segmented_dataset(batch_size, max_elem, self.indices, self.sample_ids)

    def num_samples(self) -> int:
        if len(self.sample_ids) > 0:
            return self.sample_ids[len(self.sample_ids)-1]
        return 0


class SegmentedIndices(object):
    def __init__(self, prefix: str):
        self.sample_ids: tf.Tensor = tf.placeholder(
            dtype=tf.int32,
            shape=[None], name=prefix+'_sample_ids',
        )

        self.indices: tf.Tensor = tf.placeholder(
            dtype=tf.int32,
            shape=[None], name=prefix+'_indices',
        )

    def dict(self) -> Dict[str, tf.Tensor]:
        return {
            self.sample_ids.name: self.sample_ids,
            self.indices.name: self.indices,
        }

    def feed_dict(self, feed: SegmentedIndicesFeed) -> Dict[tf.Tensor, Any]:
        return {
            self.sample_ids: feed.sample_ids,
            self.indices: feed.indices,
        }
