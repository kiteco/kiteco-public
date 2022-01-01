import argparse
import itertools
import json
import random
from typing import Dict, List, NamedTuple

import matplotlib.pyplot as plt # type: ignore


def main() -> None:
    args = parse_args()

    with open(args.relevant, "r") as f:
        relevant = json.load(f)

    with open(args.retrieved, "r") as f:
        retrieved = json.load(f)

    with open(args.quip_titles, "r") as f:
        quip_titles = json.load(f)

    with open(args.issue_titles, "r") as f:
        issue_titles = json.load(f)

    docs = DocSet(retrieved, relevant, quip_titles, issue_titles)
    histogram = Histogram(docs)
    report = "\n\n".join([
        histogram.format(),
        docs.format_examples(50),
    ])

    with open(args.results, "w") as f:
        f.write(report)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--retrieved", type=str)
    parser.add_argument("--relevant", type=str)
    parser.add_argument("--quip_titles", type=str)
    parser.add_argument("--issue_titles", type=str)
    parser.add_argument("--results", type=str)
    return parser.parse_args()


class Example:
    def __init__(
            self,
            quip: str,
            retrieved: List[str],
            relevant: List[str],
        ) -> None:

        self.quip = quip
        self.retrieved = retrieved
        self.relevant = relevant

    def format(
            self,
            quip_titles: Dict[str, str],
            issue_titles: Dict[str, str],
        ) -> str:

        title = format_quip(self.quip, quip_titles,)

        quips = [format_issue(r, issue_titles) for r in self.retrieved]
        hits = ["Yes" if r in self.relevant else "" for r in self.retrieved]
        data = [
            (i+1, quip, hit)
            for i, (quip, hit) in enumerate(zip(quips, hits))
        ]
        num_retrieved = 10
        retrieved_rows = [
            f"|{rank}|{quip}|{hit}|"
            for (rank, quip, hit) in data[:num_retrieved]
        ]
        relevant_rows = [
            f"|{rank}|{quip}|{hit}|"
            for (rank, quip, hit) in data[num_retrieved:] if hit
        ]
        retrieved = "\n".join([
            f"### Top {num_retrieved} retrieved:",
            "|Rank|Document|Relevant|",
            "|-|-|-|",
        ] + retrieved_rows)
        relevant = "\n".join([
            f"### Relevant but not in top {num_retrieved} retrieved:",
            "|Rank|Document|Relevant|",
            "|-|-|-|",
        ] + relevant_rows)

        return "\n\n".join([
            f"## {title}",
            retrieved,
            relevant,
        ])


class DocSet:
    def __init__(
            self,
            retrieved: Dict[str, List[str]],
            relevant: Dict[str, List[str]],
            quip_titles: Dict[str, str],
            issue_titles: Dict[str, str],
        ) -> None:

        self.retrieved = retrieved
        self.relevant = relevant
        self.quip_titles = quip_titles
        self.issue_titles = issue_titles

    def format_examples(self, num_examples: int) -> str:
        quips = sorted(k for k, v in self.retrieved.items() if v is not None)
        random.seed(0)
        sample = random.sample(quips, min(len(quips), num_examples))
        relevant = [quip for quip in sample if quip in self.relevant]
        examples = [self.make_example(quip) for quip in relevant]
        subsections = [
            example.format(self.quip_titles, self.issue_titles)
            for example in examples
        ]
        return "\n\n".join(["# Examples:"] + subsections)

    def make_example(self, quip: str) -> Example:
        return Example(quip, self.retrieved[quip], self.relevant[quip])


class Histogram:
    def __init__(self, docs: DocSet) -> None:
        quips = sorted(
            k for k, v in docs.retrieved.items()
            if v is not None
        )
        self.buckets: List[float] = []
        self.nobucket: float = 0
        self.num_quip = len(docs.quip_titles)
        self.num_issues = len(docs.issue_titles)
        for quip in quips:
            ranking = {f: i for i, f in enumerate(docs.retrieved[quip])}
            if quip not in docs.relevant:
                continue
            for issue in docs.relevant[quip]:
                if issue not in ranking:
                    self.nobucket += 1 / len(docs.relevant[quip])
                    continue
                bucket = ranking[issue]
                while bucket >= len(self.buckets):
                    self.buckets.append(0)
                self.buckets[bucket] += 1 / len(docs.relevant[quip])

    def format(self) -> str:
        total = sum(self.buckets) + self.nobucket
        exact_pdf = [100 * h / total for h in self.buckets]
        exact_cdf = list(itertools.accumulate(exact_pdf))
        area_under_cdf = compute_area_under_cdf(exact_cdf)
        plot_cdf(exact_cdf, "histogram.png")
        pdf = list(map(round, exact_pdf))
        cdf = list(map(round, exact_cdf))
        unranked_pct = round(100 * self.nobucket / total)
        summary = "\n".join([
            f"- Number of quip documents: {self.num_quip}",
            f"- Number of issues: {self.num_issues}",
            f"- Normalized area under CDF: {round(area_under_cdf, 3)}",
        ])
        chart = f"![](histogram.png)"
        table = "\n".join([
            "|Ranking|Weighted Frequency|Percent|Cumulative|",
            "|-|-|-|-|",
        ] + [
            f"|{r+1}|{round(f, 2)}|{p}|{c}|"
            for r, (f, p, c) in enumerate(zip(self.buckets, pdf, cdf))
        ] + [
            f"|Unranked|{self.nobucket}|{unranked_pct}|100|",
        ])
        return "\n\n".join(["# Histogram:", summary, chart, table])


def compute_area_under_cdf(cdf: List[float]) -> float:
    return (sum(cdf) - 0.5*cdf[-1]) / (100*len(cdf))


def plot_cdf(exact_cdf: List[float], path: str) -> None:
    plt.style.use("ggplot")
    fig, ax = plt.subplots(figsize=(7, 7))
    ax.set_title("Cumulative distribution")
    ax.plot(list(range(len(exact_cdf) + 1)), [0.0] + exact_cdf)
    ax.set_xlim(-1, len(exact_cdf))
    ax.set_ylim(0, 100)
    ax.set_xlabel("rank")
    ax.set_ylabel("percent")
    fig.savefig(path)


def format_quip(suffix: str, quip_titles: Dict[str, str]) -> str:
    title = quip_titles[suffix]
    quip = "https://kite.quip.com"
    return f"[{title}]({quip}/{suffix})"


def format_issue(number: str, issue_titles: Dict[str, str]) -> str:
    title = issue_titles[number].replace("[", "\[").replace("]", "\]")
    quip = "https://github.com/kiteco/kiteco/issues"
    return f"[{title}]({quip}/{number})"


if __name__ == "__main__":
    main()
