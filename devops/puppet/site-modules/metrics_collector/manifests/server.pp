class metrics_collector::server {
  package { 'nginx':
    ensure => present,
  }

  file { '/etc/nginx/nginx.conf':
    content => epp('metrics_collector/nginx.conf.epp', {'streams' => $metrics_collector::streams}),
  }

  file { '/var/log/metrics':
      ensure => directory,
      owner  => www-data
  }

  file { '/etc/systemd/system/nginx.service.d':
    ensure  => directory
  }

  -> file { '/etc/systemd/system/nginx.service.d/nginx.conf':
    ensure  => file,
    content => "[Service]\nLimitNOFILE=65536\n"
  }

  -> service { 'nginx':
    ensure    => 'running',
    enable    => 'true',
    require   => [
      Package['nginx'],
      File['/etc/nginx/nginx.conf'],
      File['/var/log/metrics']
    ],
    subscribe => File['/etc/nginx/nginx.conf'],
  }

  logrotate::rule { 'metrics':
    path          => '/var/log/metrics/*.log',
    rotate        => 5,
    size          => '50M',
    compress      => true,
    delaycompress => false,
    create        => true,
    create_mode   => '0644',
    create_owner  => 'www-data',
    sharedscripts => true,
    rotate_every  => 'hour',
    postrotate    => '/bin/kill -USR1 `cat /var/run/nginx.pid` 2>/dev/null || true',
  }
}
