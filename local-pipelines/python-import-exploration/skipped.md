# General
Packages that are not included in import exploration for various reasons.

# Versioning issues
Packages that are skipped because we were unable to find an official
release version.

- `django-passbook`
- `pyasn`
- `tubing`
- `vtk`
- `nova`

# Misc
- `python-quantum` not on pypi or apt
- `django-south` only used as a binary
- `appnope` only for use on osx
- `pyside` needs to be manually installed for linux, http://pyside.readthedocs.io/en/latest/building/linux.html
- `django-celery` poorly maintained, requires.txt says django >= 1.8 but this package is not python 3 compatible.
- `django-ember` does not appear to be maintained anymore.
- `bencode` does not build properly
- `trepan` does not build properly
- `setuptools_scm` does not build properly
- `pytz` does not build properly
- `pysnmp` does not build properly
- `pysmi` does not build properly
- `pyasn1` does not build properly
- `pyasn1-modules` does not build properly
- `poster` does not build properly
- `nilsimsa` does not build properly
- `holmium.core` does not build properly
- `google-apputils` does not build properly
- `frida` does not build properly
- `flask-cache` does not build properly
- `SQLObject` does not build properly
- `sklearn` is subsumed by `scikit-learn`
- `uWSGI` is a binary package
- `uwsgitop` is a binary package
- `wincertstore` is Windows only
- `mysql` is a proxy for `MySQL-python`
- `argparse` is shipped with Python `>=2.7` or `>=3.2`
- `ipython_genutils` is deprecated
- `uuid` conflicts with the standard library package (is it the same?)
- `importlib` is a backport of the Python 2.7 library
- `mixpanel-py` is deprecated (replaced by `mixpanel`)
- `youtube_dl` is a binary package
- `unittest2` is almost identical to Py2.7 `unittest` (it's a backport)
- `pip2` is mostly a binary package, and we fail to discover top-levels
- `letsencrypt`, etc are binary packages

# Packages to Revisit

## Problems With the Top-Level
* `azure__2.0.0`
* `carbon__1.1.1`
* `fake-factory__9999.9.9`: top-level is Faker, but we try faker
* `graphite-web__1.1.1`
* `ipython__6.2.1`: blacklisted (why?)
* `prettytable__7`
* `ptyprocess__0.5.2`
* `pyqt5__5.10.0`
* `python-consul__0.7.2`: all explored names have slashes (paths)
* `python-gnupg__0.4.1`
* `rst2pdf__0.92`
* `terminado__0.8.1`
* `testpath__0.3.1`
* `uritemplate.py__3.0.2`: notably distinct from `uritemplate`
* `zodb3__3.11.0` : top-level is ZODB, but we try ZODB3

## Configuration Needed
* `django-haystack__2.6.1`
* `django-pyscss__2.0.2`
* `django-tables2__1.17.1`

## Broken Requirements (Django)
* `django-appconf__1.0.2`: made PR (accepted)
* `django-autoslug__1.9.3`: made PR (pending)
* `django-cors-headers__2.1.0`
* `django-countries__5.0`
* `django-filter__1.1.0`
* `django-fsm__2.6.0`
* `django-guardian__1.4.9`
* `django-ipware__2.0.1`
* `django-jsonfield__1.0.1`
* `django-nose__1.4.5`
* `django-object-actions__0.10.0`
* `django-picklefield__1.0.0`
* `django-redis-cache__1.7.1`
* `djangorestframework__3.7.7`
* `django-rest-swagger__2.1.2`
* `django-ses__0.8.5`
* `django-uuidfield__0.5.0`: requires django<1.10
* `dj-static__0.0.6`

## Broken Requirements (Other)
* `ghostscript__0.6`: requires Ghostscript c lib (libgs)
* `gitpython__2.1.8`: requires git executable
* `glance_store__0.23.0`: requires netbase via eventlet
* `mozcrash__1.0`: requires mozinfo
* `openstackdocstheme__1.18.1`: requires sphinx
* `oslo.messaging__5.35.0`: requires netbase via eventlet
* `oslo.service__1.29.0`: requires netbase via eventlet
* `os-win__3.0.0`: requires netbase via eventlet
* `python-magic__0.4.15`: requires libmagic

## Miscellaneous
* `subliminal__2.0.5`: weird `NotImplementedError` in `pkg_resources`
- `webpy` exploration does not terminate within 2 days (infinite loop?)


# Blacklisted packages from old import graph
See `dockertools/entrypoint/entrypoint.py`
