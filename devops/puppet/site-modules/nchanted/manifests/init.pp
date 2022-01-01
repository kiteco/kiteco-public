class nchanted (

) {
  require kite_base

  package { 'nginx':
    ensure => '1.14.0-0ubuntu1.7',
  }

  file { '/etc/nginx/nginx.conf':
    content => file('nchanted/etc/nginx/nginx.conf')
  }

  file { '/etc/nginx/sites-available/rc.kite.com':
    content => file('nchanted/etc/nginx/sites-available/rc.kite.com')
  }

  file {'/etc/nginx/sites-enabled/default':
    ensure => absent,
  }
  file {'/etc/nginx/sites-enabled/rc.kite.com':
    ensure => 'link',
    target => '/etc/nginx/sites-available/rc.kite.com'
  }

  file { '/etc/systemd/system/nginx.service.d':
    ensure  => directory
  }

  -> file { '/etc/systemd/system/nginx.service.d/nginx.conf':
    ensure  => file,
    content => "[Service]\nLimitNOFILE=1048576\n"
  }

  -> service { 'nginx':
    ensure    => 'running',
    enable    => 'true',
    require   => [
      Package['nginx'],
      File['/etc/nginx/nginx.conf'],
      File['/etc/nginx/modules-enabled/50-mod-nchan.conf'],
      Archive['/usr/lib/nginx/modules/ngx_nchan_module.so'],
      Archive['/etc/nginx/htpasswd'],
    ],
    subscribe => [
      File['/etc/nginx/nginx.conf'],
      File['/etc/nginx/sites-available/rc.kite.com'],
    ]
  }

  archive {'/var/kite/bin/convcohort':
    ensure  => present,
    extract => false,
    source  => "s3://kite-deploys/v${$facts['kiteco_version']}/convcohort",
    require => [
      Package['awscli'],
      File['/var/kite/bin'],
    ],
  }
  ~> exec {'convcohort permissions':
    command     => 'chmod 755 /var/kite/bin/convcohort',
    path        => ['/bin', '/usr/bin'],
    refreshonly => true,
    onlyif      => "test `stat -c '%a' /var/kite/bin/convcohort` != 755",
  }
  ~> systemd::unit_file { 'convcohort.service':
    content => file('nchanted/convcohort.service'),
  }
  ~> service { 'convcohort':
    ensure   => 'running',
  }

  archive { '/etc/nginx/htpasswd':
    source  => 's3://XXXXXXX/htpasswd/nchan.htpasswd',
    extract => false,
    creates => ['/etc/nginx/htpasswd'],
    require => Package['awscli'],
  }


  file { '/etc/nginx/modules-available/50-mod-nchan.conf':
    content => 'load_module modules/ngx_nchan_module.so;'
  }
  -> file {'/etc/nginx/modules-enabled/50-mod-nchan.conf':
    ensure => 'link',
    target => '/etc/nginx/modules-available/50-mod-nchan.conf'
  }

  archive { '/usr/lib/nginx/modules/ngx_nchan_module.so':
    ensure  => present,
    source  => 's3://kite-deploys/nchan/ngx_nchan_module_1.14.0_1.2.7.so',
    require => Package['awscli'],
  }

  metricbeat::modulesd { 'nginx':
    require => Package['metricbeat'],
    content => file('nchanted/etc/metricbeat/modules.d/nginx.yml'),
  }
}
