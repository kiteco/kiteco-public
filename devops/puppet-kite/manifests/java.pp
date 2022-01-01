# == Class: kite::java8
#
# Sets up java8 via webupd8team PAA
#
class kite::java {
  include kite::ubuntu::bootstrap

  include apt
  apt::ppa { 'ppa:webupd8team/java':
    require => Package['software-properties-common']
  }

  file { '/tmp/java.preseed':
    source => 'puppet:///modules/kite/java/java.preseed',
    mode   => '0600',
  }

  package { 'oracle-java8-installer':
    ensure       => present,
    responsefile => '/tmp/java.preseed',
    require      => [
      Apt::Ppa['ppa:webupd8team/java'],
      File['/tmp/java.preseed'],
      Class['apt::update'],
    ]
  }
}
