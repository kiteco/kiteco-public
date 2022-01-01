import argparse
import datetime
import json
import logging
import os
import time
from typing import Dict, List

import github


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )
    args = parse_args()
    pulls = get(os.environ["GITHUB_AUTH_TOKEN"], "kiteco/kiteco", 12)
    write(pulls, args.pulls)
    logging.info("Finished successfully")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pulls", type=str)
    return parser.parse_args()


def get(token: str, repo_name: str, months: int) -> Dict[int, List[str]]:
    g = github.Github(token)
    cutoff = datetime.datetime.utcnow() - datetime.timedelta(days=30.4*months)
    repo = g.get_repo(repo_name)
    pulls = repo.get_pulls(state="closed", base="master")

    data: Dict[int, List[str]] = {}
    for pull in pulls:
        if pull.created_at < cutoff:
            break
        created_fmt = pull.created_at.strftime("%Y-%m-%d")
        logging.info(f"Getting #{pull.number} ({created_fmt}): {pull.title}")
        hold_until = time.time() + 0.750
        filenames = [f.filename for f in pull.get_files()]
        data[pull.number] = filenames
        time.sleep(max(hold_until - time.time(), 0))
    return data


def write(pulls: Dict[int, List[str]], pullsPath: str) -> None:
    with open(pullsPath, "w") as fp:
        json.dump(pulls, fp, indent=2)


if __name__ == "__main__":
    main()
