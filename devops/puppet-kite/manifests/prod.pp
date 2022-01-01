# == Class: kite::prod
#
# Common configuration for prod/test machines
#
class kite::prod (
  $environment = undef,
  $hostname = undef,
  $vagrant_ip = undef,
) {
  # If we are in vagrant the owner and group is "vagrant"
  # Otherwise, the owner and group is "ubuntu"
  if str2bool($::vagrant) {
    $owner = "vagrant"
    $group = "vagrant"
  } else {
    $owner = "ubuntu"
    $group = "ubuntu"
  }

  # Install things
  include nginx
  include kite::python
  include kite::ubuntu::bootstrap
  include kite::golang::install
  include kite::postgresql_dev

  # Setup gopath
  $home = "/home/$owner"
  $gopath = "$home/go"
  class { 'kite::golang::gopath':
    path  => $gopath,
    owner => $owner,
    group => $group,
  }

  # Create directory structure for kiteco repo.
  $repo = "$gopath/src/github.com/kiteco/kiteco"
  $repo_dirs = [ "$gopath/src/github.com",
                 "$gopath/src/github.com/kiteco"]

  if str2bool($::vagrant) {
    # Make the symlink from $GOPATH/src/github.com/kiteco/kiteco to /kiteco
    file { $repo:
      ensure => link,
      target => "/kiteco",
    }
  }

  # Puppet will create the directories in the right sequence here.
  file { $repo_dirs:
    ensure => directory,
    owner  => $owner,
    group  => $group,
  }

  # Setup /var/kite to point to /mnt/kite. In vagrant, this is
  # not really needed, but it makes it consistent with our EC2 nodes
  # for now (which mount their storage at /mnt)
  file { "/mnt/kite":
    ensure => directory,
    owner  => $owner,
    group  => $group,
  } ->
  file { "/var/kite":
    ensure => link,
    owner  => $owner,
    group  => $group,
    target => "/mnt/kite"
  } ->
  file { ["/var/kite/log", "/var/kite/data", "/var/kite/bin", "/var/kite/tmp"]:
    ensure => directory,
    owner  => $owner,
    group  => $group
  }

  file { "/deploy":
    ensure => directory,
    owner  => $owner,
    group  => $group
  }

  # Create self-signed ssl certificates
  file { "$home/certs":
    ensure => directory,
    owner  => $owner,
    group  => $group,
  }

  if str2bool($::vagrant) {
    exec { "make-certs":
      command => "/usr/bin/openssl req -new -newkey rsa:2048 -days 364 -nodes -x509 -subj '/C=US/ST=CA/L=SF/O=./OU=./CN=$vagrant_ip' -keyout $home/certs/server.key -out $home/certs/server.crt",
      unless  => "/bin/ls $home/certs/server.key",
      require => File["$home/certs"],
    }
  } else {
    exec { "make-certs":
      command => "/usr/bin/yes 'xx' | /usr/bin/openssl req -new -newkey rsa:2048 -days 364 -nodes -x509 -keyout $home/certs/server.key -out $home/certs/server.crt",
      unless  => "/bin/ls $home/certs/server.key",
      require => File["$home/certs"],
    }
  }

  # Set the environment variables for login shells
  file { "/etc/profile.d/kite.sh":
    content => template("kite/prod/profile.sh.erb"),
    owner   => "root",
    group   => "root",
    mode    => "a+x",
  }

  # Set the system environment variables
  file { "/etc/environment":
    content => template("kite/prod/environment.sh.erb"),
    owner   => "root",
    group   => "root",
  }

  # Setup docker
  class { 'kite::docker':
    user => $owner, # Add user to docker group
  }

  package { 's3cmd':
    provider => 'pipx',
    ensure   => present,
  }

  # Set nginx configuration
  file { "/etc/nginx/sites-available/usernode":
    content => template("kite/nginx/usernode.erb"),
    owner   => "root",
    group   => "root",
    notify  => Service["nginx"],
  } ->
  file { "/etc/nginx/sites-enabled/usernode":
    ensure => 'link',
    target => "/etc/nginx/sites-available/usernode",
    notify  => Service["nginx"],
  }
}
