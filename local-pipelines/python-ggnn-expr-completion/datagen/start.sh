source env.sh

# cleanup
pkill -f run.sh
killall graph_data_server

rm -rf $OUT_DIR
mkdir -p $OUT_DIR
rm -rf /data/logs/*
mkdir -p /data/logs

nohup ./run.sh > /data/logs/run.log 2>&1 &

