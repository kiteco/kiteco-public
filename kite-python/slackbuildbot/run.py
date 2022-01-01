#!/usr/bin/env python

"""Main process for the slackbot-based release system"""
import random
import argparse

from bot import respond_to, default_reply
import build_commands
import server

# Permission handling
PERMITTED = {"tarak", "juan", "naman", "jonathan", "tony", "ryan", "ed"}
NOT_ALLOWED = [
    "I can't let you do that",
    "My mother told me not to talk to strangers",
    "You can't tell me what to do",
    "You're not the boss of me!",
    "Don't be messing with things you don't fully understand",
    "You are not in the sudoers file.  This incident will be reported",
    "I'm sorry. I'm afraid I can't do that",
    "[Slackbot won't obey! It hurt itself in its confusion!]",
    "I WILL NOT FALTER!",
    "no you",
    ]

# lock list
# NOTE: only use locks defined as constants here
L_REPO = "repo" # lock for commands that change the repo
L_RELEASE = "release" # single shared lock for release commands

def allowed(bot):
    """Check permissions first before allowing a command

    Responds with a mildly amusing message if not allowed
    """

    sender = bot.sender
    if sender in PERMITTED:
        return True

    not_allowed = random.choice(NOT_ALLOWED)
    bot.reply(not_allowed)
    # output user ID in case they need to be added
    return False

@default_reply
async def unknown_command(bot):
    """Show default response for unknown commands"""
    bot.reply("Unknown command. To show a list of commands, type `list commands`")

@respond_to(r"^list commands")
async def list_commands(bot):
    """Show the list of acceptable commands, along with the first line of the docstring"""
    intro = "Write `help <command name>` to show documentation for a command\n\n"
    commands_list = [
        (f.__name__, _sdoc(f.__doc__))
        for f in bot.commands.values()]
    # filter out commands with empty docs
    commands_list = [i for i in commands_list if i[1]]
    commands_text = ".\n".join(["`{}`: {}".format(*t) for t in commands_list])
    bot.send(intro + commands_text)

def _sdoc(docstr):
    """Small helper to get the first line of a docstring

    Mostly to help reduce verbosity in the code
    """
    if docstr is not None:
        split_txt = docstr.strip().split("\n")
        if split_txt:
            return split_txt[0]
    return "" # if empty

@respond_to(r"^help `*([a-zA-Z_ ]+)`*")
async def show_help(bot, command):
    """Show the docstring for the command function as a help message

    Usage: help [command function name]

    Since the docstring serves as both documentation for the code and help text for the slackbot
    interface, the docstring should follow this format:

    ```
    [one-line brief summary]

    Usage: [simplified usage text]

    [more details]
    ```

    The usage format is just the command written out where required words are unbracketed and
    non-required/parameter words are bracketed and can be a short description instead of the
    literal words. See other docstrings for examples. Usage line can be omitted if the command has
    no parameters.
    """
    # find command in command list
    command_f = None
    for reg, func in bot.commands.items():
        if func.__name__ == command:
            command_f = func
            regex_str = reg

    if command_f is None:
        bot.reply(
            "Help for command not found. Type `list commands` for a list of available commands")
    else:
        msg_fmt = "`{}`: {}\n\nCommand regex:\n```{}```".format(
            command_f.__name__, command_f.__doc__.strip(), regex_str.pattern)
        bot.reply(msg_fmt)

cmd_regex = (
    r"^stage (mac|backend|website) ([\w.]+)$"
    )
@respond_to(cmd_regex)
async def stage_release(bot, artifact_type, ref):
    """Stage a new release

    Usage: stage [thing to stage] [Git ref]

    Refer to the command regex for all valid things to stage
    """
    if not allowed(bot):
        return

    # set kwargs based on messsage
    kwargs = {}
    kwargs["prepare"] = False
    kwargs["binaries"] = False
    kwargs["quiet"] = True
    kwargs["backend"] = artifact_type == "backend"
    kwargs["website"] = artifact_type == "website"
    kwargs["client"] = "macos" if artifact_type == "mac" else False
    kwargs["ref"] = ref

    async with bot.require_lock(L_RELEASE):
        cmd_runner = build_commands.CommandRunner(bot)
        await cmd_runner.stage_release(**kwargs)

