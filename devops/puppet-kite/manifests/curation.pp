# == Class: kite::curation
#
# Base module for curation configuration.
#
class kite::curation {
  # Configuration of nginx happens via hiera
  include nginx
  include kite::python

  # Installs dependencies for Stanford's CoreNLP parser
  include kite::curation::corenlp

  # Required by the codeexample authoring tool's linter
  package { 'autopep8':
    provider => 'pipx',
    ensure   => present,
  }
  package { 'pylint':
    provider => 'pipx',
    ensure   => present,
  }
}
