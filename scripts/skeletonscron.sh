#!/bin/bash

# Reminder, that this script is run on build.kite.com!

LOCKFILE=/var/lock/skeletonscron
DATE=`date +"%Y-%m-%d %H-%M-%S"`

HOME=/home/ubuntu
GOROOT=/usr/local/go
GOPATH=$HOME/go
KITECO=$GOPATH/src/github.com/kiteco/kiteco
LOGDIR=$HOME/cron/logs/skeletons
CURATION=$HOME/curation-team

# So we can find go and other installed binaries
PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# check_and_notify takes a commands return code ($1) and a message to display
# if the return code is not 0 ($2). NOTE: Use $? to check the return code of a
# command after running it.
function check_and_notify {
	if [[ $1 != 0 ]]; then
		echo "[$DATE] $2"
		kite-email \
			-to="errors@kite.com" \
			-subject="[curation-python-skeletons-pipeline] $2" \
			-body=$3
		exit $1
	fi
}

(
	flock -nx 200 || exit 1

	if [[ `hostname` != "build.kite.com" ]]; then
		echo "this is not build.kite.com. exiting."
		exit 1
	fi

	# Create logdir if it doesn't exist
	mkdir -p $LOGDIR
	check_and_notify $? "could not mkdir -p $LOGDIR" ""

	echo "verifying kiteco repo and pulling most recent version"
	cd $KITECO
	BRANCH=`git symbolic-ref --short -q HEAD`
	if [ "$BRANCH" != "master" ]; then
		check_and_notify 1 "FATAL: on $BRANCH instead of master" ""
	fi
	git reset --hard
	git pull

	echo "verifying curation team repo and pulling most recent version"
	cd $CURATION
	# ensure we are on master
	BRANCH=`git symbolic-ref --short -q HEAD`
	if [ "$BRANCH" != "master" ]; then
		check_and_notify 1 "FATAL: on $BRANCH instead of master" ""
	fi
	git reset --hard
	git pull


	# clean, build, geneate bindata, verify, commit, push
	cd $KITECO/local-pipelines/curation-python-skeletons

	echo "cleaning ..." 
	make clean &> $LOGDIR/make_clean.log
	check_and_notify $? "could not run make clean" $LOGDIR/make_clean.log

	echo "building skeletons ..." 
	make skeletons &> $LOGDIR/make_skeletons.log
	check_and_notify $? "could not run make skeletons." $LOGDIR/make_skeletons.log

	echo "generating bindata ..."
	make bindata &> $LOGDIR/make_bindata.log
	check_and_notify $? "could not run make bindata." $LOGDIR/make_bindata.log

	echo "validating dataset ..."
	make validate &> $LOGDIR/make_validate.log
	check_and_notify $? "could not run make validate." $LOGDIR/make_validate.log
	

	echo "committing changes to git ..."
	cd $KITECO

	git commit kite-go/lang/python/pythonskeletons/bindata.go -m "[autocommit] updating curation python skeletons dump"  &> $LOGDIR/git_commit.log
	check_and_notify $? "could not commit updated bindata.go" $LOGDIR/git_commit.log

	git push -u origin master &> $LOGDIR/git_push.log
	check_and_notify $? "could not git push" $LOGDIR/git_push.log

) 200>$LOCKFILE
