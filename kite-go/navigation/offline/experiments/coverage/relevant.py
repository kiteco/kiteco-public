import argparse
import collections
import csv
import json
import os
from typing import Dict, List, Set

import analyze


def main() -> None:
    args = parse_args()
    commits = read_git_cache(args.git_cache, args.repo_root)
    retrieved = read_retrieved(args.retrieved_path)
    relevant = get_relevant(commits, retrieved, args.repo_root)

    with open(args.relevant_path, "w") as fp:
        json.dump(relevant, fp, indent=2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--git_cache", type=str)
    parser.add_argument("--repo_root", type=str)
    parser.add_argument("--retrieved_path", type=str)
    parser.add_argument("--relevant_path", type=str)
    return parser.parse_args()


def read_git_cache(git_cache: str, repo_root: str) -> Dict[str, List[str]]:
    with open(git_cache, "r") as fp:
        data = json.load(fp)

    repo = data["Repos"][repo_root]
    commits = repo["Commits"]
    files = repo["Files"]
    return {
        base: [files[i] for i in idxs]
        for base, idxs in commits.items()
        if idxs
    }


def read_retrieved(path: str) -> Set[str]:
    retrieved = set()
    with open(path, "r") as fp:
        reader = csv.reader(fp)
        for x, y, _ in reader:
            retrieved.add(x)
            retrieved.add(y)
    return retrieved


def get_relevant(
        commits: Dict[str, List[str]],
        retrieved: Set[str],
        repo_root: str,
    ) -> Dict[str, Dict[str, float]]:

    tests: Dict[str, Dict[str, float]] = collections.defaultdict(
        lambda: collections.defaultdict(float)
    )
    for commit in commits.values():
        exists = [
            f for f in commit
            if stat_ok(os.path.join(repo_root, f)) and f in retrieved
        ]
        commit_codes = [f for f in exists if not analyze.is_test(f)]
        commit_tests = [f for f in exists if analyze.is_test(f)]
        for code in commit_codes:
            for test in commit_tests:
                tests[code][test] += 1 / len(commit_codes)
    return tests


def stat_ok(path: str) -> bool:
    try:
        os.lstat(path)
        return True
    except FileNotFoundError:
        return False


if __name__ == "__main__":
    main()
