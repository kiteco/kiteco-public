from typing import NamedTuple, List, Dict

from kite.asserts.asserts import FieldValidator


import datetime
import dateutil.parser
import json

import numpy as np

import matplotlib
matplotlib.use('TkAgg')
import matplotlib.pyplot as plt


percentiles = [25, 50, 75, 95]


def percentiles_str(name: str, data: list) -> str:
    return '{} percentiles ({}): {}'.format(name, percentiles, np.percentile(data, percentiles))


def time_parser():
    return dateutil.parser.parse


def milliseconds(td: datetime.timedelta) -> float:
    return float(td.seconds * 1000) + (float(td.microseconds) / 1000.)


def seconds(td: datetime.timedelta) -> float:
    return float(td.total_seconds())


def microseconds(td: datetime.timedelta) -> float:
    return float(td.seconds * 1000000) + float(td.microseconds)


class Job(NamedTuple):
    start: datetime.datetime
    end: datetime.datetime

    @classmethod
    def from_json(cls, d: dict) -> 'Job':
        v = FieldValidator(cls, d)
        return Job(
            start=v.get('Start', str, build=time_parser()),
            end=v.get('End', str, build=time_parser()),
        )


class Metrics(NamedTuple):
    source: str
    start: datetime.datetime
    end: datetime.datetime
    jobs: List[Job]

    @classmethod
    def from_json(cls, d: dict) -> 'Metrics':
        v = FieldValidator(cls, d)

        return Metrics(
            source=v.get('Source', str),
            start=v.get('Start', str, build=time_parser()),
            end=v.get('End', str, build=time_parser()),
            jobs=v.get_list('Jobs', dict, build_elem=Job.from_json),
        )

    def percent_utilization(self) -> float:
        total = milliseconds(self.end - self.start)
        if total == 0:
            return 0
        jobs = 0
        for j in self.jobs:
            if j.end < j.start:
                # job was cancelled so just use the end time of
                # the prefetcher
                td = self.end - j.start
            else:
                td = j.end - j.start
            jobs += milliseconds(td)
        # we have two workers running in parallel so
        # if jobs > total we count it as 100 percent utilization
        if jobs > total:
            return 100.
        return 100. * float(jobs) / float(total)


def read_data(fname: str, num=100) -> List[Metrics]:
    data = []
    with open(fname) as f:
        for line in f:
            if len(data) >= num > 0:
                return data
            data.append(Metrics.from_json(json.loads(line)))
    return data


def plot_hists(filename: str, label_fstr: str, entries: Dict[str, List[float]]):
    fig, axes = plt.subplots(nrows=len(entries), ncols=1)
    idx = 0
    for s, ps in entries.items():
        ax = axes[idx]
        ax.hist(ps)
        ax.set_xlabel(label_fstr.format(s))
        idx += 1
    plt.tight_layout()
    plt.savefig(filename)


def percent_utilization_by_source(data: List[Metrics]):
    source = {}
    for d in data:
        if d.source not in source:
            source[d.source] = []
        source[d.source].append(d.percent_utilization())

    plot_hists('utilization.png', '% utilization {}', source)
    for s, us in source.items():
        print('stats for {}'.format(s))
        print('  median utilization (% of 100): {}'.format(np.median(us)))
        print('  mean utilization (% of 100): {}'.format(np.mean(us)))
        print('  {}'.format(percentiles_str('utilization', us)))


def lifetime_by_source(data: List[Metrics]):
    source = {}
    for d in data:
        if d.end < d.start:
            continue
        if d.source not in source:
            source[d.source] = []
        source[d.source].append(seconds(d.end - d.start))
    plot_hists('lifetimes.png', 'lifetime in seconds {}', source)
    for s, ss in source.items():
        print('stats for {}'.format(s))
        print('  median lifetime (s): {}'.format(np.median(ss)))
        print('  mean lifetime (s): {}'.format(np.mean(ss)))
        print('  {}'.format(percentiles_str('lifetime', ss)))


def source_stats(data: List[Metrics]):
    source = {}
    for d in data:
        if d.source not in source:
            source[d.source] = []
        source[d.source].append(len(d.jobs))

    for s, js in source.items():
        print('stats for {}'.format(s))
        print('  median num jobs: {}'.format(np.median(js)))
        print('  mean num jobs: {}'.format(np.mean(js)))
        print('  {}'.format(percentiles_str('num jobs', js)))


def completed_job_durations(data: List[Metrics]):
    source = {}
    for d in data:
        if d.source not in source:
            source[d.source] = []
        for j in d.jobs:
            if j.end < j.start:
                continue
            source[d.source].append(milliseconds(j.end - j.start))
    plot_hists('completedjobdurations.png', 'job duration in ms {}', source)
    for s, js in source.items():
        print('stats for {}'.format(s))
        print('  median completed job duration (ms): {}'.format(np.median(js)))
        print('  mean completed job duration (ms): {}'.format(np.mean(js)))
        print('  {}'.format(percentiles_str('completed job duration', js)))


def prefetcher_stats(data: List[Metrics]):
    source = {}
    for d in data:
        source[d.source] = source.get(d.source, 0) + 1
    for s, n in source.items():
        print('got {} {} prefetchers'.format(n, s))


def main():
    data = read_data('data.json', 0)
    percent_utilization_by_source(data)
    lifetime_by_source(data)
    source_stats(data)
    completed_job_durations(data)
    prefetcher_stats(data)


if __name__ == '__main__':
    main()