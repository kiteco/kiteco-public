#!/bin/bash
# This script updates the testdata .json files in this directory.
# Requirements:
# - Kite running on localhost:46624
# - The 1st argument must be a file in a whitelisted directory, it will be overwritten with temporary data to let Kite work on it

# Outputs the index of substring $2 in $1
function indexOf {
    echo "$1" | grep -b -o "$2" | cut -d: -f1
}

function updateJson {
    PYTHON_FILE="$1"
    JSON_FILE="$2"
    [[ ! -f "$PYTHON_FILE" ]] && echo "Python file not found: $PYTHON_FILE. Exiting." && exit 1

    ESCAPED_FILE="$(echo "$TMP_FILE" | sed -e 's/\//:/g')"
    OFFSET="$(indexOf "$(cat $PYTHON_FILE)" "<caret>")"
    CONTENT="$(cat $PYTHON_FILE | sed 's/<caret>//')"
    # Contains JSON escaped file content
    ESCAPED_CONTENT="$(echo "$CONTENT" | tr '\n' '~' | sed 's/~/\\n/g' | sed 's/"/\\"/g')"

    echo
    echo "Updating $JSON_FILE with callee of $PYTHON_FILE @ $OFFSET"

    echo "$CONTENT" > "$TMP_FILE"
    MD5="$(md5sum "$TMP_FILE" | egrep -o '^[^ ]+')"
    echo -e "\tOffset:\t $OFFSET" 1>&2
    echo -e "\tMD5:\t $MD5" 1>&2
#    echo "        JSON: \"$ESCAPED_CONTENT\"" 1>&2

    curl -s -f -XPOST --data "{\"source\":\"intellij\",\"action\":\"focus\",\"filename\":\"$TMP_FILE\",\"text\":\"$ESCAPED_CONTENT\",\"selections\":[{\"start\":$OFFSET,\"end\":$OFFSET}]}" -H 'Accept: application/json' 'http://localhost:46624/clientapi/editor/event'
    [[ "$?" -ne "0" ]] && echo -e "Focus event failed for $PYTHON_FILE. Exiting" 1>&2 && exit 1

    sleep 5
    curl -f -s "http://localhost:46624/api/buffer/intellij/$ESCAPED_FILE/$MD5/callee?offset_bytes=$OFFSET" > temp.json
    [[ "$?" -ne "0" ]] && echo -e "JSON could not be retrieved for $PYTHON_FILE. Exiting.\n\tResponse: $(curl -s "http://localhost:46624/api/buffer/intellij/$ESCAPED_FILE/$MD5/callee?offset_bytes=$OFFSET")" 1>&2 && rm -f temp.json && exit 1

    mv "temp.json" "$JSON_FILE"
}

TMP_FILE="$1"
[[ -z "$TMP_FILE" ]] && echo -e "Usage: $0 whitelistedTempFile.py\n" && exit 1
[[ ! -d "$(dirname "$TMP_FILE")" ]] && echo "Parent dir of temp file not found: $TMP_FILE. Exiting." && exit 1

# Update all our test data files.
updateJson "json.loads.0.py" "json.loads.json"
updateJson "vararg.0.py" "vararg.json"
updateJson "regularArg.defValue.py" "regularArg.json"
updateJson "constructor.0.py" "constructor.json"
