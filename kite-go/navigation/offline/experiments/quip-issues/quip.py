import argparse
import json
import logging
import os
import time
from typing import Dict, NamedTuple

import requests


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )
    args = parse_args()

    with open(args.relevant, "r") as fp:
        relevant = json.load(fp)

    quip = Quip(os.environ["QUIP_AUTH_TOKEN"])
    suffixes = set(relevant.keys())
    logging.info(f"Found {len(suffixes)} suffixes")
    titles: Dict[str, str] = {}
    for suffix in suffixes:
        logging.info(f"Getting {suffix}")
        time.sleep(1)
        doc = quip.get(suffix)
        titles[suffix] = doc.title
        with open(f"{args.docs}/{suffix}.py", "w") as fp:
            fp.write(f'"""\n{doc.contents}\n"""')
    logging.info(f"Retrieved {len(titles)} documents")

    with open(args.titles, "w") as fp:
        json.dump(titles, fp, indent=2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--relevant", type=str)
    parser.add_argument("--docs", type=str)
    parser.add_argument("--titles", type=str)
    return parser.parse_args()


class Doc(NamedTuple):
    title: str
    contents: str


class Quip:
    def __init__(self, token: str) -> None:
        self.base = "https://platform.quip.com/1"
        self.token = token

    def get(self, thread_id: str) -> Doc:
        resp = requests.get(
            f"{self.base}/threads/{thread_id}",
            headers={"Authorization": f"Bearer {self.token}"},
        )
        resp.raise_for_status()
        data = resp.json()
        return Doc(data["thread"]["title"], data["html"])


if __name__ == "__main__":
    main()
