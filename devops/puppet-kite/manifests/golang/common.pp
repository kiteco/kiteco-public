# == Class: kite::golang::common
#
# Installs some packages that are often useful with golang.
#
class kite::golang::common {
  package{ ["mercurial", "protobuf-compiler"]:
    ensure => present,
  }
}
