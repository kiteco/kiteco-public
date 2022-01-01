import logging
import json

from kite.model.model import DataFeeder

from .raw_sample import RawSample


class FileFeeder(DataFeeder):
    def __init__(self, json_filename: str, count: int, start_offset: int=0):
        """
        Cycle through a file containing JSON-encoded samples

        :param json_filename: contains JSON-encoded RawSamples separated by newlines
        :param count: the number of records to read before restarting
        :param start_offset: the byte offset at which to start reading files
        """
        self._filename = json_filename
        self._count = count
        self._start_offset = start_offset
        self._file = None
        self._reset()

    def next(self) -> RawSample:
        if self._counter >= self._count:
            self._reset()
            return self.next()

        line = self._file.readline()
        if not line:
            if self._counter == 0:
                raise Exception("reached end of {} without reading lines".format(self._filename))
            self._reset()
            return self.next()

        data = json.loads(line)
        self._counter += 1
        return RawSample.from_json(data)

    def count(self) -> int:
        return self._count

    def _reset(self):
        if self._file is not None:
            self._file.close()
        self._file = open(self._filename, 'r')
        self._file.seek(self._start_offset)
        self._counter = 0

    def stop(self):
        if self._file is not None:
            self._file.close()


class FileFeederSplit(object):
    def __init__(self, json_filename: str, val_fraction: float=0.2):
        """
        Create train/val FileFeeders which cycle through disjoint partitions of a file containing JSON-encoded samples

        :param json_filename: contains JSON-encoded RawSamples separated by newlines
        :param val_fraction: fraction of samples to use in the validation set
        """
        num_train = 0
        val_offset = 0

        assert 0 < val_fraction < 1, "val_fraction out of range: {}".format(val_fraction)

        with open(json_filename, 'r') as f:
            lines = 0
            for _ in f:
                lines += 1

            num_train = int(lines * (1.0 - val_fraction))
            num_val = int(lines * val_fraction)

            assert num_train > 0, "{} has too few samples ({}) to create train set".format(json_filename, lines)
            assert num_val > 0, "{} has too few samples ({}) to create validation set".format(json_filename, lines)

            logging.info("{} has {} samples, will use {} for train and {} for validation".format(
                json_filename, lines, num_train, num_val))

            # find the offset of the first validation sample
            f.seek(0)
            line = 0
            while True:
                if line == num_train:
                    val_offset = f.tell()
                    break
                f.readline()
                line += 1

        self._train_feeder = FileFeeder(json_filename, num_train)
        self._val_feeder = FileFeeder(json_filename, num_val, start_offset=val_offset)

    def train_feeder(self) -> FileFeeder:
        return self._train_feeder

    def val_feeder(self) -> FileFeeder:
        return self._val_feeder

