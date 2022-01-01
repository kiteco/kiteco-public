#!/bin/bash

DEST="/deploy"
LOGDIR="/var/kite/log"

TARGET="mockserver"
HOST="mock.kite.com"

scp mockserver.py mock.kite.com:/deploy
ssh mock.kite.com "sudo killall gunicorn"
ssh mock.kite.com "sudo -b gunicorn --pythonpath /deploy --bind 0.0.0.0:80 mockserver:app"
