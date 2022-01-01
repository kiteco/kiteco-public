import argparse
import datetime
import json
import logging
import os
import pathlib
import time

import github


def main():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )
    args = parse_args()
    data = get_data(args.token, args.repo_name, args.months)
    write(data, args.repo_name)
    logging.info("Finished successfully")
    logging.info(f"Got data for {len(data)} pull requests")


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "token",
        help="An access token from https://github.com/settings/tokens",
    )
    parser.add_argument(
        "repo_name",
        help="The full repo name, e.g. 'vinta/awesome-python'",
    )
    parser.add_argument(
        "--months",
        type=int,
        default=12,
        help="Number of months back to get pull request data from",
    )
    return parser.parse_args()


def get_data(token, repo_name, months):
    """
    Gets pull request data from GitHub.
    Note the API allows 5000 requests per hour.
    We get files for at most one pull request every 750ms.
    https://developer.github.com/v3/#rate-limiting
    """
    prev_data = _get_existing_data(repo_name)

    g = github.Github(token)
    cutoff = datetime.datetime.utcnow() - datetime.timedelta(days=30.4*months)
    cutoff_fmt = cutoff.strftime("%Y-%m-%d")
    logging.info(f"Getting pull requests since {cutoff_fmt}")
    repo = g.get_repo(repo_name)
    pulls = repo.get_pulls(state="closed", base="master")

    data = {}
    for pull in pulls:
        if pull.created_at < cutoff:
            break
        created_fmt = pull.created_at.strftime("%Y-%m-%d")
        if str(pull.number) in prev_data:
            logging.info(f"Skipping #{pull.number} ({created_fmt}): {pull.title}")
            continue
        logging.info(f"Getting #{pull.number} ({created_fmt}): {pull.title}")
        hold_until = time.time() + 0.750
        data[pull.number] = [f.filename for f in pull.get_files()]
        time.sleep(max(hold_until - time.time(), 0))
    return data


def write(data, repo_name):
    filename = _get_filename(repo_name)
    logging.info(f"Writing pull request data to {filename}")
    os.makedirs(os.path.dirname(filename), exist_ok=True)
    with open(filename, "w") as fp:
        json.dump(data, fp, indent=2)


def _get_filename(repo_name):
    home = pathlib.Path.home()
    kite_home = (os.path.join(home, ".kite") if os.name != "nt"
                 else os.path.join(home, "AppData", "Local", "Kite"))
    owner, repo = repo_name.split("/")
    dirname = os.path.join(kite_home, "github", owner, repo)
    return os.path.join(dirname, "pulls.json")


def _get_existing_data(repo_name):
    filename = _get_filename(repo_name)
    if not os.path.exists(filename):
        return {}
    with open(filename, "r") as f:
        return json.loads(f.read())


if __name__ == "__main__":
    main()
