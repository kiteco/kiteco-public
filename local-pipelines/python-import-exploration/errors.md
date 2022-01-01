# General
This file is meant to catalog the errors found when building
dockerimages and installing python packages. The hope is that
we can use this as a basis for automatically inferring the
correct installation instructions for a new package
by trial and error e.g try building image, if it fails,
look for one of these errors, then try solution.

These fixes all assume:
- The OS is Ubuntu
- Any pip installs should be normalized to the appropriate version of
  pip e.g `pip3` for python 3.

# Errors
ERROR
`ImportError: No module named six.moves`
SOLUTION
`pip install six`

ERROR
`Could not run curl-config`
SOLUTION
`apt install libcurl4-openssl-dev`

ERROR
`AttributeError: 'module' object has no attribute 'lru_cache'`
SOLUTION
`use python 3`

ERROR
`ctypes.util.find_library() did not manage to locate a library called 'augeas'`
SOLUTION
`apt install libaugeas0 augeas-lenses`

ERROR
`EnvironmentError: mysql_config not found`
SOLUTION
`apt-get install libmysqlclient-dev`

ERROR
`fatal error: sasl/sasl.h: No such file or directory`
SOLUTION
`apt install libsasl2-dev`

ERROR
`lber.h: No such file or directory`
SOLUTION
`apt install libldap2-dev libssl-dev libsasl2-dev`

ERROR
`'encoding' is an invalid keyword argument for this function`
SOLUTION
`use python 3`

ERROR
`Could not run curl-config: [Errno 2] No such file or directory`
SOULTION
`apt install libcurl4-openssl-dev`

ERROR
`openssl/ssl.h: No such file or directory`
SOLUTION
`apt install libssl-dev`