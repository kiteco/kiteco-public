
# This file contains hooks for packages that cannot be imported without first
# doing some setup. For example, django requires a settings module to have been
# configured before many of its submodules can be imported.

import logging
from kite.pkgexploration.dist import normalize_name

# # # # Hooks:

def setup_flask():
    # this ensures that flask.request contains the members it would contain if we were processing an http request
    logging.info("Running custom setup for flask")
    import flask
    app = flask.Flask("example")
    ctx = app.test_request_context()
    ctx.push()


def setup_django():
    # make sure django settings and apps are configured
    logging.info("Running custom setup for django")
    import django.conf
    if not django.conf.settings.configured:
        from .importhooks import djangosettings
        django.conf.settings.configure(default_settings=djangosettings)
        import django
        django.setup()


def setup_djangoallauth():
    logging.info("Running custom setup for django-allauth")
    import django.conf
    from .importhooks import django_allauthsettings
    django.conf.settings.configure(default_settings=django_allauthsettings)
    import django
    django.setup()


def setup_djangoboto():
    logging.info("Running custom setup for django-boto")
    import django.conf
    from .importhooks import django_botosettings
    django.conf.settings.configure(default_settings=django_botosettings)
    import django
    django.setup()


def setup_djangobulkupdate():
    logging.info("Running custom setup for django-bulk-update")
    # explicitly do nothing since they setup django for us.


def setup_djangoalgoliasearch():
    logging.info("Running custom setup for algoliasearch-django")
    import django.conf
    from .importhooks import django_algoliasearchsettings as sets
    django.conf.settings.configure(default_settings=sets)
    import django
    django.setup()


def setup_djangorecaptcha():
    logging.info("Running custom setup for django-recaptcha")
    import django.conf
    from .importhooks import django_recaptchasettings as sets
    django.conf.settings.configure(default_settings=sets)
    import django
    django.setup()

# # # # Hook helpers:

# NOTE: these should be keyed by the pip name for the package, e.g `pip install package`,
#       not the name that is used to import the package, this ensures that we can handle post import hooks for dependencies.
POST_IMPORT_HOOKS = {normalize_name(k): v for k, v in {
    "flask": setup_flask,
    "django": setup_django,
    "djangorestframework": setup_django,
    "django-allauth": setup_djangoallauth,
    "django-boto": setup_djangoboto,
    "django-bulk-update": setup_djangobulkupdate,
}.items()}
PRE_IMPORT_HOOKS = {normalize_name(k): v for k, v in {
    "algoliasearch-django": setup_djangoalgoliasearch,
    "django-recaptcha": setup_djangorecaptcha,
}.items()}


def run_hook(name, dct):
    name = normalize_name(name)
    if name in dct:
        dct[name]()


def post_import(name, _has_run={}):
    if _has_run.get(name):
        return
    _has_run[name] = True
    run_hook(name, POST_IMPORT_HOOKS)


def pre_import(name, _has_run={}):
    if _has_run.get(name):
        return
    _has_run[name] = True
    run_hook(name, PRE_IMPORT_HOOKS)
