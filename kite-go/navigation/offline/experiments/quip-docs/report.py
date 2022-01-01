import argparse
import itertools
import json
import random
from typing import Dict, List


def main() -> None:
    args = parse_args()

    with open(args.relevant, "r") as f:
        relevant = json.load(f)

    with open(args.retrieved, "r") as f:
        retrieved = json.load(f)

    with open(args.titles, "r") as f:
        titles = json.load(f)

    docs = DocSet(retrieved, relevant, titles)
    histogram = Histogram(docs)
    report = "\n\n".join([
        histogram.format(),
        docs.format_documents(),
        docs.format_examples(50),
    ])

    with open(args.results, "w") as f:
        f.write(report)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--retrieved", type=str)
    parser.add_argument("--relevant", type=str)
    parser.add_argument("--titles", type=str)
    parser.add_argument("--results", type=str)
    return parser.parse_args()


class Example:
    def __init__(
            self,
            path: str,
            retrieved: List[str],
            relevant: Dict[str, List[str]],
        ) -> None:

        self.path = path
        self.retrieved = retrieved
        self.relevant = relevant

    def format(self, titles: Dict[str, str]) -> str:
        title = format_github_code(self.path)
        return "\n\n".join([
            f"## {title}",
            self.format_relevant(titles),
            self.format_retrieved(titles),
        ])

    def format_retrieved(self, titles: Dict[str, str]) -> str:
        paths = [format_quip(r, titles) for r in self.retrieved]
        hits = ["Yes" if r in self.relevant else "" for r in self.retrieved]
        pulls = [
            ", ".join(format_github_pull(p) for p in self.relevant.get(r, []))
            for r in self.retrieved
        ]
        return "\n".join([
            "### Retrieved:",
            "|Rank|Document|Relevant|Pulls|",
            "|-|-|-|-|",
        ] + [
            f"|{rank+1}|{path}|{hit}|{pull}|"
            for rank, (path, hit, pull) in enumerate(zip(paths, hits, pulls))
        ])

    def format_relevant(self, titles: Dict[str, str]) -> str:
        retrieved_set = set(self.retrieved)
        missing = [
            format_quip(r, titles)
            for r in self.relevant
            if r not in retrieved_set
        ]
        if not missing:
            return ""
        return "\n".join(
            ["### Relevant but not retrieved:"] + [f"- {r}" for r in missing]
        )


class DocSet:
    def __init__(
            self,
            retrieved: Dict[str, List[str]],
            relevant: Dict[str, Dict[str, List[str]]],
            titles: Dict[str, str],
        ) -> None:

        self.retrieved = retrieved
        valid_docs = {
            doc
            for docs in retrieved.values()
            for doc in docs
        }
        self.relevant = {}
        for code, docs in relevant.items():
            valids = {
                doc: pulls
                for doc, pulls in docs.items()
                if doc in valid_docs
            }
            if not valids:
                continue
            self.relevant[code] = valids
        self.titles = titles

    def format_documents(self) -> str:
        documents = sorted({doc for c in self.relevant.values() for doc in c})
        links = [format_quip(r, self.titles) for r in documents]
        return "\n".join(["# Documents:"] + [f"- {link}" for link in links])

    def format_examples(self, num_examples: int) -> str:
        paths = sorted(k for k, v in self.retrieved.items() if v is not None)
        random.seed(0)
        sample = random.sample(paths, num_examples)
        relevant = [path for path in sample if path in self.relevant]
        examples = [self.make_example(path) for path in relevant]
        subsections = [example.format(self.titles) for example in examples]
        return "\n\n".join(["# Examples:"] + subsections)

    def make_example(self, path: str) -> Example:
        return Example(path, self.retrieved[path], self.relevant[path])


class Histogram:
    def __init__(self, docs: DocSet) -> None:
        unique = {doc for c in docs.relevant.values() for doc in c}
        self.num_docs = len(unique)
        paths = sorted(
            k for k, v in docs.retrieved.items()
            if v is not None
        )
        self.buckets: List[int] = []
        self.nobucket = 0
        for path in paths:
            ranking = {f: i for i, f in enumerate(docs.retrieved[path])}
            if path not in docs.relevant:
                continue
            for rel in docs.relevant[path]:
                if rel not in ranking:
                    self.nobucket += 1
                    continue
                bucket = ranking[rel]
                while bucket >= len(self.buckets):
                    self.buckets.append(0)
                self.buckets[bucket] += 1

    def format(self) -> str:
        total = sum(self.buckets) + self.nobucket
        exact_pdf = [100 * h / total for h in self.buckets]
        exact_cdf = list(itertools.accumulate(exact_pdf))
        pdf = list(map(round, exact_pdf))
        cdf = list(map(round, exact_cdf))
        unranked_pct = round(100 * self.nobucket / total)
        summary = "\n".join([
            f"- Total samples: {total}",
            f"- Number of documents: {self.num_docs}",
        ])
        table = "\n".join([
            "|Ranking|Frequency|Percent|Cumulative|",
            "|-|-|-|-|",
        ] + [
            f"|{r+1}|{f}|{p}|{c}|"
            for r, (f, p, c) in enumerate(zip(self.buckets, pdf, cdf))
        ] + [
            f"|Unranked|{self.nobucket}|{unranked_pct}|100|",
        ])
        return "\n\n".join(["# Histogram:", summary, table])


def format_github_code(path: str) -> str:
    github = "https://github.com/kiteco/kiteco/blob/master"
    return f"[`{path}`]({github}/{path})"


def format_github_pull(pull: str) -> str:
    github = "https://github.com/kiteco/kiteco/pull"
    return f"[{pull}]({github}/{pull})"


def format_quip(suffix: str, titles: Dict[str, str]) -> str:
    title = titles[suffix]
    quip = "https://kite.quip.com"
    return f"[`{title}`]({quip}/{suffix})"


if __name__ == "__main__":
    main()
