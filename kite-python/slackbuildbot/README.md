slack buildbot
==============

#### General Function
- There are two global locks, `repo` and `release`. The `release` bot command uses the release lock, **all** other commands use the `repo` lock. This means that you can run any non-release command and a release command simultaneously. You cannot currently run multiple build commands simultaneously.

  - The per-user build environments described below should provide a good starting point for breaking the `repo` lock into multiple more granular locks. The primary issue with this as of Oct 2018 is the reliance on the windows VM to build windows plugins. This resource would seem to require some additional work to make build commands run in parallel.

- When a user sends commands to the bot via slack, the bot will use (and create if necessary) a new isolated repository on the box running the slack bot.

  - It sets up a `~/build_repos/<USERNAME>/` directory as the GOPATH, and attempts to initialize a new go environment rooted in this directory.

- Start the bot with `SAFE_MODE=true` set in the environment to make most (all?) destructive operations into noops. This means there should be no PRs created for bindata, no pushes for modified repos, etc. 
  - **Note:** This was a hastily built feature for development purposes, so don't assume it's going to make all operations 100% safe. It's likely still possible to break things with some edge case or another, so think through the commands you're going to run even with `SAFE_MODE` enabled. That said, I'm leaving the functionality in the repo for future devs to simplify their process some.

#### Oddities and Gotchas

- **FIRST USER COMMAND ISSUE** There is a known issue where the macos plugin build may hang indefinitely without an error message while trying to call `xcodebuild`. The fix for this is to *manually open the `.xcodeproj` file once in xcode* by double clicking it in the finder. The path should be something like `~/build_repos/<USERNAME>/src/github.com/kiteco/kiteco/osx/Kite.xcodeproj`.

  - I wrote a short script that automatically attempts this step, though I'm not sure that it fixes the problem, despite being the same steps that are performed manually. -nick johnson

- **Building submodules** If you're developing a new plugin submodule for the first time, you will have to manually go into solness, into your folder in `~/build_repos`, and do a `git submodule update --init`