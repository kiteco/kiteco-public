#! /usr/bin/env bash
exec docker run -e PUPPET_FACT_node_name="$1.kite.dev" -v $PWD/local/facts.d:/etc/facter/facts.d/ -v $HOME/.aws:/root/.aws/ -v $PWD:/opt/puppet -it kite/base:latest
