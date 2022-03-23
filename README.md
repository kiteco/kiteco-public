This is a public version of the main Kite repo
=================================
The main Kite repo (originally `kiteco/kiteco`) was intended for private use. It has been lightly adapted for publication here by replacing private information with `XXXXXXX`. As a result many components here may not work out of the box.

Intro to parts of the codebase
=================================

**Q: How did we analyze all the code on Github?**

We used a variety of infrastructure, on a mix of cloud platforms depending on what was the most economical, though it was mostly on AWS.

We used mostly map-reduce to manage workflows that needed to be run over large datasets. You can see a list of some of our map-reduce jobs [here](https://github.com/kiteco/kiteco-public/tree/master/local-pipelines) (`local-pipelines`) and [here](https://github.com/kiteco/kiteco-public/tree/master/emr-pipelines) (`emr-pipelines`). I believe tasks in local-pipelines are intended to be ran on single instances whereas EMR is AWS's map-reduce infrastructure.

Here are some example tasks, with a particular focus on Python analysis:

-   [Github crawler](https://github.com/kiteco/kiteco-public/blob/master/kite-go/cmds/github-crawler/crawler.go)
-   [Package exploration](https://github.com/kiteco/kiteco-public/tree/master/kite-python/kite_pkgexploration/kite/pkgexploration), which imports and introspects Python packages (as you can imagine, this is very tricky code to get right); see the README.md in that directory
    -   Related: [import exploration](https://github.com/kiteco/kiteco-public/tree/master/local-pipelines/python-import-exploration), which, if I understand correctly, runs package exploration and other logic to e.g. upload results to s3
    -   As an example of what package exploration produces, I'm attaching the graph it produced for numpy 1.14.0. For example, if you load the JSON object as `o`, then `o["shards"]["numpy"]["139784536077376"]` will give you the object `numpy.polynomial.tests.test_laguerre.TestEvaluation.c1d` (I just picked one node from the graph at random), and you will see that it has members like `min` `conjugate` and `tofile`, with pointers for each of those to their node from the graph.
    -   (also extracts docstrings)
-   [Type induction](https://github.com/kiteco/kiteco-public/tree/master/local-pipelines/python-offline-metrics/cmds/type-induction) (the [logic](https://github.com/kiteco/kiteco-public/tree/70fa808fcdec5e776d8a7e3ecacd1960e6bfa4d6/kite-go/typeinduction)), which statistically estimates the return types of functions based on attributes accessed on their return values across Github
    -   I'm attaching `type-induction-numpy.json.zip` which contains the output of type induction for numpy. For example, if this JSON file is loaded as `o`, if you look at `o[17]` you will see that there is a 51.9% probability that `numpy.matrix.argmin` returns a `numpy.matrix`.
-   [Dynamic analysis](https://github.com/kiteco/kiteco-public/tree/master/kite-go/dynamicanalysis), which runs short Python scripts like [this one](https://www.kite.com/python/examples/4810/pandas-get-the-dimensions-of-a-%60dataframe%60) (we have a set of 2-3k of these that cover 50% of all open-source Python libraries when weighed by usages/popularity) and extracts type information at runtime
    -   I'm attaching 20190320.json.zip which contains all of the information extracted from this process. As an example, the JSON object on line 34 tells us that `jinja2.environment.Template.render` is a function that returned a `__builtin__.unicode`
-   [Extraction of return types from docs](https://github.com/kiteco/kiteco-public/blob/master/kite-go/lang/python/pythonresource/cmd/build/docs-returntypes/main.go)
    -   I'm attaching docs_returntypes.json.zip which should be pretty self-explanatory. My main comment is that, as in other datasets, the large numbers are pointers to other nodes in the graph.

Several return type sources are unified in [this command](https://github.com/kiteco/kiteco-public/blob/70fa808fcdec5e776d8a7e3ecacd1960e6bfa4d6/kite-go/lang/python/pythonresource/cmd/build/returntypes/main.go).

A lot of this pipeline seems to be orchestrated through [this Makefile](https://github.com/kiteco/kiteco-public/blob/master/kite-go/lang/python/pythonresource/cmd/Makefile). This is broadly documented a bit [here](https://github.com/kiteco/kiteco-public/blob/master/kite-go/lang/python/pythonresource/cmd/build/README.md).

This pipeline results in a number of files per package::version, with the following elements:

-   SymbolGraph (graph of entities)
-   ArgSpec (function signatures)
-   PopularSignatures (function-call patterns that are popular on Github, e.g. "how do people most commonly call matplotlib.pyplot.plot?")
-   SignatureStats
-   Documentation
-   SymbolCounts
    -   aside: here is an example SymbolCounts entry from numpy: `{"Symbol":"numpy.distutils.fcompiler","Data":{"Import":901,"Name":780,"Attribute":711,"Expr":1501,"ImportThis":406,"ImportAliases":{"FC":2,"_fcompiler":2,"fcompiler":8}}}`
    -   this means that numpy.distutils.fcompiler is imported 901 times, used in an expression 1501 times, and is imported "as" most commonly as "fcompiler" although sometimes as "FC" or "_fcompiler"
-   Kwargs
-   ReturnTypes

I'm attaching the final resource build for numpy here as "resource-manager-numpy.zip". You can download the 800MB zip file with all the Python open-source packages [here](https://drive.google.com/file/d/1iObSIPzzJ-OSlaWBkh3vXr-LgQ5Fjood/view?usp=sharing).

The bullet list of resources above is from the code [here](https://github.com/kiteco/kiteco-public/blob/master/kite-go/lang/python/pythonresource/internal/resources/resources.go#L25). You can "find references" to see how these files get loaded from disk. In the Kite client the resource manager's main entry point is [here](https://github.com/kiteco/kiteco-public/blob/master/kite-go/lang/python/pythonresource/manager.go). Note this class includes code for dynamically loading and unloading packages' data into memory to conserve end-user memory.

By the way, we are happy to share any of our open-source-derived data. Our Github crawl is about 20 TB, but for the most part the intermediate and final pipeline outputs are pretty reasonably-sized. Although please let me know soon if you want anything because we will likely end up archiving all of this.

To reiterate, we invested a few $million into our Python tech, so you should find it to be pretty robust and high quality, which is why I'm doing some moonlight work to try to give it a shot at not getting lost.


**Q: Is this infrastructure incremental?**

Generally, no. Fortunately it didn't really need to be. I can't recall how long it took to run the full Python analysis end to end --- it was more than a day but I think less than a week.


**Q: How often did you re-run data collection and analysis of GitHub code?**

We ran several Github crawls throughout our time. I think there were something like ~4 successive crawls during a ~7 year period. Things do change, but not super frequently. The other Python package exploration is much cheaper to run so we ran it more often.


**Q: How do you deploy your ML models?**

Here are some highlights:

-   Everything is in a repeatable, code-defined pipeline
-   Some (most?) resources don't need to change often, so we didn't build everything from scratch on every build
-   Especially for ML models, we wanted to do human review of model performance for every model we shipped; we shipped a new version of the client weekly, so all the more reason that we couldn't retrain from scratch for every build
-   We used incremental updates, basically binary patches, to reduce the bandwidth consumption of every update


**Q: How did you measure the quality of your models?**

I'm not sure I can shed much light here, but here's a rough pass:

-   Background: We used tensorflow to train models offline, and do online inference on end-users' machines. (I know TabNine used tensorflow to train, and rolled their own inference on client side)
-   We used tensorboard to monitor the training process
-   I know the team invested in scrapy solutions for managing training workflows, e.g. Slack notifications when builds failed or finished, etc
-   I don't know the technical details of how we did cross-validation, metrics for model success, etc. You may be able to find it in the Github repo.

In terms of the infrastructure and code:

-   Since my answer to the top question above is mostly focused on our python analysis pipeline, rather than ML pipelines, [here](https://github.com/kiteco/kiteco-public/tree/master/local-pipelines/lexical/train) is where you can find the code and scripts related to training our "lexical" (GPT-2) models.
-   To save on cloud costs we bought our own GPU-equipped machine for training from [Lambda Labs](https://lambdalabs.com/). One of our engineers used it as a room heater in his apartment during the COVID lockdown 🙂

Btw we also trained a simple model to mix lexical/GPT-2 and other completions. (short product spec attached as "Product Spec_ Multi-provider Completions.pdf")

(Bonus: I'm attaching our product spec for lexical completions here as "Product Spec_ Lexical Completions.pdf")


**Q: Did you implement your own parsers or reuse existing ones?**

We implemented our own Python parser in Golang. It is robust to syntax errors, e.g. it can parse partial function calls. It can be found [here](https://github.com/kiteco/kiteco-public/tree/master/kite-go/lang/python/pythonparser).

We also did some parser / formatter work with JavaScript, but did not finish it. We ended up using treesitter for some things after it came out.


**Q: Could you do code linting and refactorings, given that the data about API usages you collect is never complete?**

We did not try to do this very much. We did some experimentation with linting, but to your point having a noisy linter can be worse than no linter at all. I think it's harder to use ML for linting than completions or other use cases for this reason.


**Q: Did you try to pivot to other usages of ML code analysis like automatic code reviews, security checks, etc?**

Yes we did some experimentation on a number of different ideas in late 2020 / early 2021.

- [Synthesizing status summaries](https://docs.google.com/presentation/d/1gyUe8TlqWsT2isfYpO4pbZtmtBwnesnFR5yURkk0q_s/edit?usp=sharing): From an ML perspective, the idea is to use Github PR titles to train a model that can generate "PR titles" from code changes, thus enabling us to make it easy for developers to share descriptions of the work they've been doing more easily.
- ML-enhanced code search and navigation (see attached "Code search product analysis and roadmap.pdf"): one of the key ideas being using ML to annotate a graph of relationships between code entities, so you could e.g. right-click on a string referring to a file and see "See three references to this file". (see the image below.) There was also a playbook for using a presence on developer desktops to get widespread adoption across teams.
> ![Screen Shot 2022-01-09 at 9.41.13 PM.png](readme_assets/unnamed1.png?raw=true)

- We built some prototype tech for mapping between code, issues, and wiki content. These models performed pretty well.

> ![Screen Shot 2022-01-09 at 9.48.13 PM.png](readme_assets/unnamed2.png?raw=true)

- We went on a long product strategy process wherein we spoke with something like ~50 individual developers, eng managers, etc. You can see some of the ideas that made it the furthest in the attached "Kite - Wave 5 - Product concepts for Engineering.pdf". They included smart error reporting, dynamic logs, log debugging helper, code nav/search (mentioned above), and semantic/keyword code search.

- In case either of these resonate, here are another couple lenses on some subset of the ideas we brainstormed:

> ![Screen Shot 2022-01-09 at 9.48.22 PM.png](readme_assets/unnamed3.png?raw=true)

> ![Screen Shot 2022-01-09 at 9.48.29 PM.png](readme_assets/unnamed4.png?raw=true)











[Originally for Kite employees] Getting started with the codebase
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
