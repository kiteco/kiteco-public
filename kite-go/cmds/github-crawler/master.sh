#!/bin/bash

mkdir -p logs
nohup ./github-crawler -master -input=python-repos-header.csv &> logs/master.log &
