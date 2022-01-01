# General
This directory contains a set of tools for running import exploration on a set of python packages.

# Contents

## skipped.md
The list of packages that we do not explore for various reasons.

## packagelist
This tool is used for bootstrapping the list of packages and versions to explore.

## dockertools
This tool is used to generate dockerfiles and manage docker images for import exploration.

## dockertools/dockerfiles
These are the Dockerfils for all of the packages that are explored.

## dockertools/dockercontext
This directory contains the context in which each dockerimage is built.

## dockertools/baseimage
This directory contains the Dockerfile for the base image used in import exploration.

## explore
This tool is used to explore python packages inside dockerimages.

## upload
This is a simple tool to upload the raw outputs of the import graph to s3.

## internal/pkg
This directory contains a parser and format for specifying packages that will
be converted to dockerfiles and then used in exploration.

## internal/docker
This directory contains a simple api wrapping around docker shell commands.

# Usage

## Building a graph shard for a new package
Suppose we are building a graph shard for the package `abc`.

Steps:
    - Create pkginfo, see `packagelist/packagelists/special` for examples.
    - Generate the dockerfile:
    ```
    cd dockertools
    go build
    ./dockertools files pkginfo dockerfiles/abc
    ```
    See dockertools/README.md for more details.

    - Add any special instructions to `dockertools/dockerfiles/abc`.

    - Build the dockerimage:
    ```
    cd dockertools
    go build
    ./dockertools buildimage dockerfiles/abc
    ```
    See dockertools/README.md for more details.

    - Explore the package:
    ```
    cd explore
    go build
    ./explore package dockerimagename
    ```
    See explore/README.md for more details.

## Building a new graph shard and adding it to the import graph
- TODO

# TODO
- Integrate with resource manager
- Get full pipeline running on an azure instance
- Figure out what to do with dependencies for packages that do not have a requirements.txt
  file specified. One option is to import the package and then walk the namespace and see all
  of the names that were introduced by importing the package. This has one drawback which
  is that we would not get versions.
- Alot of the packages with special instructions are needed because the library specifies compatibility
  with 2 and 3 but some dependency (e.g django) is strictly python 3.

# When did this run? Where is the data?

- The last successful run was on 2018-02-06; the corresponding dockerfiles are committed in `git`,
  and the resulting data is stored in `s3://kite-data/import-exploration/2018-02-06/`
  (`out.log` contains the top-level output).
