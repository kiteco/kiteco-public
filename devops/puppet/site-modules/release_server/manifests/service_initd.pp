class release_server::service_initd {
  initscript::service { 'release_server':
    cmd            => '/var/kite/bin/run.sh',
    define_service => true,
  }
}
