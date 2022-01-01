class kite_server () {

  require kite_base

  docker::swarm {'cluster_manager':
    init           => true,
  }

  $prefix = "/opt/kite-server"

  archive {"${prefix}/kite-server.tgz":
    ensure       => present,
    extract      => true,
    source       => "s3://kite-deploys/v${$facts['kiteco_version']}/kite-server.tgz",
    extract_path => '/opt/',
    cleanup      => true,
    creates      => "${prefix}/docker-stack.yml",
    require      => [
      Package['awscli'],
    ],
  }
  -> file {"${prefix}/kite-server-deployment-token":
    content => "XXXXXXX\n",
  }
  -> docker::secrets {'kite-server-deployment-token':
    secret_name => 'kite-server-deployment-token',
    secret_path => "${prefix}/kite-server-deployment-token",
  }
  -> docker::stack { 'kite-server':
    ensure        => present,
    stack_name    => 'kite-server',
    compose_files => ["${prefix}/docker-stack.yml"],
    require       => [Docker::Swarm['cluster_manager']],
  }
}
