import argparse
import collections
import csv
import itertools
import json
import pathlib
from typing import Dict, List, NamedTuple

import matplotlib.pyplot as plt # type: ignore


def main() -> None:
    args = parse_args()
    files = [
        "kite-go/navigation/recommend/recommend.go",
        "kite-go/client/internal/kitelocal/internal/navigation/manager.go",
        "kite-go/lang/language.go",
        "kite-go/client/internal/kitelocal/internal/completions/lexical.go",
        "kite-go/lang/lexical/lexicalcomplete/lexicalproviders/Data_inputs.go",
        "kite-go/lang/python/pythoncomplete/driver/mixing.go",
        "kite-go/lang/python/pythondocs/index.go",
    ]
    dirs = [
        "",
        "kite-go/client/internal",
        "kite-go",
        "kite-go/lang/python",
        "kite-golib",
        "kite-golib/lexicalv0",
        "kite-python",
    ]

    commits = Analyzer(
        args.commits_retrieved_path,
        args.relevant_path,
        args.max_tests,
    )
    commits_histogram = commits.histogram()

    text = Analyzer(
        args.text_retrieved_path,
        args.relevant_path,
        args.max_tests,
    )
    text_histogram = text.histogram()

    assert len(commits.tests) == len(text.tests)

    plot_histograms(
        args.histogram_path,
        commits_histogram,
        text_histogram,
        args.max_tests,
        len(commits.tests),
    )

    markdown = to_markdown(
        args.histogram_path,
        [commits.query_file(f) for f in files],
        [text.query_file(f) for f in files],
        [commits.query_directory(d) for d in dirs],
    )
    write(args.results_path, markdown)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--max_tests", type=int)
    parser.add_argument("--commits_retrieved_path", type=str)
    parser.add_argument("--text_retrieved_path", type=str)
    parser.add_argument("--relevant_path", type=str)
    parser.add_argument("--histogram_path", type=str)
    parser.add_argument("--results_path", type=str)
    return parser.parse_args()


class RetrievedResult(NamedTuple):
    path: str
    weight: float
    idx: int
    coverage: float


class MissingResult(NamedTuple):
    path: str
    weight: float


class FileResult(NamedTuple):
    path: str
    retrieved: List[RetrievedResult]
    missing: List[MissingResult]


class DirResult(NamedTuple):
    path: str
    coverage: float
    num_files: int


class Test(NamedTuple):
    path: str
    rank: int


class Sample(NamedTuple):
    path: str
    test: Test


class Histogram:
    def __init__(self, size: int) -> None:
        self.pdf = [0. for _ in range(size)]
        self.unranked = 0.

    def cdf(self) -> List[float]:
        acc = list(itertools.accumulate([0.] + self.pdf))
        total = acc[-1] + self.unranked
        return [100. * a / total for a in acc]

    def add_ranked(self, rank: int, value: float) -> None:
        self.pdf[rank] += value

    def add_unranked(self, value: float) -> None:
        self.unranked += value


def plot_histograms(
        path: str,
        commits: Histogram,
        text: Histogram,
        xmax: int,
        total_tests: int,
    ) -> None:

    plt.style.use("ggplot")
    fig, ax = plt.subplots(figsize=(7, 7))
    cdfs = [commits.cdf(), text.cdf()]
    labels = ["With commits", "Text only"]
    for cdf, label in zip(cdfs, labels):
        ax.plot(
            list(range(len(cdf) + 1)),
            [0.0] + cdf,
            label=label,
        )
    ax.set_title("Cumulative distribution")
    ax.set_xlim(0, xmax)
    ax.set_ylim(0, 100)
    ax.set_xlabel(f"test rank (out of {total_tests} test files)")
    ax.set_ylabel("percent")
    ax.legend(loc="lower right")
    fig.savefig(path)


