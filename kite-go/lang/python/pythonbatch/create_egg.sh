#!/bin/bash

EGGDIR=$1
mkdir -p $EGGDIR
mkdir tmp
cp setup.py tmp
cp -r mymath tmp
cd tmp
python setup.py bdist_egg
cp -r dist/mymath-*.egg $EGGDIR
cd ..
rm -rf tmp
