class kite::stagingrelease (
  $environment = undef,
  $hostname = undef,
) {
  if str2bool($::vagrant) {
    $owner = "vagrant"
    $group = "vagrant"
  } else {
    $owner = "ubuntu"
    $group = "ubuntu"
  }
  
  file { ["/var/kite", "/var/kite/releases", "/var/kite/log"]:
    ensure => directory,
    owner => $owner,
    group => $group,
  }

  include kite::python # for pip
  include nginx
  include kite::ubuntu::bootstrap

  # Set the system environment variables 
  file { "/etc/environment":
    content => template("kite/prod/environment.sh.erb"),
    owner   => "root",
    group   => "root",
  }

  # install s3cmd (for deployments)
  package { 's3cmd':
    provider => 'pipx',
    ensure => present,
  }

  if str2bool($::vagrant) {
    exec { "make-certs":
      command => "/usr/bin/openssl req -new -newkey rsa:2048 -days 364 -nodes -x509 -subj '/C=US/ST=CA/L=SF/O=./OU=./CN=192.168.30.10' -keyout /etc/ssl/release.kite.com.key -out /etc/ssl/server.crt",
      unless  => "/bin/ls /etc/ssl/server.crt",
    }
  } else {
    exec { "get-cert":
      command => "/usr/local/bin/s3cmd get s3://XXXXXXX/ssl/server.crt /etc/ssl/server.crt",
      unless => "/bin/ls /etc/ssl/server.crt",
    }
    exec { "get-key":
      command => "/usr/local/bin/s3cmd get s3://XXXXXXX/ssl/server.key /etc/ssl/server.key",
      unless => "/bin/ls /etc/ssl/server.key",
    }
  }

  # nginx config
  file { "/etc/nginx/sites-available/release.kite.com":
    content => template("kite/nginx/release.kite.com.erb"),
    owner => "root",
    group => "root",
    notify => Service["nginx"],
  } ->
  file { "/etc/nginx/sites-enabled/release.kite.com":
    ensure => 'link',
    target => "/etc/nginx/sites-available/release.kite.com",
    notify => Service["nginx"],
  }

  package { "ca-certificates":
    ensure => present,
  }
    
  class { 'postgresql::globals':
    manage_package_repo => true,
    version => '9.4',
    encoding => 'UTF-8',
    locale => 'en_US.UTF-8',
  }->
  class { 'postgresql::server':
    listen_addresses => '*',
    pg_hba_conf_defaults => false,
  }

  postgresql::server::pg_hba_rule { 'local access as postgres user':
    type        => 'local',
    database    => 'all',
    user        => 'postgres',
    auth_method => 'ident',
    order       => 1,
  }

  postgresql::server::pg_hba_rule { 'local access':
    type        => 'local',
    database    => 'all',
    user        => 'all',
    auth_method => 'md5',
    order       => 2,
  }

  postgresql::server::pg_hba_rule { 'IPv4 access':
    type        => 'host',
    database    => 'all',
    user        => 'all',
    address     => '127.0.0.1/32',
    auth_method => 'md5',
    order       => 3,
  }

  if str2bool($::vagrant) {
    postgresql::server::pg_hba_rule { 'vm host access':
      type        => 'host',
      database    => 'all',
      user        => 'all',
      address     => '192.168.30.1/32',
      auth_method => 'md5',
      order       => 3,
    }
  } else {
    postgresql::server::pg_hba_rule { 'vpn access':
      type        => 'host',
      database    => 'all',
      user        => 'all',
      address     => '10.86.0.0/24',
      auth_method => 'md5',
      order       => 3,
    }
  }
  
  postgresql::server::pg_hba_rule { 'IPv6 access':
    type        => 'host',
    database    => 'all',
    user        => 'all',
    address     => '::1/128',
    auth_method => 'md5',
    order       => 4,
  }

  postgresql::server::role { 'kite':
    login         => true,
    password_hash => postgresql_password('XXXXXXX', 'XXXXXXX'),
  }

  postgresql::server::db { 'apprelease':
    owner    => 'kite',
    user     => 'XXXXXXX',
    password => 'XXXXXXX',
  }
}
