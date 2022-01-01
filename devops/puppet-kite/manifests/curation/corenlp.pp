# == Class: kite::curation::corenlp
#
# Installs some packages that are required for Standford's CoreNLP parser
#
class kite::curation::corenlp {
  include kite::java

  # Some python libraries required for the python interface to corenlp.
  package { ['pexpect', 'unidecode', 'jsonrpclib']:
    provider => 'pipx',
    ensure   => present,
  }
}
