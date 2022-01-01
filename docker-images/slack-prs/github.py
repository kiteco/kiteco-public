import collections
import calendar
import json

import requests
import iso8601

API_TOKEN = "XXXXXXX"

REPO = "kiteco/kiteco"


def get(url):
	r = requests.get("https://api.github.com" + url, headers={"Authorization": "token "+API_TOKEN})
	msg = r.json()
	if r.status_code != 200:
		raise Exception("Failed to get %s: github said '%s'" % (url, msg["message"]))
	return msg


def patch(url, payload):
	r = requests.patch("https://api.github.com" + url,
		headers={"Authorization": "token "+API_TOKEN},
		data=json.dumps(payload))
	msg = r.json()
	if r.status_code != 200:
		raise Exception("Failed to get %s: github said '%s'" % (url, msg["message"]))
	return msg


def fetch_prs():
	return get("/repos/%s/pulls" % REPO)


def prs_for(username, prs):
	usertag = "@"+username
	for pr in prs:
		role = None
		if pr["state"] == "open" and usertag in pr["body"]:
			role = "a reviewer"
		elif pr["user"]["login"] == username:
			role = "the author"

		if role is not None:
			yield role, pr


def fetch_comments(pr_number):
	return get("/repos/%s/pulls/%d/comments" % (REPO, pr_number))


def summarize_updates_for(username, comments):
	updates = []
	for comment in comments:
		if comment["user"]["login"] == username:
			updates = []
		else:
			updates.append(comment["user"]["login"])

	update_counts = collections.defaultdict(int)
	for user in updates:
		update_counts[user] += 1

	return update_counts


def summarize_updates_since(username, comments, since):
	updates = []
	for comment in comments:
		t = parse_timestamp(comment["created_at"])
		if comment["user"]["login"] == username:
			updates = []
		elif t > since:
			updates.append(comment["user"]["login"])

	update_counts = collections.defaultdict(int)
	for user in updates:
		update_counts[user] += 1

	return update_counts


def parse_timestamp(timestamp):
    """
    Convert a UTC timestamp formatted in ISO 8601 into a UNIX timestamp
    https://gist.github.com/rca/3702066
    """

    # use iso8601.parse_date to convert the timestamp into a datetime object.
    parsed = iso8601.parse_date(timestamp)

    # now grab a time tuple that we can feed mktime
    timetuple = parsed.timetuple()

    # return a unix timestamp
    return calendar.timegm(timetuple)
