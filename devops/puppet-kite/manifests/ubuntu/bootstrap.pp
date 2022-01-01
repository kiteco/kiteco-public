# == Class: kite::ubuntu::bootstrap
#
# Entry-point to bootstrap a ubuntu system. Installs some common packages.
#
class kite::ubuntu::bootstrap {
  include kite::ubuntu::sources

  include sys::htop
  include sys::curl
  include sys::wget
  include sys::unzip
  include sys::tmux
  include sys::screen
  include sys::rsync
  include sys::git

  # If we are in vagrant the owner and group is "vagrant"
  # Otherwise, the owner and group is "ubuntu"
  if str2bool($::vagrant) {
    $owner = "vagrant"
    $group = "vagrant"
  } else {
    $owner = "ubuntu"
    $group = "ubuntu"
  }

  # gives us add-apt-repository
  package { 'software-properties-common':
    ensure => present,
  }

  # sys::gcc installs build-essential
  include sys::gcc

  package{ ['vim', 'emacs24']:
    ensure => present,
  }

  package { 'gawk':
    ensure => present,
  }

  # for systray package
  package { ['libgtk-3-dev', 'libappindicator3-dev']:
    ensure => present,
  }

  # create a .emacs file that disables creating ugly backup files everywhere
  file { "/home/$owner/.emacs":
    source  => "puppet:///modules/kite/emacs/dotemacs",
    owner   => $owner,
    group   => $group,
    mode    => "ug+w",
  }
}
