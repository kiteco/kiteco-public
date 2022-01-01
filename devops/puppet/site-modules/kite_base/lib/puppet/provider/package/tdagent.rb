module Puppet::Parser::Functions
  Puppet::Type.type(:package).provide :tdagent, :parent => :gem, :source => :gem do
    has_feature :install_options, :versionable
    commands :gemcmd => '/opt/td-agent/usr/sbin/td-agent-gem'
  end
end
