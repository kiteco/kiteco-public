Getting started with the codebase
=================================

Our codebase is primarily located at github.com/kiteco/kiteco (http://github.com/kiteco/kiteco). There are a few auxiliary repositories that host very experimental code, but the goal is to make the “kiteco” repository the point of truth for all of our services.


Summary (TL;DR)
---------------

* Our codebase is primarily Go. (`kite-go`, `kite-golib` directories)
* Infrastructure uses Terraform (for AWS) provisioning, and Fabric/shell scripts for deployment and management of remote hosts (`devops` directory)
* You need VPN credentials to access any of our remote AWS (or Azure) hosts.
* Platform-specific logic & instructions live in subdirectories `osx`, `windows`, `linux`. You probably don't need these.

Git LFS
--
We use [Git LFS](https://git-lfs.github.com/) to store our various `bindata.go` files. You will need to install the command line tool to get the contents of those files when you pull the repository. Installation instructions are on their website, but for MacOS you can install it by running (from inside the `kiteco` repository)
```
brew update
brew install git-lfs
git lfs install
```
Then do a `git pull` to get the bindata.go files. If they do not download from LFS, try running `git lfs pull` (you should only need to do this once - subsequent `git pull`s should update the bindata correctly).

### Optional: Improving Performance

`git lfs install` installs a [smudge filter](https://git-scm.com/docs/gitattributes) that automatically downloads and replaces the contents of newly checked out "pointer files" with their content.
By default smudge filters operate on checked out blobs in sequence, so cannot download in batch as would typically happen when running `git lfs pull`.
Furthermore, by default, git checkouts will block on downloading the new LFS files which can be annoying.
You might prefer to disable the smudge filter (this can be run even if you've already run the regular `git lfs install`):
```
git lfs install --skip-smudge
git lfs pull
```

Then, when building after a new checkout, you may see an error of the form "expected package got ident."
This occurs because `go` reads some Go files and sees the Git LFS pointers instead of the actual data file.
At this point, you can download the latest files with `git lfs pull` and rebuilding should work.

Nothing needs to be done when pushing LFS blobs. That will still happen automatically.

Go
--

The bulk of our code is currently in Go.
This can be found at github.com/kiteco/kiteco/kite-go (http://github.com/kiteco/kiteco/kite-go).
To get started working in this part of the codebase, first make sure you have your Go environment setup correctly (i.e Go is installed,  $GOPATH is set, etc.).

Locally, however, you will need to install Go 1.15.3. The following steps will get you going.

Set `$GOPATH` in your .profile / .bashrc/ .bash_profile / .zshrc, e.g:

```sh
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

Make sure to create these directories as well:

```sh
mkdir $HOME/go
mkdir $HOME/go/src $HOME/go/bin $HOME/go/pkg
```

If you are on a Mac and set the above in either .bashrc or .zshrc, make sure to load it in either your .profile or .bash_profile.
See [this](http://www.joshstaiger.org/archives/2005/07/bash_profile_vs.html) for an explanation.

It would be useful to become familiar with how `go` code is organized. Check out https://golang.org/doc/code.html for more on this topic.

Navigate to where the `kiteco` repo will live in your `GOPATH`, and clone the repo.

```sh
# Create kiteco directory within GOPATH, and clone the repo there
mkdir -p ~/go/src/github.com/kiteco
cd ~/go/src/github.com/kiteco
git clone git@github.com:kiteco/kiteco
```

To install the latest version of Go that's compatible with our codebase, run:

```sh
cd ~/go/src/github.com/kiteco/kiteco
cd devops/scripts
./install-golang.sh
```

From here, just run `make install-deps` from the root of the `kiteco` repo to get basic utilities installed.

```sh
# Install dependencies
make install-deps
```

Use `./scripts/update-golang-version.sh` if you'd like to make Kite require a newer version of Golang.

### Tensorflow

For development builds (see below), you may need to have Tensorflow installed globally on your system.

```bash
make install-libtensorflow
```

Building Kite
-------------

You're now ready to build Kite! First, build the sidebar for your platform

```bash
./osx/build_electron.sh force
# ./linux/build_electron.sh force
# ./windows/build_electron.sh force
```

This process is asynchronous to the Kite daemon build,
so you must manually rebuild the sidebar as needed.

Now build and run Kite:

```bash
make run-standalone
```

Note that this is not a full Kite build, but is the recommended approach for development, as it is much faster.
Some functionality is disabled in the development build (depending on the platform):

- Kite system tray icon
- Updater service


Development
-----------

You should be able to develop, build, and test Kite entirely on your local machine.
However, we do have cloud instances & VMs available for running larger jobs and for
[testing our cloud services](VAGRANT.md)

### Dependency Management with Go Modules
We use the [Go Modules](https://blog.golang.org/using-go-modules) system for dependency management.

General tips:
- make sure in `~/go/src/github.com/kiteco/kiteco` and not a symlink
- make sure deps are updated to the versions in `go.mod`: `go mod download`
-  Set `$GOPRIVATE` in your .profile / .bashrc/ .bash_profile / .zshrc, e.g: `export GOPRIVATE=github.com/kiteco/*`.

To add or update a dependency, all you need to do is `go get` it, which
will automatically update the `go.mod` and `go.sum` files. To remove a dependency, 
remove references to it in the code and run `go mod tidy`. In general, make sure to
run `go mod tidy` to make sure all new dependencies have been added and unused ones 
have been removed before committing any dependency changes.

The process for updating a dependency is:
- `go get -u github.com/foo/bar`
- (optional) run any `go` command, such as `go build`, `go test`
- `go mod tidy`
- `git add go.mod go.sum`
- `git commit ...`

The process for adding a dependency is:
- `go get github.com/foo/bar`
- edit code to import "github.com/foo/bar"
- `go mod tidy`
- `git add go.mod go.sum`
- `git commit ...`

#### HTTPS Auth
`godep` may attempt to clone private repositories via HTTPS, requiring manual authentication.
Instead, you can add the following section to your `~/.gitconfig` in order to force SSH authentication:

```
[url "git@github.com:"]
	insteadOf = https://github.com/
```

### Datasets, Datadeps

We bundle a lot of pre-computed datasets & machine learning models into the Kite app
through the use of a custom filemap & encoding on top of [go-bindata](https://github.com/jteeuwen/go-bindata).
The data, located in `kite-go/client/datadeps`, is kept in Git-LFS.

All needed data files is first stored on S3.
There are pointers at various places in our codebase to S3 URIs.
After updating references to these datasets, the datadeps file must be manually rebuilt:

```
$ ./scripts/build_datadeps.sh
```

This will bundle all data that is loaded at Kite initialization time.
You must ensure the needed data is loaded at initialization, otherwise it will not be included!


### Logs

Some logs are displayed in Xcode, but most are written to a log file:

```shell
tail -F ~/.kite/logs/client.log
```

### Testing and Continuous Integration

Your Go code should pass several quality criteria before being allowed into the master branch. Travis CI (https://travis-ci.org/) acts as the gatekeeper between pull requests and merging. You can test your code before pushing to a pull request to speed up the process by navigating to the `kite-go` directory and running `make *` commands directly (any of `make (fmt|lint|vet|bin-check|build|test)`).

### VPN Access

You will need access to our VPN to connect to our backend hosts.

* Get VPN credentials (*.ovpn file) from @tarak (You will need to type in a password IRL - don't IM/chat it)
* Install Tunnelblick for OS X (https://code.google.com/p/tunnelblick/)
* Double click on the “.ovpn” file that contains your credentials.
* Tunnelblick should automatically apply the configuration.. look for the icon on the OS X status bar
* Click on the Tunnelblick icon, select your config, and enter your VPN password. (**NOTE**: Tunnelblick will complain saying the IP hasn't changed. Check the box to disable the message and continue.)
* Ping 'test-0.kite.com' and make sure it resolves.  It's okay if the pings timeout; ICMP is disabled by default on aws instances.

### SSH Access

Kite's Dropbox has ssh credentials for all the machines on AWS and Azure under Shared > Engineering > keys > kite-dev.pem and Shared > Engineering > keys > kite-dev-azure. Place both of these in your .ssh directory, i.e. ~/.ssh/kite-dev.pem. As a convenience, you should add the following to your `~/.ssh/config`:

```
Host *.kite.com
    ForwardAgent yes
    IdentityFile ~/.ssh/kite-dev.pem
    User ubuntu

# Test instances are on Azure
Host test-*.kite.com
    User ubuntu
    IdentityFile ~/.ssh/kite-dev-azure
```

Don't forget to set appropriate permissions on the credential files (e.g. 700)
