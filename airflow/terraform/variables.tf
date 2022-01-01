variable service_name {
  default = "airflow"
}

variable region {
  default = "us-east-1"
}

variable webserver_port {
  default = 8080
}

variable repository_name {
  type = string
  default = "kite-airflow"
}

variable tag {
  type = string
}

variable tasks {
  default = {
    webserver = {
      port = 8080
      cpu = 0.5 * 1024.0
      memory = 1 * 1024.0
      load_balancer = true
    },
    scheduler = {
      port = 8793
      cpu = 1 * 1024.0
      memory = 2 * 1024.0
      load_balancer = false
    },
    worker = {
      port = 8793
      cpu = 2 * 1024.0
      memory = 4 * 1024.0
      load_balancer = false
    }
  }
}