class Analyzer:
    def __init__(
            self,
            retrieved_path: str,
            relevant_path: str,
            max_tests: int,
        ) -> None:

        self.max_tests = max_tests
        self.data: List[Sample] = []
        self.tests = set()
        with open(retrieved_path, "r") as fp:
            reader = csv.reader(fp)
            batches = itertools.groupby(reader, key=lambda x: x[0])
            for base, batch in batches:
                if is_test(base):
                    continue
                recs = [rec for _, rec, _ in batch]
                for test, idx in find_tests(recs):
                    self.data.append(Sample(base, Test(test, idx)))
                    self.tests.add(test)
        with open(relevant_path, "r") as fp:
            self.relevant: Dict[str, Dict[str, float]] = json.load(fp)

    def histogram(self) -> Histogram:
        histogram = Histogram(len(self.tests))
        for path, weighted_tests in self.relevant.items():
            result = self.query_file(path)
            ranks = {r.path: i for i, r in enumerate(result.retrieved)}
            for test, weight in weighted_tests.items():
                if test in ranks:
                    histogram.add_ranked(ranks[test], weight)
                else:
                    histogram.add_unranked(weight)
        return histogram

    def query_file(self, path: str) -> FileResult:
        retrieved = [
            RetrievedResult(
                path=test.path,
                weight=self.get_relevant(base, test.path),
                idx=test.rank,
                coverage=appraise(test.rank),
            )
            for base, test in self.data
            if base == path
        ][:self.max_tests]
        retrieved_tests = {r.path for r in retrieved}
        missing = [
            MissingResult(path=test, weight=weight)
            for test, weight in self.relevant.get(path, {}).items()
            if test not in retrieved_tests
        ]
        return FileResult(path, retrieved, missing)

    def query_directory(self, path: str) -> List[DirResult]:
        if path != "" and not path.endswith("/"):
            path += "/"
        coverages: Dict[str, float] = collections.defaultdict(float)
        files = collections.defaultdict(set)
        depth = len(path.split("/"))
        for base, test in self.data:
            if not base.startswith(path):
                continue
            parts = base.split("/")
            if len(parts) == depth:
                continue
            group = "/".join(parts[:depth])
            coverages[group] += appraise(test.rank)
            files[group].add(base)
        dirs = [
            DirResult(
                path=g,
                coverage=coverages[g] / len(files[g]),
                num_files=len(files[g]),
            )
            for g in coverages
        ]
        return sorted(dirs, key=lambda d: d.coverage, reverse=True)

    def get_relevant(self, base: str, test: str) -> float:
        if base not in self.relevant or test not in self.relevant[base]:
            return 0
        return self.relevant[base][test]


def find_tests(paths: List[str]) -> List[Test]:
    recs = []
    for idx, rec in enumerate(paths):
        if is_test(rec):
            recs.append(Test(rec, idx))
    return recs or [Test("", -1)]


def is_test(path: str) -> bool:
    return "test" in path


def appraise(idx: int) -> float:
    if idx == -1:
        return 0
    return 2. ** -idx


def to_markdown(
        histogram_path: str,
        commits_files: List[FileResult],
        text_files: List[FileResult],
        directories: List[List[DirResult]],
    ) -> str:

    return "\n\n".join([
        markdown_histograms(histogram_path),
        markdown_files("Files using commits", commits_files),
        markdown_files("Files using text only", text_files),
        markdown_directories(directories),
    ])


def markdown_histograms(histogram_path: str) -> str:
    return "\n".join([
        "# Histograms",
        "",
        "Note validation data leaks into training data when using commits.",
        "",
        f"![]({histogram_path})",
    ])


def markdown_files(
        group_title: str,
        files: List[FileResult],
    ) -> str:

    lines = [f"# {group_title}"]
    for f in files:
        total_coverage = sum(r.coverage for r in f.retrieved)
        lines += [
            "",
            f"## {fmt_path(f.path)}",
            "",
            f"Coverage: {fmt_float(total_coverage)}",
        ]
        total_coverage = sum(r.coverage for r in f.retrieved)
        lines += [
            "",
            "Retrieved:",
            "",
            "|Test rank|Total rank|Coverage|Test|Weighted Hits|",
            "|-|-|-|-|-|",
        ] + [
            fmt_retrieved(i, r)
            for i, r in enumerate(f.retrieved)
        ]
        if not f.missing:
            continue
        lines += [
            "",
            "Relevant but not retrieved:",
            "",
            "|Test|Weighted Hits|",
            "|-|-|",
        ] + [
            f"|{r.path}|{r.weight}|"
            for r in f.missing
        ]
    return "\n".join(lines)


def markdown_directories(dirs: List[List[DirResult]]) -> str:
    lines = [
        "",
        "# Directories",
    ]
    for d in dirs:
        lines += [
            "",
            "|Directory|Coverage|Number of files|",
            "|-|-|-|",
        ]
        lines += [
            f"|{fmt_path(r.path)}|{fmt_float(r.coverage)}|{r.num_files}|"
            for r in d
            if r.num_files > 1
        ]
    return "\n".join(lines)


def fmt_path(path: str) -> str:
    github = "https://github.com/kiteco/kiteco/blob/master"
    return f"[`{path}`]({github}/{path})"


def fmt_float(num: float) -> str:
    return "{:.6f}".format(num)


def fmt_retrieved(i: int, r: RetrievedResult) -> str:
    coverage = fmt_float(r.coverage)
    path = fmt_path(r.path)
    return f"|{i}|{r.idx}|{coverage}|{path}|{r.weight}|"


def write(path: str, text: str) -> None:
    with open(path, "w") as fp:
        fp.write(text)


if __name__ == "__main__":
    main()
