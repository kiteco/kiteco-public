"""Server for receiving and handling Slack Event Subscriptions API"""

from aiohttp import web

from bot import Bot, ResponseQueue, isDM

async def slack_event(request):
    """Handler for slack events"""
    # Slack request is always POST with json data
    event = await request.json()

    # debug
    if request.app["debug"]:
        print("="*100)
        import pprint
        pprint.pprint(event)
        print("="*100)

    # handle challenge verification
    if "challenge" in event.keys():
        return web.Response(text=event["challenge"])

    # check if the event is relevant
    e = relevant_event(event)
    if e:
        # catch all exceptions so that we don't return non-200s
        try:
            await handle_event(request.app, e)
        except Exception as e: # pylint: disable=broad-except
            print(e)

    return web.Response()

def relevant_event(event):
    """Check if an event is relevant and returns the inner event dict if it is"""
    if "event" in event.keys():
        e = event["event"]
        # only handle message type events
        if e["type"] == "message":
            return e

    return None

async def handle_event(app, event):
    """Wrapper for handling events so that all exceptions can be caught and handled, to ensure we
    don't return non-200s and cause Slack to repost the message

    NOTE: because this simply adds the respond_to call to the ResponseQueue to be executed by one
    of its workers, this will not wrap any exceptions thrown by the command itself - those are
    handled by Bot.execute
    """

    # ignore messages from bots
    if "bot_id" in event.keys() or event.get('subtype', None) in {"bot_message", "file_share"}:
        print("ignoring bot msg  ", event["text"][:20])
        return

    text = event["text"]
    channel = event["channel"]
    sender = event["user"]

    # if this is not a DM, only listen to messages that contain the bot name tag
    if not isDM(channel):
        bot_tag = "<@{}>".format(Bot.bot_slack_id)
        if bot_tag in text:
            text = strip_bot_tag(text, bot_tag)
        else:
            return

    bot = Bot(channel, sender)
    await app["response_queue"].add(bot.respond_to(text))

def strip_bot_tag(text, bot_tag):
    """Helper to strip the bot tag out of the message"""
    # split on bot tag, strip trailing whitespace, join non-empty with space
    return " ".join([t.strip() for t in text.split(bot_tag) if t])

async def start_background_tasks(a):
    """Start response queue in background at startup"""
    rq = ResponseQueue()
    a["response_queue"] = rq
    a["response_queue_run"] = a.loop.create_task(rq.run())

async def cleanup_background_tasks(a):
    """Cleanup response queue"""
    # first cancel all worker tasks
    a["response_queue"].cancel()
    # then cancel the run() task
    a["response_queue_run"].cancel()
    await a["response_queue_run"]

## Server initialization

def run(debug=False):
    """Run the server"""

    app = web.Application()
    app.router.add_post("/slack", slack_event)
    app.on_startup.append(start_background_tasks)
    app.on_cleanup.append(cleanup_background_tasks)
    app["debug"] = debug

    web.run_app(app, port=8989)
