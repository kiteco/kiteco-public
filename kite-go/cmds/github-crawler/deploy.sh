#!/bin/bash

for i in `seq 0 4`; do
    echo "stoping crawler-$i..."
    ssh crawler-$i.kite.com "killall github-crawler"
done

GOOS=linux go build

for i in `seq 0 4`; do
    echo "deploying to crawler-$i..."
    scp crawler.sh crawler-$i.kite.com:
    scp github-crawler crawler-$i.kite.com:
done

echo "starting master @ crawler-0..."
scp master.sh crawler-0.kite.com:
ssh crawler-0.kite.com "./master.sh"

for i in `seq 0 4`; do
    echo "starting crawler-$i..."
    ssh crawler-$i.kite.com "rm -rf outputdir/*"
    ssh crawler-$i.kite.com "./crawler.sh"
done

echo "done"
