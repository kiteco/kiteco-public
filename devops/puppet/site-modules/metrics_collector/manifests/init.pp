class metrics_collector (
  Array[String] $streams,
  Boolean $stdout_output,
  String $kinesis_output,
) {

  require kite_base
  contain metrics_collector::server
  contain metrics_collector::fluentd

  include metrics_collector::server
  include metrics_collector::fluentd

  metricbeat::modulesd { 'nginx':
    require => Package['metricbeat'],
    content => file('metrics_collector/etc/metricbeat/modules.d/nginx.yml')
  }
}
