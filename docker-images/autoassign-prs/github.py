import collections
import calendar
import requests
import json

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
		raise Exception("Failed to patch %s: github said '%s'" % (url, msg["message"]))
	return msg


def put(url, payload):
	r = requests.put("https://api.github.com" + url,
		headers={"Authorization": "token "+API_TOKEN},
		data=json.dumps(payload))
	msg = r.json()
	if r.status_code != 200:
		raise Exception("Failed to put %s: github said '%s'" % (url, msg["message"]))
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
	# first get the pr-level comments, which are comments on specific lines of code
	pr_comments = get("/repos/%s/pulls/%d/comments" % (REPO, pr_number))
	# next get the issue-level comments, which are not tied to any line of code
	issue_comments = get("/repos/%s/issues/%d/comments" % (REPO, pr_number))
	# now combine them and sort by timestamp
	return sorted(pr_comments + issue_comments, key=lambda c: c["created_at"])
