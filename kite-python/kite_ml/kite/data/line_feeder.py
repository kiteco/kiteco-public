from typing import List

import logging
import os
import gzip

from kite.model.model import DataFeeder


class LineFeeder(DataFeeder):
    def __init__(self, in_dir: str, batch_size=20, cycle=True, ext='.json.gz'):
        """
        Read files matching the specified extension from the specified directory and return lines (as a string)
        from each file in a list of length batch_size.

        :param: cycle: if true, once the reader exhausts the list of files in the directory it will start
                       reading the old files again, if false then the reader will block until more files
                       are available.
        :param in_dir: directory containing the samples
        :param ext: the extension of the files to read
        """
        self._in_dir = in_dir
        self._ext = ext
        self._already_read = set()
        self._cycle = cycle
        self._filename = None
        self._file = None
        self._count = 0
        self._batch_size = batch_size

    def all(self) -> List[str]:
        lines = []
        for cand in self._candidates():
            with self._open(cand) as f:
                line = f.readline()
                while len(line) > 0:
                    lines.append(line)
                    line = f.readline()
        return lines

    def next(self) -> List[str]:
        if self._file is None:
            self._reset()
        lines = []
        while len(lines) < self._batch_size:
            line = self._file.readline()
            while len(line) == 0:
                self._reset()
                line = self._file.readline()
            lines.append(line)
        return lines

    def _reset(self):
        if self._file is not None:
            self._file.close()
            self._mark_done()

        self._filename = self._choose_next_file()
        logging.info('starting to read {}'.format(self._filename))
        self._file =self._open(self._filename)
        self._count = 0

    def _open(self, filename: str):
        path = os.path.join(self._in_dir, filename)
        if self._ext.endswith('.gz'):

            return gzip.open(path, 'r')
        return open(path, 'r')

    def _mark_done(self):
        self._already_read.add(self._filename)

    def _choose_next_file(self) -> str:
        while True:
            candidates = self._candidates()

            if candidates:
                return candidates[0]

            if self._cycle:
                # No new files available, start over
                self._already_read = set()
                self._filename = None
                self._file = None
                self._count = 0

    def _candidates(self) -> List[str]:
        candidates = [f for f in os.listdir(self._in_dir)
                      if f.endswith(self._ext) and f not in self._already_read]
        candidates.sort()
        return candidates

    def stop(self):
        if self._file is not None:
            self._file.close()
