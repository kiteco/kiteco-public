from typing import List, NamedTuple

import logging
import os
import json
import gzip
import random
import time

from kite.model.model import DataFeeder


class Feed(NamedTuple):
    context: List[List[int]]
    langs: List[int]


class FileDataFeeder(DataFeeder):
    def __init__(self, in_dir: str, ext='.json.gz', batch_size=20, shard=0, num_shards=1):
        """
        Read files matching the specified extension from the specified directory and feed
        the samples in those files. If all files have been read, block until new files appear.

        :param in_dir: directory containing the samples
        :param ext: the extension of the files to read
        """
        self._in_dir = in_dir
        self._ext = ext
        self._already_read = set()
        self._batch_size = batch_size
        self._shard = shard
        self._num_shards = num_shards

        self._filename = None
        self._file = None
        self._count = 0
        self._reset()

    def next(self) -> Feed:
        contexts = list()
        langs = list()
        while len(contexts) < self._batch_size:
            line = self._file.readline()
            while len(line) == 0:
                self._reset()
                line = self._file.readline()

            try:
                sample = json.loads(line)
                contexts.append(sample['context'])
                langs.append(sample['lang'])
            except Exception as e:
                print("problem reading line:", line, e)
                self._reset()

        self._count += 1

        feed = Feed(context=contexts, langs=langs)
        return feed

    def _reset(self):
        if self._file is not None:
            self._file.close()
            self._mark_done()

        self._filename = self._choose_next_file()
        logging.info('starting to read {}'.format(self._filename))
        path = os.path.join(self._in_dir, self._filename)
        self._file = gzip.open(path, 'r')
        self._count = 0

    def _mark_done(self):
        self._already_read.add(self._filename)

    def _my_shard(self, idx):
        return idx%self._num_shards == self._shard

    def _choose_next_file(self) -> str:
        while True:
            candidates = [f for f in os.listdir(self._in_dir) if f.endswith(self._ext)]

            # Wait until all shards can start training before returning
            if len(candidates) < self._num_shards:
                time.sleep(10)
                continue

            # sort before sharding via enumerate!
            candidates.sort()
            candidates = [f for idx, f in enumerate(candidates) if self._my_shard(idx)]

            # shard before filtering via already_read!
            candidates = [f for f in candidates if f not in self._already_read]
            if candidates:
                return candidates[0]

            # No new files available, sleep until a new file appears
            time.sleep(10)

    def stop(self):
        if self._file is not None:
            self._file.close()
