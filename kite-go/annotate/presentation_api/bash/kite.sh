#!/usr/bin/env bash

function kite_wrap {
	echo "[[KITE[[$1]]KITE]]"
}

function kite_show_region_delimiter {
	kite_wrap 'SHOW {"region": "'$FOO'", "type": "region"}'
}

function kite_line {
	kite_wrap "LINE $1"
}
