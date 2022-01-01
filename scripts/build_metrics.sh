#!/bin/bash
echo "building.."
GOOS=linux go build -o ./usernode-inspector github.com/kiteco/kiteco/kite-go/user/cmds/usernode-inspector
GOOS=linux go build -o ./status-inspector github.com/kiteco/kiteco/kite-go/cmds/status-inspector
GOOS=linux go build -o ./web-snapshots github.com/kiteco/kiteco/kite-go/cmds/web-snapshots

echo "killing existing.."
ssh metrics.kite.com "killall usernode-inspector"
ssh metrics.kite.com "killall status-inspector"
ssh metrics.kite.com "killall web-snapshots"

sleep 1

echo "syncing.."
scp usernode-inspector metrics.kite.com:
scp status-inspector metrics.kite.com:
scp web-snapshots metrics.kite.com:

sleep 1

echo "starting.."

ssh metrics.kite.com 'nohup ./status-inspector -port=":4040" &> status.log &'
ssh metrics.kite.com 'nohup ./usernode-inspector -port=":3030" &> usernode.log &'
ssh metrics.kite.com 'nohup ./web-snapshots -urls=http://metrics.kite.com,http://users.kite.com &> snapshots.log &'

rm -f usernode-inspector
rm -f status-inspector
rm -f web-snapshots
