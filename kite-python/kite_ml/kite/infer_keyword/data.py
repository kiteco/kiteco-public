from typing import Callable, Generator, List, Set, Tuple

import json
import random
import tqdm

from .config import Config
from .features import FeatureEncoder
from .raw import RawRecord


class Dataset(object):
    def __init__(self, config: Config):
        self.config = config
        self.records: List[RawRecord] = []

    def load(self, filename: str, max_count: int = -1) -> 'Dataset':
        print("loading dataset from {0}".format(filename))
        count = max_count
        with open(filename, 'r') as file:
            for line in tqdm.tqdm(file):
                line = line.strip()
                raw_dict = json.loads(line)

                if self.config.validate_input:
                    RawRecord.assert_valid(raw_dict, self.config)
                record = RawRecord(raw_dict)
                self.records.append(record)
                count -= 1
                if count == 0:
                    break

        print("{0} records read".format(len(self.records)))
        return self

    # train_test_split randomly splits the dataset into two such that the first dataset returned has approximately
    # original_size * (1 - test_size) elements and the second one has approximately original_size * test_size elements.
    def train_test_split(self, test_size: float) -> Tuple['Dataset', 'Dataset']:
        test_indices: Set[int] = set({})

        test_count = int(test_size * len(self.records))
        while len(test_indices) < test_count:
            test_indices.add(random.randrange(0, len(self.records)))

        train_records = []
        test_records = []

        for i, record in enumerate(self.records):
            if i in test_indices:
                test_records.append(record)
            else:
                train_records.append(record)

        return self._with_records(train_records), self._with_records(test_records)

    def filter(self, include_fn: Callable[[RawRecord], bool]) -> 'Dataset':
        return self._with_records([r for r in self.records if include_fn(r)])

    def _with_records(self, records: List[RawRecord]) -> 'Dataset':
        d = Dataset(config=self.config)
        d.records = records
        return d


class Batch(object):
    def __init__(self, records: List[RawRecord], feature_encoder: FeatureEncoder):
        self.features: List[List[int]] = [feature_encoder.feature_list(r) for r in records]
        self.is_keyword: List[int] = [r.is_keyword for r in records]
        self.keyword_cat: List[int] = [r.keyword_cat for r in records]


class DataFeeder(object):
    def __init__(self, dataset: Dataset, feature_encoder: FeatureEncoder, batch_size: int):
        self.dataset = dataset
        self.feature_encoder = feature_encoder
        self.batch_size = batch_size
        self.n_batches = int(len(dataset.records) / batch_size)

    def __iter__(self) -> Generator[Batch, None, None]:
        for batch_i in range(self.n_batches):
            min_i = batch_i * self.batch_size
            max_i = (batch_i + 1) * self.batch_size
            yield Batch(self.dataset.records[min_i:max_i], self.feature_encoder)
