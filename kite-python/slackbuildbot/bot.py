"""Module for the Slack API handler Bot class"""

import re
import os
import asyncio
import traceback
import collections
import datetime

import slackclient

class Bot(object):
    """Class for handling interfacing between the Slack API and the release commands.

    One bot is created for each request.
    """

    # shared slack info
    bot_slack_id = "UA4M2TENR"
    channel_info = {}
    user_info = {}

    # shared commands dict; initialized once during run.py
    commands = {}

    # command locks to prevent some commands from running at the same time
    cmd_locks = collections.defaultdict(asyncio.Lock)

    def __init__(self, channel, sender):
        """Initialize with the channel and sender to respond to"""

        # slack client
        slack_token = os.environ["SLACK_TOKEN"]
        self._sc = slackclient.SlackClient(slack_token)

        # lookup channel and user
        self.lookup_user(sender)
        self.lookup_channel(channel)

        # set channel and sender for this response
        self.channel = channel
        self.sender = self.user_info[sender]

    async def respond_to(self, text):
        """Look up a match for the text in the commands dict and call the function if it exists"""

        # check command regex for a match
        for regexp in self.commands:
            match = re.search(regexp, text)
            if match is None:
                continue

            # if match found, call the function with the capture groups as params
            params = match.groups()
            self.func_name = self.commands[regexp].__name__
            self.func_start_time = datetime.datetime.now()
            return await self.execute(self.commands[regexp], params)

        # if no match, reply default
        return await self.default_reply()

    async def execute(self, f, params):
        """Wrapper for executing commands to catch and message exceptions"""
        try:
            return await f(self, *params)
        except Exception as e: # pylint: disable=broad-except
            traceback.print_tb(e.__traceback__)
            # if no exception string, add filler
            if not e:
                e = "<empty exception message>"

            print(e)
            self.send("Error while executing `{}{}`:\n{}".format(f.__name__, params, e))

    async def default_reply(self):
        """The default default reply function

        Can be overridden using @default_reply
        """
        self.send("unknown command")

    def msg_prefix(self):
        try:
            return "[{} {}] ".format(self.func_name, self.sender)
        except AttributeError:
            return ""

    def send(self, text, channel=""):
        """Send a slack message from the bot"""
        if not channel:
            channel = self.channel

        r = self._sc.api_call("chat.postMessage", channel=channel, text="{}{}".format(self.msg_prefix(), text), link_names=True)
        if not r["ok"]:
            raise Exception("Slack API returned error: {}".format(r["error"]))

    def reply(self, text):
        """Reply to a slack message"""
        reply_fmt = "@{} {}"
        text = reply_fmt.format(self.sender, text)

        r = self._sc.api_call(
            "chat.postMessage",
            channel=self.channel,
            text="{}{}".format(self.msg_prefix(), text),
            link_names=True)
        if not r["ok"]:
            raise Exception("Slack API returned error: {}".format(r["error"]))

    def upload(self, filename, filepath, title=""):
        """Upload a file to current channel"""
        print("bot upload for filename: {}, filepath: {}, title: {}".format(filename, filepath, title))
        if not title:
            title = filename

        with open(filepath, "rb") as f:

            r = self._sc.api_call(
                "files.upload",
                channels=self.channel,
                file=f,
                filename=filename,
                title=title)
            if not r["ok"]:
                raise Exception("Slack API returned error: {}".format(r["error"]))

    def lookup_channel(self, channel_id):
        """Look up the channel ID and update channel_info"""
        if channel_id in self.channel_info.keys():
            return

        r = self._sc.api_call("conversations.info", channel=channel_id)
        if not r["ok"]:
            raise Exception("Slack API returned error: {}".format(r["error"]))
        info = r["channel"]

        # if channel_id is for a DM channel, also trigger user lookup
        if isDM(channel_id):
            self.lookup_user(info["user"])
            return

        # update info dict
        self.channel_info[channel_id] = info["name"]

    def lookup_user(self, user_id):
        """Look up the user ID and update user_info"""
        if user_id in self.user_info.keys():
            return

        r = self._sc.api_call("users.info", user=user_id)
        if not r["ok"]:
            raise Exception("Slack API returned error: {}".format(r["error"]))
        info = r["user"]
        self.user_info[user_id] = info["name"]

    def require_lock(self, lockname, wait=False):
        """Require that a lock from the Bot cmd_locks be acquired

        If wait is True, it will block until the lock is acquired. If False, it will return
        immediately with an exception

        Until more sophisticated logic is needed, locks should be used at the command level, i.e.
        inside functions in run.py
        """

        lock = self.cmd_locks[lockname]

        # if we are not waiting and the lock is locked, throw exception
        if not wait and lock.locked():
            raise Exception('Could not acquire lock {}'.format(lockname))

        # otherwise, return lock
        return lock

    def release(self, lockname):
        """Release lock from cmd_locks; shorthand convenience method"""
        self.cmd_locks[lockname].release()

class ResponseQueue(object):
    """Class for handling queue of respond_to tasks from bots"""

    def __init__(self):
        # task queue
        self.q = asyncio.Queue()
        # worker list
        self.workers = []

    async def add(self, f):
        """Add a tasks to the queue"""
        await self.q.put(f)

    async def work(self):
        """Run forever and execute tasks"""
        while 1:
            item = await self.q.get()

            try:
                # NOTE: this await should theoretically not throw an exception when used with
                # Bot.execute, since execute does the exception handling, but I've left it in in
                # case in the future this might be called with a function without exception
                # handling.
                await item
            except Exception as e: # pylint: disable=broad-except
                traceback.print_tb(e.__traceback__)
                print(e)
            finally:
                self.q.task_done()

    async def run(self, count=10):
        """Start a number of work calls to execute tasks in the queue"""
        self.workers = [asyncio.ensure_future(self.work()) for i in range(count)]
        await self.q.join()

    def cancel(self):
        """Cancel all workers"""
        for worker in self.workers:
            worker.cancel()

def respond_to(regexp):
    """Decorator for adding a command for the bot to respond to"""
    def wrapper(func):
        """wrapper"""
        Bot.commands[re.compile(regexp)] = func
        return func
    return wrapper

def default_reply(func):
    """Decorator for replacing the default reply"""
    Bot.default_reply = func

def isDM(channel_id):
    """Helper to check if a channel ID is for a DM"""
    return channel_id.startswith("D")
