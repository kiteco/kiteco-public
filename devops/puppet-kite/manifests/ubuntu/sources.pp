# == Class: kite::ubuntu::sources
#
# Setup aptitude sources, and ensure this class is setup
# before installing any packages via "package".
#
class kite::ubuntu::sources {
  include apt
  include apt::update

  apt::source { 'docker':
      location          => 'https://apt.dockerproject.org/repo',
      release           => 'ubuntu-trusty',
      repos             => 'main',
      key               => 'XXXXXXX',
      key_source        => 'https://apt.dockerproject.org/gpg',
      required_packages => 'debian-keyring debian-archive-keyring',
      pin               => 10,
      include_src       => false,
  }

  Class['kite::ubuntu::sources'] -> Package<| |>
}
