#!/bin/bash
set -e

rm -f launch.log
touch launch.log

if ! sudo docker info | grep -q "Swarm: active" &> /dev/null; then
	echo "[info] initializing swarm"
	sudo docker swarm init &>> launch.log
fi

ready=`sudo docker stack ps --format \"{{.Name}}:{{.CurrentState}}\" kite-server 2>> launch.log | grep -i running | wc -l`
if [ $ready != "0" ]; then
	echo "[info] found running services; shutting them down"
	sudo docker stack rm kite-server &>> launch.log
	until [ $ready == "0" ]; do
        	sleep 5
        	ready=`sudo docker stack ps --format \"{{.Name}}:{{.CurrentState}}\" kite-server 2>> launch.log | grep -i running | wc -l`
	done;
fi

if ! sudo docker secret ls | grep -q "kite-server-deployment-token" &>> launch.log; then
	echo "[info] registering deployment token"
	cat deployment_token | sudo docker secret create kite-server-deployment-token - &>> launch.log
fi

sudo service docker restart

echo "[info] bringing up services in 10 seconds..."
sleep 10

echo "[info] bringing up services"
sudo docker stack deploy -c docker-stack.yml kite-server &>> launch.log

total=-1
until [ $ready == $total ]; do
        sleep 5
        ready=`sudo docker stack ps --format \"{{.Name}}:{{.CurrentState}}\" kite-server 2>> launch.log | grep -i running | wc -l`
	    total=`sudo docker stack ps kite-server 2>> launch.log | grep -v PORTS | grep -v Shutdown | wc -l`
        echo "[info] $ready/$total services ready"
done;

echo "[info] all services ready!"
