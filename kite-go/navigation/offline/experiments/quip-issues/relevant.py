import argparse
import json
from collections import defaultdict
from typing import Dict, List

def main() -> None:
    args = parse_args()

    with open(args.links, "r") as fp:
        links = json.load(fp)

    quip_issues = make_quip_issues(links)

    with open(args.relevant_issues, "w") as f:
        json.dump(quip_issues, f, indent=2)


def make_quip_issues(links: Dict[str, List[str]]) -> Dict[str, List[str]]:
    quip_issues: Dict[str, List[str]] = defaultdict(list)
    for x, ys in links.items():
        for y in set(ys):
            quip_issues[y.split("/")[-1]].append(x.split("/")[-1])
    return quip_issues


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--links", type=str)
    parser.add_argument("--relevant_issues", type=str)
    return parser.parse_args()


if __name__ == "__main__":
    main()
