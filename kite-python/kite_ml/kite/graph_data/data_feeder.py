
from typing import List

import datetime

import requests

import time

import logging

import threading

import urllib

from .session import RawSessionResponse, RawSample, Request, RequestInit

from ..model.model import DataFeeder


class EndpointDataFeeder(DataFeeder):
    def __init__(self, endpoint: str, req: RequestInit, retry_wait=120, retry_attempts=5):
        """
        @param retry_wait: time in seconds to wait between retrying the endpoint.
        @param retry_attempts: number of times to retry the endpoint consecutively before throwing an exception.
        """

        self._base_endpoint = endpoint
        self._endpoint = urllib.parse.urljoin(endpoint, 'session')
        self._retry_attempts = retry_attempts
        self._retry_wait = retry_wait

        self._request_times: List[datetime.timedelta] = []
        self._decode_times: List[datetime.timedelta] = []
        self._unpack_times: List[datetime.timedelta] = []

        resp = requests.post(self._endpoint, json=req.to_json())

        assert resp.status_code == 200, \
            "endpoint {0} responded with code {1}: {2}".format(endpoint, resp.status_code, resp.text)

        raw_resp = RawSessionResponse.from_json(resp.json())

        self._session = raw_resp.session
        self._samples = raw_resp.samples
        self._offset = 0
        self._start_batch = datetime.datetime.now()

        self._done = threading.Event()
        ping_endpoint = urllib.parse.urljoin(self._base_endpoint, 'session/ping')
        t = threading.Thread(target=lambda: self._ping(ping_endpoint), name='pinger')
        t.start()

    def stop(self):
        self._done.set()
        kill_endpoint = urllib.parse.urljoin(self._base_endpoint, 'session/kill')
        requests.post(kill_endpoint, json=Request(self._session).to_json())

    def _ping(self, endpoint: str):
        
        ping_interval = 5 * 60
        lock_interval = 5
        num_locks = int(ping_interval / lock_interval)
        while True:
            resp = requests.post(endpoint, json=Request(self._session).to_json())

            assert resp.status_code == 200, \
                'ping endpoint got status code {0}: {1}'.format(resp.status_code, resp.text)

            for _ in range(num_locks):
                if self._done.wait(timeout=lock_interval):
                    return

    def _maybe_print_times(self):
        def mean(l: List[datetime.timedelta]) -> float:
            if len(l) == 0:
                return 0.
            s = float(sum(l, datetime.timedelta(0)).microseconds / 1000)
            return s/float(len(l))
        if len(self._decode_times) == 3:
            mean_request = mean(self._request_times)
            self._request_times = []
            mean_decode = mean(self._decode_times)
            self._decode_times = []
            mean_unpack = mean(self._unpack_times)
            self._unpack_times = []
            logging.info('feeder means (ms): request: {:.3f}, decode: {:.3f}, unpack: {:.3f}'.format(
                mean_request, mean_decode, mean_unpack))

    def _next(self) -> RawSample:
        if self._offset == len(self._samples):
            end = datetime.datetime.now()
            logging.info('took {0} to process batch'.format(end - self._start_batch))

            start = datetime.datetime.now()
            resp = requests.post(self._endpoint, json={
                'session': self._session,
            })
            self._request_times.append(datetime.datetime.now() - start)

            if resp.status_code != 200:
                for i in range(self._retry_attempts):
                    logging.warning('graph server returned non-zero status code {}: {}'.format(
                        resp.status_code, resp.text))
                    logging.warning('retrying attempt #{}, waiting for {} seconds'.format(i+1, self._retry_wait))
                    time.sleep(self._retry_wait)
                    resp = requests.post(self._endpoint, json={
                        'session': self._session,
                    })
                    if resp.status_code == 200:
                        break
                else:
                    raise RuntimeError('failed to reach endpoint {0} {1} times, '
                                       'got code {2}: {3}'.format(self._endpoint,
                                                                  self._retry_attempts, resp.status_code, resp.text))

            start = datetime.datetime.now()
            resp_json = resp.json()
            self._decode_times.append(datetime.datetime.now() - start)

            self._start_batch = datetime.datetime.now()
            self._offset = 0

            start = datetime.datetime.now()
            self._samples = RawSessionResponse.from_json(resp_json).samples

            self._unpack_times.append(datetime.datetime.now() - start)

            self._maybe_print_times()

        sample = self._samples[self._offset]
        self._offset += 1
        return sample

    def next(self) -> RawSample:
        return self._next()
