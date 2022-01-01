class user_base::executable {
  file {'/mnt/kite':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    }

    $process_file = "${facts[kiteco_version]}/${user_base::process_name}"

    file {'/mnt/kite/releases':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    }

    archive { "/mnt/kite/releases/${process_file}":
      ensure => present,
      source => "s3://kite-deploys/${process_file}"
    } ~> exec {'kiteco permissions':
      command     => "chmod 755 /mnt/kite/releases/${process_file}",
      path        => ['/bin', '/usr/bin'],
      refreshonly => true,
      onlyif      => "test `stat -c '%a' /mnt/kite/releases/${process_file}` != 755"
    }

    file {'/mnt/kite/s3cache':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    }

    file {'/mnt/kite/logs':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    }

    file {'/mnt/kite/certs':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    } -> archive { '/mnt/kite/certs/rds-combined-ca-bundle.pem':
      ensure => present,
      source => 's3://XXXXXXX/rds-combined-ca-bundle.pem'
    }

    file {'/mnt/kite/tmp':
      ensure => directory,
      owner  => ubuntu,
      group  => ubuntu
    }

    archive { '/var/kite/config.sh':
      ensure => present,
      source => "s3://XXXXXXX/config/${facts['aws_region']}.sh"
    } ~> exec {'kiteco_config_permissions':
      command     => 'chmod 755 /var/kite/config.sh',
      path        => ['/bin', '/usr/bin'],
      refreshonly => true,
      onlyif      => "test `stat -c '%a' /var/kite/config.sh` != 755"
    }

    file {'/var/kite':
      ensure => link,
      target => '/mnt/kite',
      owner  => ubuntu,
      group  => ubuntu
    }

    archive { '/usr/local/libtensorflow-cpu-linux-x86_64-1.15.0.tar.gz':
      ensure       => present,
      source       => 's3://kite-data/tensorflow/libtensorflow-cpu-linux-x86_64-1.15.0.tar.gz',
      extract      => true,
      extract_path => '/usr/local',
    } ~> exec {'ldconfig':
      command     => 'ldconfig',
      path        => ['/sbin'],
      refreshonly => true,
      notify      => Service[$user_base::process_name],
    }

    file {'/var/kite/run.sh':
      content => epp('user_base/run.sh.epp', {executable=>"/var/kite/releases/${process_file}"}),
      owner   => ubuntu,
      group   => ubuntu,
      mode    => '0755',
    } ~> systemd::unit_file { "${user_base::process_name}.service":
      content => file('user_base/kiteco.service'),
    }
    ~> service {$user_base::process_name:
      ensure   => 'running',
      provider => 'systemd',
    }
}
