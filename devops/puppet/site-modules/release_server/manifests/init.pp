class release_server {
  require kite_base

  case $facts['virtual'] {
    'docker': {
      include release_server::service_initd
    }
    default:  {
      include release_server::service_systemd
    }
  }

  file {'/var/kite/bin/run.sh':
    content => epp('release_server/run.sh.epp', {
      executable  => '/var/kite/bin/release server',
      env_config  => {
        'RELEASE_DB_DRIVER' => 'postgres',
        'ROLLBAR_ENV'       => 'production',
      },
      secret_keys => ['RELEASE_DB_URI', 'ROLLBAR_TOKEN']
    }),
    owner   => kite,
    group   => kite,
    mode    => '0755',
    notify  => Service['release_server'],
    require => File['/var/kite/bin'],
  }
  -> archive { '/var/kite/bin/release':
    ensure => present,
    source => "s3://kite-deploys/v${$facts['kiteco_version']}/release"
  } ~> exec {'kiteco permissions':
    command     => 'chmod 755 /var/kite/bin/release',
    path        => ['/bin', '/usr/bin'],
    refreshonly => true,
    onlyif      => "test `stat -c '%a' /var/kite/bin/release` != 755",
    notify      => Service['release_server'],
  }
}
