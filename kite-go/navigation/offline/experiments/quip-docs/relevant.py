import argparse
import json
from collections import defaultdict
from typing import Dict, List


def main() -> None:
    args = parse_args()

    with open(args.links, "r") as fp:
        links = json.load(fp)

    with open(args.pulls, "r") as fp:
        pull_code = json.load(fp)

    with open(args.extensions, "r") as f:
        extensions = [line.strip() for line in f if line.strip()]

    quip_pull = make_quip_pull(links)
    relevant = make_relevant(quip_pull, pull_code, extensions)

    with open(args.relevant, "w") as f:
        json.dump(relevant, f, indent=2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--links", type=str)
    parser.add_argument("--pulls", type=str)
    parser.add_argument("--extensions", type=str)
    parser.add_argument("--relevant", type=str)
    return parser.parse_args()


def make_quip_pull(links: Dict[str, List[str]]) -> Dict[str, List[str]]:
    g = Graph(links)
    quip_pull = {
        doc: [x for x in g.component(doc) if "pull" in x]
        for doc in g.search("quip.com")
    }
    return {doc: pulls for doc, pulls in quip_pull.items() if pulls}


class Graph:
    def __init__(self, links: Dict[str, List[str]]) -> None:
        self.nodes = set()
        self.nbrs = defaultdict(set)
        for x, ys in links.items():
            self.nodes.add(x)
            for y in ys:
                self.nodes.add(y)
                self.nbrs[x].add(y)
                self.nbrs[y].add(x)

    def search(self, substring: str) -> List[str]:
        return sorted(n for n in self.nodes if substring in n)

    def component(self, node: str) -> List[str]:
        seen = set()
        stack = [node]
        while stack:
            n = stack.pop()
            if n in seen:
                continue
            seen.add(n)
            stack.extend(self.nbrs[n])
        return sorted(seen)


def make_relevant(
        quip_pull: Dict[str, List[str]],
        pull_code: Dict[str, List[str]],
        extensions: List[str],
    ) -> Dict[str, Dict[str, List[str]]]:

    triples = sorted(
        (code, quip.split("/")[-1], pull.split("/")[-1])
        for quip, pulls in quip_pull.items()
        for pull in pulls
        for code in pull_code.get(pull.split("/")[-1], [])
    )
    relevant: Dict[str, Dict[str, List[str]]] = defaultdict(
        lambda: defaultdict(list),
    )
    blockers = ["vendor", ".pb.", "symlinkRecursiveParent", "bindata"]
    for code, quip, pull in triples:
        if not any(code.endswith(ext) for ext in extensions):
            continue
        if any(blocker in code for blocker in blockers):
            continue
        relevant[code][quip].append(pull)
    return relevant


if __name__ == "__main__":
    main()
