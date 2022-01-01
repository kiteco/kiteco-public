class user_base::fluentd {
  include fluentd

  fluentd::plugin { 'fluent-plugin-kinesis': }
  fluentd::plugin { 'fluent-plugin-ec2-metadata': }
  fluentd::plugin { 'fluent-plugin-systemd': }

  # TODO: This should be inteagrated more with Hiera data
  # Do this if/when we want to turn logging on for user-mux
  file { "${fluentd::config_path}/500-serverlogs.conf":
    ensure  => $fluentd::package_ensure,
    content => file('user_base/etc/td-agent/config.d/500-serverlogs.conf'),
    require => Class['Fluentd::Install'],
    notify  => Class['Fluentd::Service'],
  }

  # This is a little jank, but td-agent must be in systemd-journal group
  user { 'td-agent':
    ensure  => present,
    groups  => ['systemd-journal'],
    require => Package[$fluentd::package_name],
  }
}
