import logging
import os
import pickle
import time

from kite.model.model import DataFeeder

from kite.graph_data.session import RawSample


class FileDataFeeder(DataFeeder):
    def __init__(self, in_dir: str, ext='.pickle'):
        """
        Read files matching the specified extension from the specified directory and feed
        the samples pickled in those files. If all files have been read, block until
        new files appear.

        :param in_dir: directory containing the samples
        :param ext: the extension of the files to read
        """
        self._in_dir = in_dir
        self._ext = ext
        self._already_read = set()

        self._filename = None
        self._file = None
        self._count = 0
        self._reset()

    def next(self) -> RawSample:
        try:
            sample: RawSample = pickle.load(self._file)
            self._count += 1
            return sample
        except EOFError:
            self._reset()
            return self.next()

    def _reset(self):
        if self._file is not None:
            self._file.close()
            self._mark_done()

        self._filename = self._choose_next_file()
        logging.info('starting to read {}'.format(self._filename))
        path = os.path.join(self._in_dir, self._filename)
        self._file = open(path, 'rb')
        self._count = 0

    def _mark_done(self):
        logging.info('finished reading {}, got {} samples'.format(
            self._filename, self._count))

        self._already_read.add(self._filename)

    def _choose_next_file(self) -> str:
        while True:
            candidates = [f for f in os.listdir(self._in_dir)
                          if f.endswith(self._ext) and f not in self._already_read]
            if candidates:
                candidates.sort()  # always prefer the earliest file
                return candidates[0]
            # No new files available, so wait a bit and try again
            time.sleep(10)

    def stop(self):
        if self._file is not None:
            self._file.close()
