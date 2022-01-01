# this is a namespace package, allowing us to separate out e.g.
# `kite.pkgexploration` into a separate installable distribution
__path__ = __import__('pkgutil').extend_path(__path__, __name__)
