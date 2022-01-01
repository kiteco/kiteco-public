kite-python
===========

`kite-python` contains the `kite` python module. Some aspects of our system are implemented in python. This is the central location for that code. Note that we use `python3.4`. To use the `kite` module:

- Create a [`virtualenv`](http://docs.python-guide.org/en/latest/dev/virtualenvs/)
- Install packages in `requirements.txt` via [`pip`](https://pip.pypa.io/en/stable/)
- Install `kite`

```sh
$ cd kite-python
$ virtualenv -p python3.4 env     # Create virtualenv
$ source env/bin/activate         # Activate virutalenv
$ pip install -r requirements.txt # Install third party dependencies in the virtualenv
$ ./setup.py install              # Install the kite module in the virtualenv (use setup.py develop for changes to take immediate effect) 
```

Note that the `virtualenv` command above should be run with the appropriate argument for `-p`. If you use macports, the python binary will be `python3.4`. Some systems may have it linked to `python34`, or even just `python`, in which case you don't need to pass in a `-p` parameter.

If you run into Unicode errors when running `./setup.py install`, make sure you don't have non-Python source files in the `bin` folder. See https://docs.python.org/2/distutils/setupscript.html#installing-scripts.
