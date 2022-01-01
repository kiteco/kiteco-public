class release_server::service_systemd {
  systemd::unit_file { 'release_server.service':
    content => file('release_server/release_server.service'),
  }
  ~> service { 'release_server':
    ensure   => 'running',
  }
}
