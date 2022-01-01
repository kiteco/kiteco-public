import argparse
import bisect
import collections
import itertools
from typing import Dict, List, Tuple

import matplotlib.pyplot as plt # type: ignore
import numpy as np # type: ignore


def main() -> None:
    args = parse_args()
    width: int = 1
    assert 100 % width == 0
    data = read(args.records)
    glob, loc = analyze(data, width)
    plot_hits(glob, loc, width, args.hits)
    plot_cdf(glob, loc, width, args.cdf)


class Aggregator:
    def __init__(self, pcts: List[float]) -> None:
        self.pcts = pcts
        self.hits = [0 for _ in self.pcts[:-1]]
        self.misses = [0 for _ in self.pcts[:-1]]

    @property
    def hit_rates(self) -> List[float]:
        return [h / ((h + m ) or 1) for h, m in zip(self.hits, self.misses)]

    @property
    def cdf(self) -> List[float]:
        acc: List[int] = list(itertools.accumulate(self.hits))
        total = acc[-1]
        return [a / total for a in acc]

    def set_cuts(self, values: List[float]) -> None:
        self.cuts: List[float] = list(np.percentile(values, self.pcts))

    def get_bucket(self, value: float) -> int:
        if value == self.cuts[0]:
            return 0
        return bisect.bisect_left(self.cuts, value) - 1

    def add_batch(self, batch: List[Tuple[float, bool]]) -> None:
        for value, is_relevant in batch:
            bucket = self.get_bucket(value)
            if is_relevant:
                self.hits[bucket] += 1
            else:
                self.misses[bucket] += 1


def plot_hits(
        glob: Aggregator,
        loc: Aggregator,
        width: float,
        path: str,
    ) -> None:

    x = [(d + 0.5) * width for d in range(len(glob.hit_rates))]

    plt.style.use("ggplot")
    fig, ax = plt.subplots(figsize=(15, 10))

    ax.plot(x, glob.hit_rates, "-o", alpha=0.5, label="global thresholds")
    ax.plot(x, loc.hit_rates, "-o", alpha=0.5, label="local thresholds")
    ax.set_xlabel(f"score percentile (using {width}% buckets)")
    ax.set_ylabel("hit rate")
    ax.set_xticks(range(0, 101, 10))
    ax.set_yticks([d / 100 for d in range(0, 101, 10)])
    ax.set_ylim(0, 1)
    ax.legend()

    fig.savefig(path)


def plot_cdf(
        glob: Aggregator,
        loc: Aggregator,
        width: float,
        path: str,
    ) -> None:

    x = [(d + 0.5) * width for d in range(len(glob.cdf))]

    plt.style.use("ggplot")
    fig, ax = plt.subplots(figsize=(15, 10))

    ax.plot(x, glob.cdf, "-o", alpha=0.5, label="global thresholds")
    ax.plot(x, loc.cdf, "-o", alpha=0.5, label="local thresholds")
    ax.set_xlabel(f"score percentile (using {width}% buckets)")
    ax.set_ylabel("cdf")
    ax.set_xticks(range(0, 101, 10))
    ax.set_yticks([d / 100 for d in range(0, 101, 10)])
    ax.legend()

    fig.savefig(path)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--records", type=str)
    parser.add_argument("--hits", type=str)
    parser.add_argument("--cdf", type=str)
    return parser.parse_args()


def read(records_path: str) -> Dict[str, List[Tuple[float, bool]]]:
    data: Dict[str, List[Tuple[float, bool]]] = collections.defaultdict(list)
    with open(records_path, "r") as fp:
        for line in itertools.islice(fp, 1, None):
            label, _, _, score, is_relevant = line.strip().split(",")
            data[label].append((float(score), is_relevant == "true"))
    return data


def analyze(
        data: Dict[str, List[Tuple[float, bool]]],
        width: int,
    ) -> Tuple[Aggregator, Aggregator]:

    pcts = [float(d) for d in range(0, 101, int(width))]
    loc = Aggregator(pcts)
    glob = Aggregator(pcts)
    glob.set_cuts([v for batch in data.values() for v, _ in batch])
    for label, batch in data.items():
        loc.set_cuts([v for v, _ in batch])
        loc.add_batch(batch)
        glob.add_batch(batch)
    return glob, loc


if __name__ == "__main__":
    main()
