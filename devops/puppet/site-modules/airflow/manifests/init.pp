class airflow (

) {

  require 'kite_base'

  package {'libpq-dev':}
  -> package {'pipenv':
    provider => 'pip3'
  }

  accounts::user { 'airflow':
    managehome => false,
    uid        => '4001',
    gid        => '4001',
  }
  -> file {'/opt/airflow':
    ensure  => directory,
    owner   => 'airflow',
    group   => 'airflow',
    recurse => true,
  }
  -> file {'/opt/airflow/airflow.cfg':
    owner  => 'airflow',
    group  => 'airflow',
    source => 'puppet:///modules/airflow/airflow.cfg',
  }

  file {'/run/airflow':
    ensure => directory,
    owner  => 'airflow',
    group  => 'airflow',
  }

  $services = ['scheduler', 'webserver', 'worker']

  $services.each |String $service| {
    systemd::unit_file { "airflow-${service}.service":
      source => "puppet:///modules/airflow/airflow-${service}.service",
    }
    ~> service {"airflow-${service}":
      ensure  => 'running',
      require => [File['/etc/sysconfig/airflow'], File['/opt/airflow'], File['/run/airflow'], Package['pipenv']],
    }
  }

  file {'/etc/sysconfig/':
    ensure => 'directory'
  }
  -> file {'/etc/sysconfig/airflow':
    content => epp('airflow/airflow.env.epp'),
  }
}
