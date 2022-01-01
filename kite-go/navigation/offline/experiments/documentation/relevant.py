import argparse
import collections
import csv
import json
import random
from typing import Dict, List, Set


def main() -> None:
    args = parse_args()

    with open(args.extensions, "r") as f:
        extensions = [line.strip() for line in f if line.strip()]

    data: Dict[str, List[str]] = collections.defaultdict(list)
    with open(args.git, "r") as f:
        csvreader = csv.reader(f)
        for commit, modifed_file in csvreader:
            data[commit].append(modifed_file)

    validation: Dict[str, Set[str]] = collections.defaultdict(set)
    blockers = ["vendor", ".pb.", "symlinkRecursiveParent", "bindata"]
    max_files = 10
    for commit, modified_files in data.items():
        docs, queries = [], []
        for modified_file in modified_files:
            if not any(modified_file.endswith(ext) for ext in extensions):
                continue
            if any(blocker in modified_file for blocker in blockers):
                continue
            if modified_file.lower().endswith("readme.md"):
                docs.append(modified_file)
                continue
            queries.append(modified_file)
        if len(queries) > max_files:
            continue
        for query in queries:
            for doc in docs:
                validation[query].add(doc)
    clean = {query: sorted(docs) for query, docs in sorted(validation.items())}

    with open(args.relevant, "w") as f:
        json.dump(clean, f, indent=2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--git", type=str)
    parser.add_argument("--extensions", type=str)
    parser.add_argument("--relevant", type=str)
    return parser.parse_args()


if __name__ == "__main__":
    main()
