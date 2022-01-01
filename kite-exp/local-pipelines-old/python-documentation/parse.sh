#!/bin/bash
go run parse-sphinx/*.go -root=artifacts -output=artifacts/python.json.gz
if [[ "$?" != "0" ]]; then
    echo "error parsing sphinx documents"
fi
