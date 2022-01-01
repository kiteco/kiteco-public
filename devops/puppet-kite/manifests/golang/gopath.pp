# == Class: kite::golang::gopath
#
# Sets up a user's $GOPATH, including src, bin, pkg and the environment variable.
#
# === Parameters:
#
# [*path*]
#   The path to use for $GOPATH
#
# [*owner*]
#   The owner of $GOPATH
#
# [*group*]
#   Group of $GOPATH
#
class kite::golang::gopath(
  $path   = "/usr/local/gopath",
  $owner  = undef,
  $group  = undef,
) {
  include kite::golang

  file { $path:
    ensure => directory,
    owner  => $owner,
    group  => $group,
  }

  file { [ "$path/src", "$path/bin", "$path/pkg" ]:
    ensure => directory,
    owner  => $owner,
    group  => $group,
  }

  file { "/etc/profile.d/gopath.sh":
    content => template("kite/golang/gopath.sh.erb"),
    owner   => root,
    group   => root,
    mode    => "a+x",
  }

}
