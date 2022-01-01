# == Class: kite::golang::install
#
# Installs golang from the official source.
#
# === Parameters:
#
# [*version*]
#   The version of golang to install.
#
# [*arch*]
#   The architecture to install golang for.
#
# [*goroot*]
#   Where to install golang. Also sets up $GOROOT.
#
class kite::golang::install (
  $version = $kite::golang::params::version,
  $arch    = $kite::golang::params::arch,
  $goroot  = $kite::golang::params::goroot,
) inherits kite::golang::params {
  include sys::curl
  include kite::golang::common

  $filename = "go${version}.${arch}.tar.gz"
  $download_url = "https://storage.googleapis.com/golang/${filename}"
  $unarchive_location = dirname($goroot)

  Exec {
    path => "$goroot/bin:/usr/local/bin:/usr/bin:/bin",
  }

  exec { "download":
    command => "curl -o /tmp/$filename $download_url",
    creates => "/tmp/$filename",
    unless  => "which go && go version | grep '$version'",
    require => Class["sys::curl"],
  } ->
  exec { "unarchive":
    command => "tar -C $unarchive_location -xzf /tmp/$filename && rm /tmp/$filename",
    onlyif  => "test -f /tmp/$filename"
  }

  exec { "remove-previous":
    command => "rm -rf $goroot",
    onlyif  => [
      "test -d /usr/local/go",
      "which go && test `go version | cut -d' ' -f 3` != 'go$version'",
    ],
    before => Exec["unarchive"],
  }

  file { "/etc/profile.d/golang.sh":
    content => template("kite/golang/golang.sh.erb"),
    owner   => root,
    group   => root,
    mode    => "a+x",
  }
}
