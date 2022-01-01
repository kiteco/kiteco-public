"""
This module contains functions used in the solness commands as well as utility functions
"""

import json
import os
import errno
import asyncio
import subprocess
import time
import glob
import shutil
import collections
import traceback
import glob
from datetime import datetime

from git import Repo, GitCommandError, InvalidGitRepositoryError
import git.exc

DEPLOYMENTS_CMD = "deployments"

class CommandRunner(object):
    """docstring for CommandRunner"""
    def __init__(self, bot):
        super(CommandRunner, self).__init__()
        # NOTE: the bot attribute is the slackbot interface
        self.bot = bot

        # this lets us know across various git functions if we're in the special
        # case of working with the release repo, which has various edge case treatment
        self.is_release_operation = False

        self.safe_mode = bool(os.environ.get("SAFE_MODE", False))
        if self.safe_mode:
            print("SAFE MODE ENABLED")

        self.cmd_env = os.environ.copy()

        # directories
        self.HOME_DIR = os.path.expanduser("~")
        self.ROOT_GIT_REPO = os.path.expanduser("~/kiteco")
        # self.GIT_REPO = os.path.expanduser("~/kiteco")
        self.BASE_PATH = os.path.expanduser("~/build_repos")

        self.build_paths()

    def build_paths(self):
        # this command sets all the subpaths dependant on the main repo paths
        if self.is_release_operation:
            cmd_prefix = "release"
        else:
            cmd_prefix = self.bot.sender

        self.SENDER= cmd_prefix

        base_path = os.path.join(self.BASE_PATH, "{}/".format(cmd_prefix))
        repo_path = os.path.join(base_path, "src", "github.com", "kiteco", "kiteco")

        #  the release path is the place to put successfully finished plugin builds so the release commands always have
        #  a good version ready to deploy
        release_path = os.path.join(self.BASE_PATH, "release/")
        release_repo_path = os.path.join(release_path, "src", "github.com", "kiteco", "kiteco")
        self.RELEASE_PLUGINS_DIR = os.path.join(release_repo_path, "plugins")
        self.PLUGIN_BIN_PATHS = {
            "Sublime Text 3": "sublimetext3-plugin/st_package_builder/target/st3/Kite.sublime-package",
        }

        self.cmd_env['GOPATH'] = base_path
        self.cmd_env['KITECO'] = repo_path
        self.cmd_env['SAFE_MODE'] = "true" if self.safe_mode else ""
        self.GIT_REPO = repo_path
        self.PLUGIN_REPO = os.path.join(release_path, "src", "github.com", "kiteco", "plugins")

        self.SCRIPTS = os.path.join(self.GIT_REPO, "scripts")
        self.SCRIPT_OUTPUT = os.path.expanduser("~/tmp/")
        self.WINDOWS_BUILD_DIR = os.path.join(self.SCRIPTS, "staged-windows-build")
        self.LINUX_BUILD_DIR = os.path.join(self.SCRIPTS, "staged-linux-build")
        self.MACOS_DMG_DIR = os.path.join(self.GIT_REPO, "osx")
        self.PLUGINS_DIR = os.path.join(self.GIT_REPO, "kite-go", "client", "internal", "plugins")
        self.DATA_OUTPUT = "/var/kite/kitelocal-data"

        self.PLUGIN_DIRS = {
            "Sublime Text 3": os.path.join(self.PLUGINS_DIR, "sublimetext", "sublime3"),
            "Vim/Neovim": os.path.join(self.PLUGINS_DIR, "vim"),
        }

    async def get_repo(self):

        # ensure path exists, if it doesn't, clone our root repo to it
        try:
            return Repo(self.GIT_REPO)
        except git.exc.NoSuchPathError:
            # path doesnt exist, clone our existing repo
            base_repo = Repo(self.ROOT_GIT_REPO)
            new_repo = Repo.clone_from(url=base_repo.remotes.origin.url, to_path=self.GIT_REPO)
            await self.run_subprocess(["git", "lfs", "install"],
                cwd=self.GIT_REPO,
                success_mesg="initialized git lfs for new repo")
            await self.run_subprocess(["git", "submodule", "update", "--init"],
                cwd=self.GIT_REPO,
                success_mesg="initialized submodules for new repo")
            await self.run_subprocess(["make", "install-deps"],
                cwd=self.GIT_REPO,
                success_mesg="make install-deps for new repo")
            return new_repo

    ## Staging commands ##

    async def stage_release(self, **kwargs):
        """Stage release

        The various kwargs determine which steps to include/omit
        """

        # set to use release build_repo, and rebuild git paths accordingly
        self.is_release_operation = True
        print('Setting repo to `release` build_path')
        self.build_paths()

        # task list
        tasks = []

        ref = kwargs['ref']
        self.bot.send("Grabbing latest {}".format(ref))
        await self.git_reset_latest(ref)

        # macos client
        if kwargs["client"] in {"all", "macos"}:
            print("Will stage macos client")
            tasks.append(self.stage_macos_client())

        if len(tasks) > 0:
            print("Waiting for macos")
            await self.gather_and_await(tasks)
            tasks = []

        if len(tasks) > 0:
            print("Waiting for windows")
            await self.gather_and_await(tasks)
            tasks = []

        if len(tasks) > 0:
            print("Waiting for linux")
            await self.gather_and_await(tasks)
            tasks = []

        # backend
        if kwargs["backend"]:
            print("Will stage backend")
            tasks.append(self.stage_backend(ref))

        # plugin upload
        if kwargs["binaries"]:
            print("Will upload plugin binaries")
            tasks.append(self.upload_plugin_binaries())

        # release channel ID
        release_ch_id = "C0M76GA13"
        # message changelog to release
        # if not kwargs["quiet"]:
            # tasks.append(self.git_release_diff(channel=release_ch_id))

        # gather and wait
        print("Awaiting build tasks")
        await self.gather_and_await(tasks)
        tasks = []

        # do website after backend
        if kwargs["website"]:
            print("Will stage website")
            tasks.append(self.stage_website())

        # gather and wait
        print("Awaiting 2nd queue of build tasks")
        await self.gather_and_await(tasks)
        tasks = []

        # release is ready
        print("Release is ready")
        if not kwargs["quiet"]:
            self.bot.send("Release is ready! @here")
            self.bot.send(
                "New release is on staging, please test your changes @here",
                channel=release_ch_id)

    async def build_binaries(self):
        """Build binaries, create pull request, wait for CI to pass, then either merge or abort

        NOTE: this command involves git operations - use with caution

        NOTE: if needed we should let the command indicate which binaries to build
        """

        # do a master pull on the submodules
        await self.git_pull_plugins()

        # verify datadeps early and fail if they're no good
        await self.verify_datadeps()

        # dict of plugin to slack user to ping when failed
        to_ping = {
            "Sublime Text 3": "juan",
            "Vim/Neovim": "joachim",
        }

        self.bot.send("Starting plugin bindata build...")
        # run go generate on each of the plugins
        build_cmd = ["go", "generate"]
        success = "{} plugin built successfully."
        error = "Error while building {}"

        tasks = []
        for name, directory in list(self.PLUGIN_DIRS.items()):
            print("build_binaries PLUGIN_DIRS directory: `{}`".format(directory))
            kwargs = {
                "cwd": directory,
                "error_mesg": error.format(name),
                "success_mesg": success.format(name),
                "error_ping": to_ping.get(name, "here")
            }
            tasks.append(self.run_subprocess(build_cmd, **kwargs))

        # wait for bindata builds to finish
        try:
            print("Awaiting bindata builds")
            await self.gather_and_await(tasks)
        except GatherError as e:
            # send the error to slack but don't stop
            print("bindata build error", e, sep=" => ")
            self.bot.send(str(e))

        for name, directory in list(self.PLUGIN_DIRS.items()):
            # move any successful plugin builds to our release repo in case we decide to release them later
            if name in self.PLUGIN_BIN_PATHS:
                relative_binary_path = self.PLUGIN_BIN_PATHS[name]
                local_repo_binary_path = glob.glob(os.path.join(self.GIT_REPO, "plugins", relative_binary_path))
                if len(local_repo_binary_path) == 0:
                    # no files matched
                    self.bot.send("No files matched glob pattern {}".format(os.path.join(self.GIT_REPO, "plugins", relative_binary_path)))
                    continue
                local_repo_binary_path = local_repo_binary_path[0]
                release_repo_binary_path = os.path.join(self.RELEASE_PLUGINS_DIR, relative_binary_path)
                release_repo_binary_dir = "/".join(release_repo_binary_path.split("/")[:-1])
                await self.run_subprocess(
                    ["mkdir", "-p", release_repo_binary_dir],
                    success_mesg="Ensure release repo binary dir",
                    error_mesg="Failed to create release repo binary dir")
                # actually only perform the movement if we're not already operating in the release repo
                # otherwise, the below will be redundant and lead to a cp error
                if not self.is_release_operation:
                    move_cmd = ["cp", local_repo_binary_path, release_repo_binary_path]
                    await self.run_subprocess(
                        move_cmd,
                        success_mesg="Copied {} plugin to release repo to prep for later release".format(name),
                        error_mesg="Failed to copy {} plugin".format(name))

        # add updated plugin submodules
        await self.git_add_plugins()

        # try to create and merge bindata PR
        await self.git_merge_bindata()

    async def build_test(self, **kwargs):
        """Build a test build"""
        human_plugin_name = {
            "Sublime Text 3": "sublime",
            "Vim/Neovim": "vim",
        }

        self.bot.send("Starting test build...")
        print("Starting test build")
        print("GIT_REPO: `{}`".format(self.GIT_REPO))
        print("SCRIPTS: `{}`".format(self.SCRIPTS))
        # task list
        tasks = []

        r = await self.get_repo()
        # use latest master of repo + submodules
        if not kwargs["branch"]:
            # get latest master
            self.bot.send("Grabbing latest master")
            print("Grabbing latest master")
            await self.git_reset_latest()

            # pull submodules
            for submodule in r.submodules:
                print("pulling submodule plugin with path `{}`".format(submodule.path))
                if submodule.path.startswith("plugins/"):
                    # get the git command interface for the submodule
                    subgit = submodule.module().git
                    # pull changes from master
                    subgit.checkout("master")
                    subgit.pull()
            test_branch = "master"
        # use current repo state
        elif kwargs["branch"] == "current":
            self.bot.send("Grabbing latest for branch `{}`".format(test_branch))
            print("Grabbing latest for branch `{}`".format(test_branch))
            test_branch = str(await self.git_current_branch())
        # use branch
        else:
            test_branch = kwargs["branch"]
            print("resetting for latest to branch `{}`".format(test_branch))
            await self.git_reset_latest(test_branch)

        # verify datadeps early and fail if they're no good
        await self.verify_datadeps()

        self.bot.send("Building on branch `{}`".format(test_branch))

        if kwargs["binaries"]:
            # build binaries
            self.bot.send("Starting binaries build...")
            print("GIT_REPO: `{}`".format(self.GIT_REPO))
            print("SCRIPTS: `{}`".format(self.SCRIPTS))
            # run go generate on each of the plugins
            build_cmd = ["go", "generate"]
            success = "{} plugin built successfully."
            error = "Error while building {}"
            for name, directory in list(self.PLUGIN_DIRS.items()):
                # filter if only_plugins specified
                print("plugin name, directory: `{}`, `{}`".format(name, directory))
                if kwargs["only_plugins"] and len(kwargs["only_plugins"]) > 0:
                    if human_plugin_name[name] in kwargs["only_plugins"]:
                        tasks.append(
                            self.run_subprocess(
                                build_cmd, cwd=directory,
                                error_mesg=error.format(name), success_mesg=success.format(name)))
                else:
                    tasks.append(
                        self.run_subprocess(
                            build_cmd, cwd=directory,
                            error_mesg=error.format(name), success_mesg=success.format(name)))

        # gather and wait
        await self.gather_and_await(tasks)
        tasks = []

        # macos client
        if kwargs["build"] in {"clients", "macos"}:
            tasks.append(self.stage_macos_client(test=True))

        # gather and wait
        await self.gather_and_await(tasks)
        tasks = []

        # build is ready
        self.bot.reply("Test build complete")

    async def stage_macos_client(self, test=False):
        """Build and stage the MacOS client"""
        self.bot.send("Starting MacOS build...")
        print("GIT_REPO: `{}`".format(self.GIT_REPO))
        print("SCRIPTS: `{}`".format(self.SCRIPTS))
        print("MACOS_DMG_DIR: `{}`".format(self.MACOS_DMG_DIR))

        # on the first build for this xcode project it will silently hang
        #  on the solness box unless the project is opened once in xcode first
        #  this script intended to automate that effort but it seems not to work -nj

        # await self.run_subprocess(["./init_xcode_project.sh"],
        #     cwd=self.SCRIPTS,
        #     success_mesg="init xcode for build")

        cmd = ["./stage_new_build.sh", "--ignore-git"]
        # use no upload and custom version for testing
        if test:
            test_version = "99999999"
            cmd.extend(("--no-upload", "--version", test_version))

        print("Awaiting MacOS client build")
        await self.run_subprocess(cmd, success_mesg="MacOS client build complete. Find the client on s3 in the `kite-downloads` folder")

        # upload to slack if test
        # NOTE: should try to get this from script output in the future
        if test:
            dmg_output = os.path.join(self.MACOS_DMG_DIR, "Kite-{}.dmg".format(test_version))
            print("dmg output: `{}`".format(dmg_output))
            self.try_uploading(dmg_output)

    async def stage_backend(self, branch_name):
        """Build and stage the backend"""
        await self.require_deployment_tool(rebuild=True)

        self.bot.send("Deploying %s..." % branch_name)
        print("Deploying ", branch_name)
        await self.run_subprocess(
            ["./"+DEPLOYMENTS_CMD, "deployregions", branch_name],
            logfile_name="deploy-server",
            error_mesg="Deployment error!",
            success_mesg=("Deployed release %s" % branch_name))

        # check on the deployment every 5 minutes
        while not await self.check_deployment(branch_name):
            await asyncio.sleep(300)

        self.bot.send("Staging %s..." % branch_name)
        print("Staging ", branch_name)
        await self.run_subprocess(
            ["./"+DEPLOYMENTS_CMD, "switchregions", branch_name, "staging"],
            logfile_name="albswitch",
            error_mesg="Load-balancing error!",
            success_mesg="%s now available on staging.kite.com" % branch_name)

    async def stage_website(self):
        """Build and stage the website

        NOTE: must be run from the repo root directory"""

        self.bot.send("Staging website...")
        print("Staging website")

        #await self.run_subprocess(
        #    [os.path.join(self.SCRIPTS, "./webapp_precache_completions.sh")],
        #    cwd=self.GIT_REPO,
        #    success_mesg="Sandbox completions precached!",
        #    error_mesg="Error caching sandbox completions"
        #)
        await self.run_subprocess(
            [os.path.join(self.SCRIPTS, "./deploy_webapp.sh")],
            cwd=self.GIT_REPO,
            success_mesg="Website is now available on ga-staging.kite.com")

    async def upload_plugin_binaries(self, public=False):
        """Upload plugin binaries to S3

        If public is true, make the binaries publicly available"""
        message = "Plugin binaries uploaded"
        pub = ""
        if public:
            message = "Plugin binaries now publicly available"
            pub = "pub"

        print("Uploading plugin binaries")
        await self.run_subprocess(
            [os.path.join(self.SCRIPTS, "./upload_plugin_binaries.sh"), pub],
            success_mesg=message)

    async def verify_datadeps(self, **kwargs):
        """Build and upload the dataset"""
        self.bot.send("Starting kitelocal datadeps verification...")
        print("Awaiting kitelocal datadeps verification")
        await self.run_subprocess([os.path.join(self.SCRIPTS, "./verify_datadeps.sh")],
            success_mesg="Kitelocal datadeps verified",
            error_mesg="Kitelocal datadeps verification failed!")

    async def upload_sitemaps(self, **kwargs):
        """Generate and upload sitemaps"""
        self.bot.send("Starting sitemap generation and uploading...")
        print("Awaiting sitemap generation and uploading...")
        self.is_release_operation = True
        self.build_paths()
        # get latest master first
        await self.git_reset_latest(update_submodules=False)
        await self.run_subprocess([os.path.join(self.SCRIPTS, "./upload_sitemaps.sh")],
            success_mesg="Sitemap generated and uploaded",
            error_mesg="Sitemap generation and uploading failed!"
        )


    ## Release commands ##

    async def do_sleep_test(self):
        r = await self.get_repo()
        self.bot.send("repo path: {}  commit: {}".format(r.working_dir, r.commit().hexsha))
        self.bot.send("sleeping for 60s to test")
        await asyncio.sleep(60)
        self.bot.send("sleep finished")
        self.bot.send("repo path: {}  commit: {}".format(r.working_dir, r.commit().hexsha))
        return True

    async def release(self, **kwargs):
        """Release staged releases - will release things staged in the `release` build_repo

        NOTE: as mentioned in the run.release documentation, the version of the things to release is
        inconsistent across the different things to release - ideally, this should be updated so that
        it makes no git state changes (which the plugin publishes currently do) so that everything
        except for the backend will always release what is currently staged, i.e. the current state of
        the files on the machine.
        """

        self.is_release_operation = True
        self.build_paths()  # rebuild internal git paths for our edge case release operation
        # before starting, get the current release name for the changelog
        # prev_release = await self.get_release_name()

        # task list
        tasks = []
        print("Start release script")

        r = await self.get_repo()
        r.git.checkout("master")

        if self.safe_mode:
            # make release a NOOP in safe mode, just sleep for 60s and display some debugging stats instead
            self.bot.send("performing SAFE MODE release, expect many verbose NOOPs")

        # TODO: if kwargs["backend"] is given, it should check the release via deployments tool;
        # for now, just make sure the branch starts with "release_"
        backend = kwargs.get("backend")
        if backend:
            if not backend.startswith("release_"):
                print("INVALID: backend release name illegal: ", backend)
                self.bot.reply("invalid backend release `{}`".format(backend))
                return
            print("Will release backend")
            tasks.append(self.release_backend(backend))

        # clients
        if kwargs.get("macos") is not None:
            print("Will release macOS client")
            tasks.append(self.release_macos_client(kwargs['macos']))
        if kwargs.get("windows") is not None:
            print("Will release windows client")
            tasks.append(self.release_windows_client(kwargs['windows']))
        if kwargs.get("linux") is not None:
            print("Will release linux client")
            tasks.append(self.release_linux_client(kwargs['linux']))

        # npm pkg publish
        if kwargs.get("npm"):
            print("Will publish kite npm package suite")
            tasks.append(self.publish_kite_npm_pkgs())

        # plugin publish
        if kwargs.get("atom"):
            print("Will publish atom plugin")
            tasks.append(self.publish_atom_plugin())
        if kwargs.get("vscode"):
            print("Will publish vscode plugin")
            tasks.append(self.publish_vscode_plugin())

        # public plugins upload
        if kwargs.get("binaries"):
            print("Will publish plugin binaries")
            tasks.append(self.upload_plugin_binaries(public=True))

        # gather and wait
        print("Awaiting first release publishing task set")
        await self.gather_and_await(tasks)
        tasks = []

        # deploy website after backend
        if kwargs.get("website"):
            print("Will deploy website")
            tasks.append(self.release_website())

        # gather and wait
        print("Awaiting second release publishing task set")
        await self.gather_and_await(tasks)
        tasks = []

        # post-release stuff

        # send changelog to #engineering
        # engineering_ch_id = "C03NKKYL4"
        # tasks.append(self.git_release_diff(release_name=prev_release, channel=engineering_ch_id))

        # cleanup old deployments
        tasks.append(self.cleanup())

        # gather and wait
        print("Awaiting cleanup")
        await self.gather_and_await(tasks)
        tasks = []

        # update readmes
        if kwargs.get("readmes"):
            self.update_readmes()

        # release is ready
        print("Release is live")
        self.bot.reply("Release is live!")

    async def release_backend(self, release_name):
        """Release the staged backend"""
        self.bot.send("Releasing %s..." % release_name)
        print("Releasing backend ", release_name)
        await self.require_deployment_tool()
        await self.run_subprocess(
            ["./"+DEPLOYMENTS_CMD, "switchregions", release_name, "prod"],
            logfile_name="albswitch",
            error_mesg="Load-balancing error!",
            success_mesg="%s now available on alpha.kite.com" % release_name)

    async def release_macos_client(self, pct):
        """Release the staged MacOS client"""
        self.bot.send("Releasing staged MacOS client at {}%".format(pct))
        print("Releasing staged MacOS client")
        await self.run_subprocess(
            [os.path.join(self.SCRIPTS, "release_staged_client.sh")],
            logfile_name="release-client",
            error_mesg="Client release error!",
            success_mesg="Released client",
            env={'PLATFORM': 'mac', 'CANARY_PERCENTAGE': '{}'.format(pct)},
        )

    async def release_windows_client(self, pct):
        """Release the staged Windows client"""

        self.bot.send("Releasing staged Windows client at {}%".format(pct))
        print("Releasing staged Windows client")
        await self.run_subprocess(
            [os.path.join(self.SCRIPTS, "release_staged_client.sh"), self.SENDER, str(pct)],
            logfile_name="release-windows",
            error_mesg="Windows release error!",
            success_mesg="Released windows client",
            env={'PLATFORM': 'windows', 'CANARY_PERCENTAGE': '{}'.format(pct)},
        )

    async def release_linux_client(self, pct):
        """Release the staged Linux client"""

        self.bot.send("Releasing staged Linux client at {}%".format(pct))
        print("Releasing staged Linux client")
        await self.run_subprocess(
            [os.path.join(self.SCRIPTS, "release_staged_client.sh"), str(pct)],
            logfile_name="release-linux",
            error_mesg="Linux release error!",
            success_mesg="Released linux client",
            env={'PLATFORM': 'linux', 'CANARY_PERCENTAGE': '{}'.format(pct)},
        )

    async def release_website(self, **kwargs):
        """Release staged website

        NOTE: most be run from the repo root directory
        """

        print("Releasing staged website")
        repo = self.GIT_REPO
        scripts_dir = self.SCRIPTS
        if kwargs is not None:
            if 'repo' in kwargs:
                repo = kwargs['repo']
            if 'scripts_dir' in kwargs:
                scripts_dir = kwargs['scripts_dir']

        #await self.run_subprocess(
        #    [os.path.join(self.SCRIPTS, "./webapp_precache_completions.sh")],
        #    cwd=self.GIT_REPO,
        #    success_mesg="Sandbox completions precached!",
        #    error_mesg="Error caching sandbox completions"
        #)
        await self.run_subprocess(
            [os.path.join(scripts_dir, "./deploy_webapp.sh"), "prod", repo],
            cwd=repo,
            success_mesg="Website is now live")

    def setup_repo_git(self, repo_name):
        """Sets up git to a reset master for the given repo_name, and returns the output of `git pull`"""
        # pull master and check for changes
        repo = Repo(os.path.join(self.HOME_DIR, repo_name))
        git = repo.git
        git.reset(hard=True)
        git.checkout("master")
        pull_output = git.pull("origin", "master")
        return repo, pull_output

    async def publish_kite_npm_pkgs(self):
        """Publish Kite's npm package suite"""
        # we use the git repo names here, not the npm package names -
        # which are referenced in the respective package.json's
        await self.publish_kite_npm("kite-connect-js")
        await self.publish_kite_npm("kite-installer")
        await self.publish_kite_npm("kite-api-js")

    async def publish_kite_npm(self, repo_name):
        """Publish the kite npm package indicated by repo_name"""
        repo, pull_output = self.setup_repo_git(repo_name)

        # if the pull had changes, publish
        if "Already up to date." not in pull_output:
            if not self.safe_mode:
                pkg_dir = os.path.join(self.HOME_DIR, repo_name)
                self.bot.send("Publishing {} pkg".format(repo_name))
                print("Publishing {} pkg".format(repo_name))
                await self.run_subprocess(
                    [os.path.join(self.ROOT_GIT_REPO, "scripts", "publish_npm_pkg.sh")],
                    cwd=pkg_dir,
                    success_mesg="Published {} pkg".format(repo_name)
                    )
                # if successful, push up changes made to package.json
                # publish_npm_pkg.sh will have already made a version commit
                print("pushing {} commit".format(repo_name))
                repo.git.push()
            else:
                self.bot.send("no {} publish, we're in safe mode".format(repo_name))
        else:
            print("No changes to master for {}, skip publish".format(repo_name))
            self.bot.send("No changes to master for {}, skip publish".format(repo_name))

    async def publish_atom_plugin(self):
        """Publish the atom plugin"""
        repo, pull_output = self.setup_repo_git("atom-plugin")
        git = repo.git

        # if the pull had changes, publish
        if "Already up to date." not in pull_output:
            if not self.safe_mode:
                self.bot.send("Publishing atom plugin")
                print("Publishing atom plugin")
                await self.run_subprocess(
                    ["npm", "run", "prepublishOnly"],
                    cwd=os.path.join(self.HOME_DIR, "atom-plugin"),
                    success_mesg="Bundled atom plugin"
                )

                try:
                    git.add(A=True)
                except Exception as ex:
                    self.bot.send("Error git adding bundled Atom js file")
                try:
                    git.commit(m="new publish bundle")
                except GitCommandError:
                    # no changes, don't push
                    self.bot.send("Error committing generated atom js bundle")

                await self.run_subprocess(
                    ["apm", "publish", "minor"],
                    cwd=os.path.join(self.HOME_DIR, "atom-plugin"),
                    success_mesg="Published atom plugin"
                    )
            else:
                self.bot.send("no atom publish, we're in safe mode")
        else:
            print("No changes to master for atom, skip publish")
            self.bot.send("No changes to master for atom, skip publish")

    async def publish_vscode_plugin(self):
        """Publish the vs code plugin"""

        # check for and read token from env var
        try:
            token = os.environ["VSCODE_PUBLISH_TOKEN"]
        except KeyError:
            print("Cannot publish vscode with empty token")
            self.bot.send("Cannot publish vscode with empty token")
            return

        repo, pull_output = self.setup_repo_git("vscode-plugin")
        git = repo.git

        # if the pull had changes, publish
        if "Already up to date." not in pull_output:
            if not self.safe_mode:
                self.bot.send("Publishing vscode plugin")
                print("Publishing vscode plugin")

                await self.run_subprocess(
                    ["npm", "run", "cleanup"],
                    cwd=os.path.join(self.HOME_DIR, "vscode-plugin"),
                    success_mesg="Cleaned up"
                )

                await self.run_subprocess(
                    ["npm", "install"],
                    cwd=os.path.join(self.HOME_DIR, "vscode-plugin"),
                    success_mesg="Finished `npm install`"
                )

                await self.run_subprocess(
                    ["vsce", "publish", "minor", "-p", token],
                    cwd=os.path.join(self.HOME_DIR, "vscode-plugin"),
                    success_mesg="Created new minor version"
                )

                # add new publish commit
                # the below is erroring out - the commit specifically... (why??)
                # Hypothesis: the publish command automatically makes a version commit and we just need to push
                try:
                    git.push()
                except GitCommandError as err:
                    print("Error raised trying to push vscode publish commit: `{}`".format(err))
                    self.bot.send("Error raised trying to push vscode publish commit: `{}`".format(err))
            else:
                self.bot.send("skipping vscode publish, safe mode enabled")
        else:
            print("No changes to master for vscode, skip publish")
            self.bot.send("No changes to master for vscode, skip publish")


    def update_readmes(self):
        """Update plugins READMEs"""

        self.bot.send("Updating plugin READMEs")

        # get readme directories from plugin submodules
        readme_dirs = glob.glob(os.path.join(self.GIT_REPO, "plugins", "*", "README.md"))
        images_dirs = [os.path.join(os.path.dirname(d), "docs") for d in readme_dirs]

        # remove "kiteco/" and "-plugin" substring for the respective directory in the plugins repo
        plugin_dirs = [d.replace("kiteco/kiteco/", "kiteco/").replace("-plugin", "") for d in readme_dirs]

        # atom is not a submodule so pull master for it and add separately
        atom_repo = os.path.join(self.HOME_DIR, "atom-plugin")
        git = Repo(atom_repo).git
        git.checkout("master")
        git.pull()
        readme_dirs.append(os.path.join(atom_repo, "README.md"))
        images_dirs.append(os.path.join(atom_repo, "docs"))
        plugin_dirs.append(os.path.join(self.PLUGIN_REPO, "atom", "README.md"))

        # copy readmes
        plugin_images_dirs = []
        for i, d in enumerate(plugin_dirs):
            # create the plugin dirs if not already there
            if not os.path.exists(os.path.dirname(d)):
                os.makedirs(os.path.dirname(d))
            # copy readme
            shutil.copyfile(readme_dirs[i], d)
            # copy images
            if os.path.exists(images_dirs[i]):
                plugin_images_dir = os.path.join(os.path.dirname(d), "docs")
                # if destination exists, remove first
                if os.path.exists(plugin_images_dir):
                    shutil.rmtree(plugin_images_dir)

                shutil.copytree(images_dirs[i], plugin_images_dir)
                plugin_images_dirs.append(plugin_images_dir)

        # git commit, rebase master, and push
        git = Repo(self.PLUGIN_REPO).git
        for d in plugin_dirs:
            try:
                git.add(d)
            except Exception as ex:
                self.bot.send("Error git adding README for `{}`: \n```\n{}\n```".format(d, exception_str(ex)))
        for d in plugin_images_dirs:
            try:
                git.add(d)
            except Exception as ex:
                self.bot.send("Error git adding README for `{}`: \n```\n{}\n```".format(d, exception_str(ex)))
        try:
            git.commit(m="update READMEs")
        except GitCommandError:
            # no changes, don't push
            self.bot.send("No README changes")
            pass
        else:
            git.pull("--rebase", "origin", "master")
            if not self.safe_mode:
                git.push()
                self.bot.send("README changes pushed")


    ## Other commands ##

    async def sammich(self):
        """Test command permissions"""
        self.bot.reply(u"\U0001F354")

    async def check_deployment(self, release_branch):
        """
        Use the deployment tool to check if the new deployment tagged with release_branch is ready
        """

        self.bot.send("Checking deploy %s..." % release_branch)
        print("Checking deploy ", release_branch)
        await self.require_deployment_tool()
        proc = await asyncio.create_subprocess_exec(
            "./"+DEPLOYMENTS_CMD, "describeregions", release_branch, "json",
            stdout=subprocess.PIPE, cwd=self.SCRIPTS)
        so, _ = await proc.communicate()
        await proc.wait()
        # convert binary string output to normal string
        desc_out = so.decode("utf8")

        # read in json output to dict for each region
        regions = {}
        for line in desc_out.split("\n"):
            # Skip empty lines
            if not line:
                continue

            # Skip none-types or empty regions
            region = json.loads(line)
            if not region or len(region) == 0:
                continue

            # read region name from first deployment
            region_name = region[0]["Region"]
            regions[region_name] = region

        # if we got responses from fewer than 3 regions, we're not ready
        if len(regions) < 3:
            notReadyMsg = "{} not started in all regions yet ({} out of 3 found)".format(release_branch, len(regions))
            print(notReadyMsg)
            self.bot.send(notReadyMsg)
            return False

        # check readiness
        not_ready = collections.Counter()
        ready = True
        for region_name, region in regions.items():
            for deployment in region:
                if deployment["Status"] != "ready":
                    not_ready[deployment["Status"]] += 1
                    ready = False

        if ready:
            print("Release {} is ready in all regions".format(release_branch))
            self.bot.send("Looks like %s is ready in all regions!" % release_branch)
        else:
            notReadyMsg = "{} is not ready - {} servers down, {} servers loading".format(
                    release_branch, not_ready["down"], not_ready["loading"])
            print(notReadyMsg)
            self.bot.send(notReadyMsg)

        return ready

    # NOTE: currently not functional but left in for reference
    def show_statuses(self):
        """Show statuses of current commands"""

        text = "Current command statuses:\n"
        tasks = asyncio.Task.all_tasks()
        if not tasks:
            text = "There are no commands currently being run"
        for task in tasks:
            # tasks name
            task_name = task._coro.__name__ # pylint: disable=protected-access
            status = "running"
            if task.done():
                # check for cancellation
                if task.cancelled():
                    status = "cancelled"
                # check for exception
                elif isinstance(task.result(), Exception):
                    status = "errored"
                else:
                    status = "done"
            text += "{}: {}\n".format(task_name, status)

        # remove trailing newlines
        text = text.strip()

        self.bot.send(text)

    async def cleanup(self):
        """Run the cleanup command to clean up unused regions"""
        if self.safe_mode:
            self.bot.send("safe mode enabled, skipping cleanup")
            return

        self.bot.send("Cleaning up old deployments ...")
        print("Cleaning up old deployments")
        await self.require_deployment_tool()
        await self.run_subprocess(
            ["./"+DEPLOYMENTS_CMD, "cleanupregions"],
            logfile_name="cleanupregions",
            error_mesg="Error cleaning regions!",
            success_mesg="Cleanupregions completed")

    async def changelog(self):
        """Show changelog"""
        await self.git_release_diff()

    async def sleep(self, seconds):
        """Sleep asynchronously for some seconds

        Used for testing async functionality
        """
        self.bot.send("Sleeping for {} seconds".format(seconds))
        await asyncio.sleep(seconds)
        self.bot.send("I'm awake!")

    async def youre_welcome(self):
        """Upload a "you're welcome" image"""
        self.bot.upload("yw.jpg", "yw.jpg", "you're welcome")

    async def hold_lock(self, lock, seconds):
        """Acquire the given lock for some seconds"""
        self.bot.send('Acquired lock "{}", holding it for {} seconds...'.format(lock, seconds))
        await asyncio.sleep(seconds)
        self.bot.send('Released lock "{}"'.format(lock))

    ## Utility functions ##

    # git operations
    # NOTE: the recommendation is to use the Repo.git object to use the git CLI for most operations, as
    # it is much clearer and easier to maintain than using gitpython"s abstractions for git objects.

    async def git_reset_latest(self, ref="master", update_submodules=True):
        """Get the latest version of the branch, and update submodules."""

        print("resetting git repo to latest")
        r = await self.get_repo()
        git = r.git
        git.reset(hard=True)
        git.fetch("origin", tags=True)
        git.checkout(ref)
        if update_submodules:
            git.submodule("update", "--force", "--recursive")
        print("git successfully reset to latest {}".format(ref))

    async def git_new_release_branch(self):
        """Creates a new release branch at the current state, pushes, and returns branch name"""
        r = await self.get_repo()
        git = r.git
        ts = datetime.now().strftime("%Y%m%dT%H%M%S")
        branch_name = "release_{}".format(ts)
        git.checkout("-b", branch_name)
        if not self.safe_mode:
            git.push("origin", branch_name)
        return branch_name

    async def git_current_branch(self):
        """Get the current branch for this repo"""
        r = await self.get_repo()
        return r.active_branch

    async def git_pull_plugins(self):
        """ For each of the submodules under plugins/, pull to get latest master.
        """
        await self.run_subprocess(
            ["git", "submodule", "update", "--init"],
            cwd=self.GIT_REPO,
            success_mesg="initialized submodules for new repo")
        r = await self.get_repo()
        for submodule in r.submodules:
            if not submodule.path.startswith("plugins/"):
                continue

            print("pulling plugin submodule with path: `{}`".format(submodule.path))
            await self.run_subprocess(
                ["git", "submodule", "update", "--remote", "--force", submodule.path],
                cwd=self.GIT_REPO,
                success_mesg="updated plugin submodules")

    async def git_add_plugins(self):
        """ For each of the submodules under plugins/, add the changed commit to the main repo.
        """
        r = await self.get_repo()
        for submodule in r.submodules:
            if not submodule.path.startswith("plugins/"):
                continue

            # add commit to main repo
            r.git.add(submodule.path)

    async def git_merge_bindata(self):
        """Creates a PR for bindata, waits for CI, and merges or aborts

        Reports progress through the message slackbot interface
        """
        repo = await self.get_repo()
        git = repo.git

        # create a new branch for plugins
        ts = datetime.now().strftime("%Y%m%dT%H%M%S")
        branch_name = "bindata_build_{}".format(ts)
        self.bot.send("Creating bindata build PR {}".format(branch_name))
        print("creating bindata build PR")
        git.checkout("-b", branch_name)
        # add bindata
        print("adding bindata at repo.working_dir: `{}`".format(repo.working_dir))
        git.add(os.path.join(repo.working_dir, "**", "bindata.go"))
        # commit and push
        print("committing bindata")
        git.commit("-m", "bindata build {}".format(ts))
        if self.safe_mode:
            self.bot.send("safe mode enabled, aborting before pushing or PR")
            self.bot.send("bindata build available locally on branch {}".format(branch_name))
            return

        print("pushing bindata")
        git.push("origin", branch_name)
        # create pull request
        pr_message = branch_name
        print("creating bindata pull request")
        proc = subprocess.run(
            ["hub", "pull-request", "-m", pr_message], cwd=repo.working_dir, stdout=subprocess.PIPE, encoding="utf8")
        pr_link = proc.stdout.strip()
        self.bot.send("Created PR for {} at {}".format(branch_name, pr_link))
        print("Created Bindata PR")
        print("CI status: ")

        # wait for CI to finish
        ci_status = "pending"
        while ci_status in {"pending", "no status"}:
            proc = subprocess.run(["hub", "ci-status"], cwd=repo.working_dir, stdout=subprocess.PIPE, encoding="utf8")
            ci_status = proc.stdout.strip()
            print(ci_status)
            time.sleep(30)

        if ci_status == "success":
            # get latest master
            git.checkout("master")
            git.pull("origin", "master")
            # merge PR
            print("Attempting to merge pr: `{}`".format(pr_link))
            proc = subprocess.run(["hub", "merge", pr_link], cwd=repo.working_dir, stdout=subprocess.PIPE, encoding="utf8")
            output = proc.stdout.strip()
            print(output)

            # # push master
            if not self.safe_mode:
                 git.push("origin", "master")

            self.bot.send("CI passed, merged PR")
        else:
            raise Exception("CI failed - aborting")

    async def git_release_diff(self, release_name=None, channel=None):
        """Display a changelog since last release"""

        # if not given, get current release
        if release_name is None:
            release_name = await self.get_release_name()
        r = await self.get_repo()
        git = r.git
        git.fetch()
        changes = git.log(
            "origin/{}..origin/master".format(release_name),
            pretty="format:[%ad] %an: %s",
            date="short")
        # wrap in code block with title
        changes = "Changes since {}:\n```\n{}\n```".format(release_name, changes)

        # send to channel where command came from
        if not channel:
            self.bot.send(changes)
        # send to specific channel
        else:
            self.bot.send(changes, channel=channel)

    async def get_release_name(self, target="prod"):
        """Get the name of the current release on target"""
        self.bot.send("Getting release name on {}".format(target))
        await self.require_deployment_tool()

        # NOTE: we call subprocess_exec directly instead of using run_subprocess because we want to
        # retrieve and use the output
        proc = await asyncio.create_subprocess_exec(
            "./"+DEPLOYMENTS_CMD, "list",
            stdout=subprocess.PIPE, cwd=self.SCRIPTS)
        so, _ = await proc.communicate()
        await proc.wait()
        # convert binary string output to normal string
        list_out = so.decode("utf8")

        # find the current alpha release names
        #
        # rough format for the output:
        #
        # region: <name>
        #   prod: <release name>
        #   staging: <release name>
        # available:
        #   <release name>
        #   ...
        # ...
        #
        # all regions should have the same release for prod/staging
        release_name = ""
        for line in list_out.split("\n"):
            line = line.strip()
            if target in line:
                # release name is last word
                release_name = line.split()[-1]

            # once we find any, exit
            if release_name:
                break

        return release_name

    async def require_deployment_tool(self, rebuild=False):
        """Check if the deployments tool has been built and build if not"""
        if rebuild or not os.path.exists(os.path.join(self.SCRIPTS, DEPLOYMENTS_CMD)):
            print("Building {} tool".format(DEPLOYMENTS_CMD))
            await self.run_subprocess(
                ["go", "build", "github.com/kiteco/kiteco/kite-go/cmds/"+DEPLOYMENTS_CMD],
                logfile_name="go-build-deployments",
                error_mesg="Error building `{}` tool".format(DEPLOYMENTS_CMD),
                success_mesg="Built `{}` tool successfully".format(DEPLOYMENTS_CMD))

    # pylint: disable=too-many-arguments
    async def run_subprocess(
            self, args, cwd=None,
            logfile_name="cmd-log", error_mesg="Error!", success_mesg="Success!",
            additional_logfile_path=None,
            error_ping=None, success_ping=None,
            env=None):
        """Run a shell using subprocess, output results through slack"""

        if cwd is None:
            cwd = self.SCRIPTS
        bot = self.bot

        if self.safe_mode:
            # print out all subprocess commands and arguments while in safe mode
            bot.send("running {} in dir {}".format(" ".join(args), cwd))

        # mkdir -p $self.SCRIPT_OUTPUT
        try:
            os.makedirs(self.SCRIPT_OUTPUT)
        except OSError as e:
            if e.errno == errno.EEXIST and os.path.isdir(self.SCRIPT_OUTPUT):
                pass
            else:
                raise

        if env is not None:
            env.update(self.cmd_env)
        else:
            env = self.cmd_env

        ts = datetime.now().isoformat()
        outpath = os.path.join(self.SCRIPT_OUTPUT, "%s-out-%s.txt" % (logfile_name, ts))
        errpath = os.path.join(self.SCRIPT_OUTPUT, "%s-err-%s.txt" % (logfile_name, ts))
        with open(outpath, "w") as outfile, open(errpath, "w") as errfile:
            print("subprocess info:", args, outfile, errfile, cwd, sep="\n")
            proc = await asyncio.create_subprocess_exec(*args, stdout=outfile, stderr=errfile, cwd=cwd, env=env)
            await proc.wait()
        if proc.returncode == 0:
            print("Success!", success_mesg)
            if not os.path.exists(outpath) and os.stat(outpath).st_size > 0:
                bot.upload(os.path.basename(outpath), outpath)

            if success_ping is not None:
                bot.send("@{} {}".format(success_ping, success_mesg))
            else:
                bot.send(success_mesg)

            return
        else:
            bot.send("(return code %d)" % proc.returncode)
            self.try_uploading(outpath)
            self.try_uploading(errpath)
            if additional_logfile_path is not None:
                self.try_uploading(additional_logfile_path)

            if error_ping is not None:
                bot.send("@{} {}".format(error_ping, error_mesg))
            else:
                bot.send(error_mesg)

            raise Exception("Command error")

    def try_uploading(self, filepath):
        """Try to upload a file through slack"""
        self.bot.send("attempting to upload filepath: %s" % filepath)
        for path in glob.glob(filepath):
            if os.path.isdir(path):
                self.bot.send("[unable to upload file because it is a directory: %s]" % path)
            elif os.stat(path).st_size == 0:
                self.bot.send("[unable to upload file because it is empty: %s]" % path)
            else:
                self.bot.upload(os.path.basename(path), path)

    async def gather_and_await(self, tasks, return_exceptions=True):
        """Gather tasks and await for tasks to finish"""
        group = asyncio.gather(*tasks, return_exceptions=return_exceptions)
        res = await group
        # check for exceptions in the result
        self.check_exceptions(res)
        return res

    def check_exceptions(self, results):
        """Check results for exceptions and raise them"""

        exceptions = [r for r in results if isinstance(r, Exception)]

        if exceptions:
            raise GatherError("The following exceptions were raised:\n\n```\n{}\n```".format(
                "\n```\n```\n".join(
                    [exception_str(e) for e in exceptions])))

def exception_str(e):
    """Helper to convert exception with traceback into string"""
    return "\n".join(traceback.format_tb(e.__traceback__)) + "\n" + str(e)

class GatherError(Exception):
    """Exception for when some task errors in an async gather"""
    pass

if __name__ == "__main__":
    pass
