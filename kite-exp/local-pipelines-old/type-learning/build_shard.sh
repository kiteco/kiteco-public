#!/bin/bash

# This script builds one shard of the import graph inside the import graph image
# It is helpful when debugging the import graph exploration process.

PKG=$1

TMP=$(mktemp -d)

echo
echo "Exploring $PKG."
docker run -i --rm \
	-v $TMP:/host \
	-w /host \
	kiteco/import-exploration $PKG

echo
echo "Building import graph containing one shard."
go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/merge-import-graphs
merge-import-graphs \
	--output $TMP/graph.gob.gz \
	--shards $TMP/graph/$PKG.json \
	--strings $TMP/strings.json.gz \
	--argspecs $TMP/argspecs.json.gz \
	--source $TMP/source.json.gz

echo
echo "Starting graph viewer for $PKG. Open http://scrap.kite.com:3021. Press ctrl-c to terminate."
go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/import-graph-viewer
import-graph-viewer --path $TMP/graph.gob.gz --port ":3021" --skipskeletons
