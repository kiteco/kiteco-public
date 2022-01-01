#!/bin/bash
echo "building.."
GOOS=linux go build -o ./usernode-inspector github.com/kiteco/kiteco/kite-go/user/cmds/usernode-inspector
GOOS=linux go build -o ./status-inspector github.com/kiteco/kiteco/kite-go/cmds/status-inspector
GOOS=linux go build -o ./web-snapshots github.com/kiteco/kiteco/kite-go/cmds/web-snapshots
GOOS=linux go build -o ./systems-monitor github.com/kiteco/kiteco/kite-go/cmds/systems-monitor

echo "killing existing.."
ssh metrics-azure.kite.com "killall usernode-inspector"
ssh metrics-azure.kite.com "killall status-inspector"
ssh metrics-azure.kite.com "killall web-snapshots"
ssh metrics-azure.kite.com "killall systems-monitor"

sleep 1

echo "syncing.."
scp usernode-inspector metrics-azure.kite.com:
scp status-inspector metrics-azure.kite.com:
scp web-snapshots metrics-azure.kite.com:
scp systems-monitor metrics-azure.kite.com:

sleep 1

echo "starting.."

ssh metrics-azure.kite.com 'nohup ./status-inspector -port=":4040" &> status.log &'
ssh metrics-azure.kite.com 'nohup ./usernode-inspector -port=":3030" &> usernode.log &'
ssh metrics-azure.kite.com 'nohup ./web-snapshots -urls=http://metrics-azure.kite.com,http://users-azure.kite.com &> snapshots.log &'
ssh metrics-azure.kite.com 'nohup ./systems-monitor &> monitor.log &'

rm -f usernode-inspector
rm -f status-inspector
rm -f web-snapshots
rm -f systems-monitor
