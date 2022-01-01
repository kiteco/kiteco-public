class kite::golang::params {
  $version = "1.11.4"
  $goroot  = "/usr/local/go"

  case $::osfamily {
    'debian': {
      $arch    = "linux-amd64"
    }
    default: {
      fail("Do not know how to install golang on ${::osfamily}.\n")
    }
  }
}

