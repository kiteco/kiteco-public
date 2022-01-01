import argparse
import datetime
import json
import logging
import os
import re
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
    links = get(os.environ["GITHUB_AUTH_TOKEN"], "kiteco/kiteco", 12)
    write(links, args.links)
    logging.info("Finished successfully")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--links", type=str)
    return parser.parse_args()


def get(token: str, repo_name: str, months: int) -> Dict[str, List[str]]:
    g = github.Github(token)
    cutoff = datetime.datetime.utcnow() - datetime.timedelta(days=30.4*months)
    repo = g.get_repo(repo_name)

    # The GitHub API treats pull requests as a type of issue.
    # So when we get issues, that includes pull requests.
    # https://docs.github.com/en/rest/reference/issues
    issues = repo.get_issues(state="all")

    links: Dict[str, List[str]] = {}
    regexps = [
        re.compile("https://github.com/kiteco/kiteco/issues/[0-9]+"),
        re.compile("https://github.com/kiteco/kiteco/pull/[0-9]+"),
        re.compile("https://kite.quip.com/[a-zA-Z0-9]+"),
    ]
    for issue in issues:
        time.sleep(0.1)
        if issue.created_at < cutoff:
            break
        if not issue.body:
            logging.info(f"skipping empty body: {issue.html_url}")
            continue
        logging.info(issue.html_url)
        urls = [url for exp in regexps for url in exp.findall(issue.body)]
        if not urls:
            continue
        links[issue.html_url] = urls
        for url in urls:
            logging.info(f" - found url: {url}")
    return links


def write(links: Dict[str, List[str]], linksPath: str) -> None:
    logging.info("Writing links")
    with open(linksPath, "w") as fp:
        json.dump(links, fp, indent=2)


if __name__ == "__main__":
    main()
