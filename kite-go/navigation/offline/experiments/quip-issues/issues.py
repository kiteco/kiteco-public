import argparse
import datetime
import logging
import json
import re
import os
import time
from typing import Dict, List, NamedTuple, Tuple

import github


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )
    args = parse_args()
    issues, links = get(os.environ["GITHUB_AUTH_TOKEN"], "kiteco/kiteco", 12)
    write(issues, links, args)
    logging.info("Finished successfully")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--issues", type=str)
    parser.add_argument("--titles", type=str)
    parser.add_argument("--links", type=str)
    return parser.parse_args()


class Issue(NamedTuple):
    number: str
    title: str
    body: str


def get(
        token: str,
        repo_name: str,
        months: int,
    ) -> Tuple[List[Issue], Dict[str, List[str]]]:

    g = github.Github(token)
    cutoff = datetime.datetime.utcnow() - datetime.timedelta(days=30.4*months)
    repo = g.get_repo(repo_name)

    # The GitHub API treats pull requests as a type of issue.
    # So when we get issues, that includes pull requests.
    # https://docs.github.com/en/rest/reference/issues
    issues = repo.get_issues(state="all")

    processed: List[Issue] = []
    links: Dict[str, List[str]] = {}
    regexp = re.compile("https://kite.quip.com/[a-zA-Z0-9]+")
    for issue in issues:
        time.sleep(0.1)
        logging.info(f"{issue.number} ({issue.created_at}): {issue.title}")
        if issue.created_at < cutoff:
            break
        if issue.pull_request is not None:
            # We exclude issues which are pull requests.
            continue
        urls = [url for url in regexp.findall(issue.body)]
        if not urls:
            logging.info(f"skipping body without quip links: {issue.html_url}")
            continue
        links[issue.html_url] = urls
        time.sleep(1)
        processed.append(Issue(
            number=issue.number,
            title=issue.title,
            body=issue.body,
        ))
        for url in urls:
            logging.info(f" - found url: {url}")
    return processed, links


def write(
        issues: List[Issue],
        links: Dict[str, List[str]],
        args: argparse.Namespace,
    ) -> None:

    logging.info("Writing issues")
    titles: Dict[str, str] = {}
    for issue in issues:
        titles[issue.number] = issue.title

        # Remove any links to quip.
        # The links may contain id strings which are also in the quip doc.
        # This makes it too easy for the recommender.
        clean = "\n".join(
            " ".join(word for word in line.split() if "quip.com" not in word)
            for line in issue.body.splitlines()
        )

        with open(f"{args.issues}/{issue.number}.py", "w") as fp:
            fp.write(f'"""\n{issue.title}\n\n{clean}\n"""')

    logging.info("Writing titles")

    with open(args.titles, "w") as fp:
        json.dump(titles, fp, indent=2)

    with open(args.links, "w") as fp:
        json.dump(links, fp, indent=2)


if __name__ == "__main__":
    main()
