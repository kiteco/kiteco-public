# General
This directory contains a basic binary and package for interacting with the amazon emr api programatically.

# Setup

- Create a [`virtualenv`](http://docs.python-guide.org/en/latest/dev/virtualenvs/)
- Install packages in `requirements.txt` via [`pip`](https://pip.pypa.io/en/stable/)
- Install `kite`

```sh
$ virtualenv -p python2.7 env     # Create virtualenv
$ source env/bin/activate         # Activate virutalenv
$ pip install -r requirements.txt # Install third party dependencies in the virtualenv
$ ./setup.py install              # Install the kite module in the virtualenv
```