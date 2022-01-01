#!/bin/bash

GH_DUMP_BUCKET=kite-github-crawl
GH_REPOS_LIST_BUCKET=s3://kite-emr/github-repos-list/%t.emr
TEMP_EMR=temp.emr


# 1) get list of directories (one per organization) and write as EMR
go install github.com/kiteco/kiteco/emr-pipelines/github-unzip-filter/internal/cmds/build-repos-list

build-repos-list $GH_DUMP_BUCKET $TEMP_EMR

# 2) upload list of repos to s3
go install github.com/kiteco/kiteco/kite-go/cmds/kfsput

kfsput $TEMP_EMR $GH_REPOS_LIST_BUCKET

# cleanup
rm $TEMP_EMR