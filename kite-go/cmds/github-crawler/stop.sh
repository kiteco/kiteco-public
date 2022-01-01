#!/bin/bash

for i in `seq 0 4`; do
    echo "stoping crawler-$i..."
    ssh crawler-$i.kite.com "killall github-crawler"
    ssh crawler-$i.kite.com "killall github-crawler"
done

