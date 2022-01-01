node "kite-base.kite.dev" {
  include kite_base
}

node "metrics-collector.kite.dev" {
  include kite_base
  include metrics_collector
}

node "airflow.kite.dev" {
  include airflow
}


node "user-node.kite.dev" {
  include user_base
}

node "user-mux.kite.dev" {
  include user_base
}

node /release-[0-9a-z]+/ {
  include release_server
}

node /nchan-[0-9a-z]+/ {
  include nchanted
}

node /kiteserver-[0-9a-z]+/ {
  include kite_server
}
