class kite_base::monitoring {
  class { 'elastic_stack::repo':
    version => 7,
  }

  class {'metricbeat':
    manage_repo    => false,
    major_version  => '7',
    package_ensure => 'latest',
    cloud_id       => 'metrics:XXXXXXX',
    cloud_auth     => '${cloud_auth}',
    outputs        => {'elasticsearch' => {}},
    require        => [Apt::Source['elastic'], Class['apt::update']],
  }

  systemd::dropin_file { 'metricbeat_keystore.conf':
    unit    => 'metricbeat.service',
    content => file('kite_base/metricbeat_keystore.conf'),
    notify  => Service['metricbeat'],
  }

  class { 'filebeat':
      major_version        => '7',
      systemd_override_dir => '/tmp/filebeat_cfg',
      package_ensure       => 'latest',
      manage_repo          => false,
      outputs              => {'elasticsearch' => {}},
      conf_template        => 'kite_base/etc/filebeat/filebeat.conf.erb',
      require              => [Apt::Source['elastic'], Class['apt::update']],
    }

  systemd::dropin_file { 'filebeat_keystore.conf':
    unit    => 'filebeat.service',
    content => file('kite_base/filebeat_keystore.conf'),
    notify  => Service['filebeat'],
  }

}