cmd_regex = (
    r"^build (clients|macos|windows) for testing(| from [\w\-]+)(| without binaries)(| only [\w]+)$"
    )
# pylint: disable=too-many-arguments
@respond_to(cmd_regex)
async def build_test(bot, build, branch, no_binaries, only_plugins):
    """Build a test build (only supports clients at the moment)

    Usage: build <thing to build> for testing [from <branch name>] [without binaries] [only <comma separated plugin list>]

    The functionality of each parameter in the command are as follows:
    - You can specify "clients" for the thing to build to build all clients
    - Adding "without binaries" at the end will skip the bindata build process (you would only
      include this if you know for sure that your changes do not need to be built into bindata)
    - "from <branch_name>" will use the given branch to create the build.
      - Omitting this will build from the latest master of both the main repo and submodules
      - You can also specify "current" to build from whatever state the repo is currently on the
        Solness machine (you would only use this if you have physical access to the machine)
      - Note that to test submodule changes, you need to commit the updated submodule to the branch
        you are testing from
    - "only <comma delimited plugin list>" will only build the specified plugins if included
        and if "without binaries" is not included
        - legal plugin names are: "intellij", "sublime", and "vim"

    Refer to the command regex for all valid things to build
    """
    # set kwargs based on messsage
    kwargs = {}
    kwargs["binaries"] = not no_binaries
    kwargs["build"] = build

    kwargs["only_plugins"] = []
    if only_plugins:
        plugin_list = only_plugins.strip().split("only ")
        if len(plugin_list) == 2:
            kwargs["only_plugins"] = plugin_list[1].strip().split(',')


    kwargs["branch"] = ""
    if branch:
        branch_name = branch.strip().split("from ")
        if len(branch_name) == 2:
            kwargs["branch"] = branch_name[1].strip()

    async with bot.require_lock(L_REPO):
        cmd_runner = build_commands.CommandRunner(bot)
        await cmd_runner.build_test(**kwargs)


CLIENTS = ("macos", "windows", "linux")
PLUGINS = ("vscode", "atom")
cmd_regex = r"^release ([\w%, ]+)"
@respond_to(cmd_regex)
async def release(bot, items):  # pylint: disable=too-many-branches
    """Release staged releases

    Usage: release [items to release]

    Items to release is a comma-separated (spaces optional) list of valid items to release:
    - `all [x%]`: equivalent to `clients [x%], plugins, binaries, readmes`
    - `clients [x%]`: macos, windows and linux clients. note that this will implicitly release attached plugin bindata
    - `macos [x%]`: macos client
    - `windows [x%]`: windows client
    - `linux [x%]`: linux client
    - `backend [release branch]`: backend release (requires release branch)
    - `website`: website
    - `npm`: publish kite's npm package suite (kite-connector, kite-api, kite-installer)
    - `plugins`: publish all plugins
    - `vscode`: publish vs code
    - `atom`: publish atom
    - `binaries`: publically upload binaries
    - `readmes`: copy and push plugin readmes to kiteco/plugins

    NOTE: running a release will always release the *latest staged version* of the items to
    release, save for a few important exceptions:
    - the backend will release the backend deployment given by the *release branch*
    - plugins publish will publish the *latest master* (this is not ideal as differences can emerge
      between testing staging and release)
    - readmes will publish the *latest master readme of atom* (this is because atom is not a
      submodule, therefore to update we need to pull it separately)

    The release branch is outputted by the stage command and looks like `release_[timestamp]`. Make
    sure to double-check the branch name as the backend release will fail if given the wrong branch
    name.
    """
    if not allowed(bot):
        return

    # artifacts to build
    artifacts = [s.strip() for s in items.split(",") if s.strip() != "and"]

    # kwargs to pass to build command
    kwargs = {}
    for artifact in artifacts:
        artifact_type, *artifact_args = artifact.split()

        # aggregate artifact types
        if artifact_type == 'all':
            artifacts.extend(('binaries', 'readmes'))
            artifacts.extend(PLUGINS)
            artifacts.extend(plat + ' ' + ' '.join(artifact_args) for plat in CLIENTS)
        elif artifact_type == 'clients':
            artifacts.extend(plat + ' ' + ' '.join(artifact_args) for plat in CLIENTS)
        elif artifact_type == 'plugins':
            artifacts.extend(PLUGINS)

        # client platforms (with optional canary)
        elif artifact_type in CLIENTS:
            pct = 100
            if len(artifact_args) > 0:
                pct_str = artifact_args[-1].strip()
                if not pct_str.endswith('%'):
                    bot.reply("invalid percentage `{}` (must end in %)".format(pct))
                    return
                try:
                    pct = int(pct_str[:-1])
                except ValueError as e:
                    bot.reply("invalid percentage `{}` ({})".format(pct, e))
                    return

            kwargs[artifact_type] = pct

        # backend release branch
        elif artifact_type == 'backend':
            if len(artifact_args) == 0:
                bot.reply("backend release requires a branch")
                return
            # release branch is last artifact_args
            kwargs[artifact_type] = artifact_args[-1]

        # everything else
        elif artifact_type in PLUGINS or artifact_type in (
                'website', 'binaries', 'readmes', 'npm'):
            kwargs[artifact_type] = True

        else:
            bot.reply("invalid artifact type `{}`".format(artifact_type))
            return

    async with bot.require_lock(L_RELEASE):
        cmd_runner = build_commands.CommandRunner(bot)
        await cmd_runner.release(**kwargs)

