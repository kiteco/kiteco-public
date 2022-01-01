#!/bin/bash

secs_to_human() {
    echo "$(( ${1} / 3600 ))h $(( (${1} / 60) % 60 ))m $(( ${1} % 60 ))s"
}

SKIPSTEPS=0
TMPDIR=/data/kite/tmp

for i in "$@"
do
case $i in
    --model=*)
    MODEL="${i#*=}"
    shift # past argument=value
    ;;
    --lang=*)
    LANG="${i#*=}"
    shift # past argument=value
    ;;
    --vocab=*)
    VOCAB="${i#*=}"
    shift # past argument=value
    ;;
    --contextsize=*)
    CONTEXTSIZE="${i#*=}"
    shift # past argument=value
    ;;
    --batchsize=*)
    BATCHSIZE="${i#*=}"
    shift # past argument=value
    ;;
    --steps=*)
    STEPS="${i#*=}"
    shift # past argument=value
    ;;
    --skipsteps=*)
    SKIPSTEPS="${i#*=}"
    shift # past argument=value
    ;;
    --traindir=*)
    TRAINDIR="${i#*=}"
    shift # past argument=value
    ;;
    --validatedir=*)
    VALIDATEDIR="${i#*=}"
    shift # past argument=value
    ;;
    --numgpu=*)
    NUMGPU="${i#*=}"
    shift # past argument=value
    ;;
    --tmpdir=*)
    TMPDIR="${i#*=}"
    shift # past argument=value
    ;;
    *)
    echo "Unknown option "${i}
    echo "usage: datagen.sh --model=<model> --lang=<lang> --vocab=<vocab> "
    echo "                  --contextsize=<contextsize> --batchsize=<batchsize> --steps=<steps>"
    echo "                  --traindir=<traindir> --validatedir=<validatedir> --numgpu=<numgpu> [--tmpdir=<tmpdir>]"
          # unknown option
    ;;
esac
done

if [[ -z "$MODEL" || -z "$LANG" || -z "$VOCAB" || -z "$CONTEXTSIZE" || -z "$TRAINDIR" || -z "$VALIDATEDIR" || -z $BATCHSIZE || -z $STEPS || -z $NUMGPU ]]; then
    echo "usage: datagen.sh --model=<model> --lang=<lang> --vocab=<vocab> "
    echo "                  --contextsize=<contextsize> --batchsize=<batchsize> --steps=<steps>"
    echo "                  --traindir=<traindir> --validatedir=<validatedir> --numgpu=<numgpu> [--tmpdir=<tmpdir>]"
    exit 1
fi

# Seconds is an internal variable that counts seconds the script has been running.
# This rests it to zero so we get a count of seconds datagen was running
SECONDS=0
datagen \
    --lang=$LANG \
    --vocab=$VOCAB \
    --contextsize=$CONTEXTSIZE \
    --batchsize=$BATCHSIZE \
    --steps=$STEPS \
    --skipsteps=$SKIPSTEPS \
    --stepsperfile=500 \
    --traindir=$TRAINDIR \
    --validatedir=$VALIDATEDIR \
    --numgpu=$NUMGPU \
    --tmpdir=$TMPDIR

ret=$?
if [[ $ret != 0 ]]; then
    ip=`hostname -I`
    dur="$(secs_to_human $SECONDS)"
    slack -a -C danger -t "$MODEL" "datagen failed on $ip after $dur with exit code $ret"
fi
