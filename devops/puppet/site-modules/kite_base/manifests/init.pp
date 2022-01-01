class kite_base {
  $puppet_root = "/opt/kite/puppet"

  class { '::logrotate':
    ensure => 'latest',
    config => {
      dateext       => true,
      compress      => true,
      delaycompress => true,
      minsize       => "10M",
      rotate        => 5,
      ifempty       => true,
    }
  }

  file {['/var/kite', '/var/kite/aws', '/var/kite/bin']:
    ensure => directory,
  }

  if $gce != undef {
    file {'/var/kite/aws/credentials':
      content => epp('kite_base/var/kite/aws/gcp_credentials'),
      mode    => '0777',
      require => File['/var/kite/aws']
    }
    file {'/var/kite/aws/config':
      content => file('kite_base/var/kite/aws/gcp_config'),
    }

    file {'/root/.aws':
      ensure => directory
    }
    ~> file {'/root/.aws/config':
      ensure  => link,
      target  => '/var/kite/aws/config',
      require => File['/var/kite/aws/config']
    }
  }

  file {'/var/kite/aws/run_with_secrets':
    content => file('kite_base/var/kite/aws/run_with_secrets'),
    mode    => '0777',
    require => File['/var/kite/aws']
  }

  file {'/etc/modprobe.d':
    ensure => directory
  }

  file {'/etc/modprobe.d/blacklist-conntrack.conf':
    content => 'blacklist nf_conntrack'
  }

  file {'/etc/sysctl.d':
    ensure => directory
  }

  file {'/etc/sysctl.d/60-kite.conf':
    content => file('kite_base/etc/sysctl.d/60-kite.conf')
  }

  file {'/usr/local/bin/puppet':
    ensure => 'link',
    target => '/opt/puppetlabs/bin/puppet'
  }

  file {'/usr/local/bin/facter':
    ensure => 'link',
    target => '/opt/puppetlabs/bin/facter'
  }

  include 'kite_base::packages'
  include 'kite_base::users'
  include 'kite_base::monitoring'
}
