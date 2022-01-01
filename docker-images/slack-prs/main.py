import time
import json
import argparse

import websocket
import requests

import github

MY_NAME = 'kit'  # should be able to avoid this in the future
TOKEN = 'XXXXXXX'

GITHUB_USERNAME_BY_SLACK_USERNAME = {
	"adam": "adamsmith",
	# XXXXXXX ...
}

channel_ids_by_name = {}
channel_names_by_id = {}
next_id = 0

def send(conn, channel, text):
	global next_id, last_send_timestamp
	channel_id = channel_ids_by_name.get(channel, channel)
	payload = dict(
		id=next_id,
	    type="message",
	    channel=channel_id,
	    text=text)
	msg = json.dumps(payload)
	conn.send(json.dumps(payload))
	next_id += 1
	last_send_timestamp = time.time()


def slack_escape(s):
	s = s.replace("&", "&amp;")
	s = s.replace("<", "&lt;")
	s = s.replace(">", "&gt;")
	return s


def pr_queue_for(github_username, prs, comments_by_pr):
	response = ""
	for role, pr in github.prs_for(github_username, prs):
		title, url, number = pr["title"], pr["html_url"], pr["number"]

		comments = comments_by_pr.get(number, None)
		if not comments:
			comments = github.fetch_comments(number)
			comments_by_pr[number] = comments

		updates_by_user = github.summarize_updates_for(github_username, comments)

		if len(updates_by_user) == 0:
			update_msg = "no updates"
		else:
			update_msg = ", ".join("%d new from %s" % (count, user) for user, count in updates_by_user.items())

		response += 'you are *%s* for %s %s: *%s*\n' % (role, url, slack_escape(title), update_msg)

	if response == "":
		return "you are not on any pull requests"
	else:
		return response


def updates_since(github_username, prs, comments_by_pr, since):
	response = ""
	for role, pr in github.prs_for(github_username, prs):
		title, url, number = pr["title"], pr["html_url"], pr["number"]

		comments = comments_by_pr.get(number, None)
		if not comments:
			comments = github.fetch_comments(number)
			comments_by_pr[number] = comments

		updates_by_user = github.summarize_updates_since(github_username, comments, since)

		if updates_by_user:
			status = ", ".join("%d new from %s" % (count, user) for user, count in updates_by_user.items())
			response += '*%s* (%s) %s\n' % (status, url, slack_escape(title))

	return response


def main():
	parser = argparse.ArgumentParser()
	parser.add_argument("--daily", action="store_true")
	parser.add_argument("--since", type=str)
	args = parser.parse_args()

	conn = None
	user_ids_by_name = {}
	user_names_by_id = {}
	im_channel_by_user = {}

	# Get messaging setup info
	payload = dict(token=TOKEN)
	r = requests.post('https://slack.com/api/rtm.start', data=payload).json()
	if r["ok"]:
		print("Successfully connected to messaging API")
	else:
		print("Error:\n" + str(r))
		return

	# Unacpk general info
	dial_url = r["url"]

	# Unpack channel info
	users = r["users"]
	for user in users:
		name = user["name"]
		id = user["id"]
		user_ids_by_name[name] = id
		user_names_by_id[id] = name

	# Unpack channel info
	channels = r["channels"]
	for channel in channels:
		name = channel["name"]
		id = channel["id"]
		channel_ids_by_name[name] = id
		channel_names_by_id[id] = name

	for im_channel in r["ims"]:
		im_channel_by_user[user_names_by_id[im_channel["user"]]] = im_channel["id"]

	# Open websocket
	conn = websocket.create_connection(dial_url)
	print("Connected")

	# Send private messages
	prs = github.fetch_prs()
	comments = {}
	if args.daily:
		for user, ch in im_channel_by_user.items():
			github_username = GITHUB_USERNAME_BY_SLACK_USERNAME.get(user, None)
			if github_username:
				print('Sending PM to %s...' % user)
				msg = pr_queue_for(github_username, prs, comments)
				print(msg.replace("\n", "\n    "))
				send(conn, ch, "Here is your daily pull request update:\n" + msg)

	else:
		since = 0
		try:
			if args.since:
				# Read prev timestamp
				with open(args.since) as f:
					since = float(f.read().strip())
				
				# Write new timestamp
				with open(args.since, "w") as f:
					f.write(str(time.time()))
		except (IOError, ValueError):
			pass


		for user, ch in im_channel_by_user.items():
			github_username = GITHUB_USERNAME_BY_SLACK_USERNAME.get(user, None)
			if github_username:
				msg = updates_since(github_username, prs, comments, since)
				if msg:
					print('Sending PM to %s...' % user)
					print(msg)
					send(conn, ch, msg)


if __name__ == '__main__':
	try:
		main()
	except KeyboardInterrupt:
		pass
