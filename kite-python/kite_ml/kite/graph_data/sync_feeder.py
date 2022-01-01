import logging
import os
import pickle
import time
import requests
import urllib

from kite.model.model import DataFeeder

from kite.graph_data.session import RawSample


class SyncDataFeeder(DataFeeder):
    def __init__(self, in_dir: str, endpoint: str, ext='.pickle'):
        """
        Read files from disk and coordinate with syncer to continue
        running through the data set until we are stopped

        :param in_dir: directory containing the samples
        :param ext: the extension of the files to read
        """
        self._in_dir = in_dir
        self._ext = ext
        self._already_read = set()
        self._endpoint = endpoint

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

        resp = requests.post(
            urllib.parse.urljoin(self._endpoint, '/used'),
            json={'used': [self._filename]},
        )

        logging.info('marking {} as done and posted to used with {}'.format(self._filename, resp.status_code))

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
