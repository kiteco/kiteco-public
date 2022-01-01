import time
import json
import random
import traceback
import argparse

import github

def main():
	parser = argparse.ArgumentParser(description="Add labels and assignees to github pull requests")
	parser.add_argument("prs", type=int, nargs="*")
	args = parser.parse_args()

	prs = github.fetch_prs()
	for pr in prs:
		if args.prs and pr["number"] not in args.prs:
			continue

		print(pr["number"])
		body = pr["body"]

		patch = dict()
		reviewers = []
		if body.startswith("@"):
			newline = body.find("\n")
			if newline == -1:
				newline = len(body)
			reviewers = [part[1:] for part in body[:newline].split() if part.startswith("@")]
			print("detected reviewers:", ", ".join(reviewers))

		if pr["assignee"]:
			print("already assigned to", pr["assignee"]["login"])
		elif len(reviewers) > 0:
			print("assigning to", reviewers[0])
			patch["assignee"] = reviewers[0]

		author = pr["user"]["login"]
		comments = github.fetch_comments(pr["number"])

		lgtm = False
		has_other_comments = False
		has_reviewer_comments = False
		last_comment = None
		for comment in comments:
			commenter = comment["user"]["login"]
			if commenter == author:
				last_comment = "author"
			elif commenter in reviewers:
				has_reviewer_comments = True
				last_comment = "reviewer"
			else:
				has_other_comments = True
				last_comment = "other"

			if "LGTM" in comment["body"] or "lgtm" in comment["body"]:
				lgtm = True

		has_dependencies = "Depends on" in pr["body"]

		if lgtm:
			label = "lgtm"
		elif last_comment in (None, "author"):
			label = "needs review"
		else:
			label = "awaiting author"

		print("assigning label:", label)
		github.put("/repos/kiteco/kiteco/issues/%d/labels" % pr["number"], [label])

		if len(patch) > 0:
			print(patch)
			github.patch("/repos/kiteco/kiteco/issues/%d" % pr["number"], patch)


if __name__ == "__main__":
	try:
		main()
	except KeyboardInterrupt:
		pass
