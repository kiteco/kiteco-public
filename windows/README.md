# Developing on Windows

Below are a set of instructions that get you from a vanilla Windows installation to building Kite.

Once you download this VM, you will still have to generate SSH keys, setup GOPATH and clone the repository from the `Get the codebase` section below.

# Setting up to develop on Windows

This document will go through how to setup a vanilla Windows 10 installation to work with the Kite codebase. Once you complete this guide, you should be able to build both the Copilot and kited.exe, and run them together by running the `./run_kited.sh` command in this directory.

## Install Golang

Go to the [official golang website](https://golang.org/dl/) and download the latest version of Golang [supported by our codebase](https://github.com/kiteco/kiteco#go). Download and install via the MSI installer.

## Install Chocolatey

All the other dependencies we will need can be installed via Chocolatey, a package manager for Windows. You can find detailed instructions [here](https://chocolatey.org/docs/installation). As of writing of this document, you simply have to start `cmd.exe` as Administrator (next to the start button, type "cmd.exe", right click, "Run As Administrator"), and paste the following command:

```sh
@"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -InputFormat None -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"
```

## Install dependencies

Now that you have chocolatey installed, we can start to install our dependencies. Chocolatey opperates much like `apt`, `port` or `homebrew`. One thing to note is that Chocolatey likes to run as administrator, so be sure to start `cmd.exe` (or Git Bash) as Administrator when using chocolatey. 

To install everything to build the copilot and kited.exe, run:

```sh
choco install -y git git-lfs mingw make nodejs yarn
```

## Start using Git Bash

Git ships with a very useful bash emulator called Git Bash. I recommend you put this to your taskbar. It behaves a ALMOST like a normal linux terminal. There may be better alternatives for this, but its provided by the Git for Windows package.

## Get the codebase

Generate your ssh keys:

```sh
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
```

Add your **PUBLIC KEY** (e.g `cat ~/.ssh/id_rsa.pub`) it to your GitHub account under Settings > SSH Keys.

Setup your `GOPATH` and clone the repository
```
mkdir -p ~/go/src/github.com/kiteco
cd ~/go/src/github.com/kiteco
git clone git@github.com:kiteco/kiteco
cd kiteco
git lfs pull
```

Note - if `git clone` does not work with the message `Couldn't agree a key exchange algorithm`, starting troubleshooting [here](https://stackoverflow.com/questions/54608687/windows-git-bash-fatal-could-not-read-from-remote-repository-when-pushing-thro) could help

## Start building

To run Kite, go to the `windows` directory (i.e the directory where this README is).
For 1st time setup, you'll need to build the copilot first:

```sh
.\build_electron.sh
```
To run Kite:

```sh
.\run_kited.sh
```

If `.\run_kited.sh` yields the following output:
```
Checking for the copilot (Kite.exe) ...
Found the copilot, building kited.exe (659411b458b2f469b2227a104f432082bbca8b33) ...
..\kite-go\lang\python\pythonresource\testing.go:8:2:
..\kite-go\client\datadeps\datadeps-bindata.go:1:1: expected 'package', found version
..\kite-go\client\internal\metrics\livemetrics\manager.go:22:2:
..\kite-go\client\internal\plugins\bindata.go:1:1: expected 'package', found version
./run_kited.sh: line 29: ./kited.exe: No such file or directory
```

run `git lfs pull` and try again.

If you get the error:
```
.../github.com/kiteco/kiteco/windows/kited.exe: error while loading shared libraries: ?: cannot open shared object file: No such file or directory
```
Try installing Visual C++ Redistributable for Visual Studio 2015 (https://www.microsoft.com/en-gb/download/details.aspx?id=48145, not sure what version between x86 and x64 is required, installing both works) 

# Building the Installer

To build to full Windows installer for Kite, you'll need to install a couple more packages:

```sh
choco install -y nant dotnet3.5 visualstudio2019community
```

Once these are installed, you need to make a few tweaks:
- `mkdir installer/current_build_bin/out` (theres probably a better way to handle this, just haven't tried hard enough)
- run `make kite-windows` from the root of the repo using `Git Bash` running as Administrator. 
- disable the `authenticodekey` section of `installer/BuildInstaller.build`

If you run into any issues, ask @tarakju. These tweaks won't be necessary after we update the actual Windows build VM is updated using the install instructions in this document.

# Notes

For detailed notes on previous attempts at getting a developer workflow going, check out these Quip documents. They might provide some details that could be helpful if you're trying to dig deeper. However, they could be dated...

* https://kite.quip.com/rgaSALr7Rabf/Windows-Developer-Notes
* https://kite.quip.com/pIOIAboC3eIq/Notes-on-Kited-and-Copilot-Dev-on-Windows

## VM Notes

If you're running Windows within a VM, and would like to mount a network drive to share code, the default directory is `Z:\`.
