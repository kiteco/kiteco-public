import argparse
import collections
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

    docs = DocSet(retrieved, relevant)
    histogram = Histogram(docs)
    report = "\n\n".join([
        histogram.format(),
        docs.format_documents(),
        docs.format_code(),
        docs.format_examples(),
    ])

    with open(args.results, "w") as f:
        f.write(report)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--retrieved", type=str)
    parser.add_argument("--relevant", type=str)
    parser.add_argument("--results", type=str)
    return parser.parse_args()


Docs = List[str]
DocMap = Dict[str, Docs]


class Example:
    def __init__(self, path: str, retrieved: Docs, relevant: Docs) -> None:
        self.path = path
        self.retrieved = retrieved
        self.relevant = relevant

    def format(self) -> str:
        title = format_path(self.path)
        return "\n\n".join([
            f"## {title}",
            self.format_relevant(),
            self.format_retrieved(),
        ])

    def format_retrieved(self) -> str:
        paths = [format_path(r) for r in self.retrieved]
        hits = ["Yes" if r in self.relevant else "" for r in self.retrieved]
        return "\n".join([
            "### Retrieved:",
            "|Rank|Document|Relevant|",
            "|-|-|-|",
        ] + [
            f"|{rank+1}|{path}|{hit}|"
            for rank, (path, hit) in enumerate(zip(paths, hits))
        ])

    def format_relevant(self) -> str:
        retrieved_set = set(self.retrieved)
        missing = [
            format_path(r)
            for r in self.relevant
            if r not in retrieved_set
        ]
        if not missing:
            return ""
        return "\n".join(
            ["### Relevant but not retrieved:"] + [f"- {r}" for r in missing]
        )


class DocSet:
    def __init__(self, retrieved: DocMap, relevant: DocMap) -> None:
        self.retrieved = retrieved
        valid_docs = set()
        for docs in retrieved.values():
            if docs is None:
                continue
            for doc in docs:
                valid_docs.add(doc)
        self.relevant = {}
        for k, docs in relevant.items():
            valids = [doc for doc in docs if doc in valid_docs]
            if not valids:
                continue
            self.relevant[k] = valids

    def format_documents(self) -> str:
        documents = sorted({doc for c in self.relevant.values() for doc in c})
        links = [format_path(r) for r in documents]
        return "\n".join(["# Documents:"] + [f"- {link}" for link in links])

    def format_code(self) -> str:
        code = sorted(self.relevant.keys())
        links = [format_path(c) for c in code]
        return "\n".join(["# Code:"] + [f"- {link}" for link in links])

    def format_examples(self) -> str:
        paths = sorted(k for k, v in self.retrieved.items() if v is not None)
        examples = [self.make_example(path) for path in paths]
        subsections = [example.format() for example in examples]
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


def format_path(path: str) -> str:
    github = "https://github.com/kiteco/kiteco/blob/master"
    return f"[`{path}`]({github}/{path})"


if __name__ == "__main__":
    main()
