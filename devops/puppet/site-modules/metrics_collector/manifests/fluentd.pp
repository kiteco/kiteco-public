class metrics_collector::fluentd {
  include fluentd

  fluentd::plugin { 'fluent-plugin-kinesis': }
  fluentd::plugin { 'fluent-plugin-concat': }

  file { "${fluentd::config_path}/500-metrics.conf":
    ensure  => present,
    content => epp('metrics_collector/fluentd.conf.epp'),
    require => Class['Fluentd::Install'],
    notify  => Class['Fluentd::Service'],
  }
}