cmd_regex = (
    r"^verify datadeps"
)

# pylint: disable=too-many-arguments
@respond_to(cmd_regex)
async def verify_datadeps(bot):
    """Verify kitelocal datadeps

    Usage: verify datadeps (FOR TESTING ONLY)
    """
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.verify_datadeps()

cmd_regex = (
    r"^upload sitemaps"
)
@respond_to(cmd_regex)
async def upload_sitemaps(bot):
    """Generate and upload a new sitemap for kite.com

    Usage: upload sitemaps
    Note: will first pull from latest master
    """
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.upload_sitemaps()

@respond_to(r"^hi")
async def dummy(bot):
    """Say hi"""
    bot.send("hi")

@respond_to(r"make me a sammich")
async def sammich(bot):
    """Test command permissions"""
    if not allowed(bot):
        return

    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.sammich()

@respond_to(r"^cleanup")
async def cleanup(bot):
    """Cleanup unused deployments"""
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.cleanup()

@respond_to(r"^changelog")
async def changelog(bot):
    """Show the changelog since last release"""
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.changelog()

@respond_to(r"^sleep for ([0-9]+)")
async def sleep(bot, seconds):
    """Sleep for some seconds

    Usage: sleep for [seconds]
    """
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.sleep(int(seconds))

@respond_to(r"thanks[\!1]*")
async def youre_welcome(bot):
    """Reply to a thanks message; testing for file uploads"""
    cmd_runner = build_commands.CommandRunner(bot)
    await cmd_runner.youre_welcome()

@respond_to("^hold lock ([a-zA-Z0-9_]+) for ([0-9]+)")
async def hold_lock(bot, lock, seconds):
    """Hold the given lock for some seconds

    Usage: hold [lock name] for [seconds]

    Mostly used for testing but can be useful if you want to lock down some commands temporarily
    """
    if not allowed(bot):
        return

    async with bot.require_lock(lock):
        cmd_runner = build_commands.CommandRunner(bot)
        await cmd_runner.hold_lock(lock, int(seconds))

@respond_to("^unlock ([a-zA-Z0-9_]+)")
async def release_lock(bot, lock):
    """Unlock a lock

    Usage: unlock [lock name]
    """
    if not allowed(bot):
        return

    bot.release(lock)

if __name__ == "__main__":
    # read debug arg
    parser = argparse.ArgumentParser()
    parser.add_argument("--debug", action="store_true", default=False)
    args = parser.parse_args()

    server.run(debug=args.debug)
