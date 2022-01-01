class kite_base::packages {
  package { 'htop':
    ensure => present,
  }
  $awscli_require = ($facts['gce'] == undef) ? {
    true             => [],
    default          => File['/root/.aws/config'],
  }
  package { 'python3':
    ensure   => present
  }
  -> package { 'python3-pip':
    ensure   => present
  }
  -> package { 'awscli':
    ensure   => latest,
    provider => pip3,
    require  => $awscli_require,
  }
  package { 'lsb-release':
    ensure => installed,
  }
  package { 'less':
    ensure => installed,
  }
  package { 'groff':
    ensure => installed,
  }

  apt::key { 'cloud.google':
    id     => 'XXXXXXX',
    source => 'https://packages.cloud.google.com/apt/doc/apt-key.gpg',
    server => 'packages.cloud.google.com',
  }
  -> apt::source { 'cloud.google.sdk':
      location => 'http://packages.cloud.google.com/apt',
      release  => '',
      repos    => 'cloud-sdk main',
      key      => {
        'id' => 'XXXXXXX',
      },
      include  => {
        'deb' => true,
      },
  }
  -> package { 'google-cloud-sdk':
    ensure  => installed,
    require => Class['apt::update'],
  }

  package {'apt-transport-https':
    ensure => installed,
  }
}
