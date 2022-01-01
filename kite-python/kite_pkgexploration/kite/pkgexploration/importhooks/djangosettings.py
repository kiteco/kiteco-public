# only import this for django settings

from django.conf.global_settings import *

INSTALLED_APPS = [
    "django.contrib.admin",
    "django.contrib.contenttypes",
    "django.contrib.sites",
    "django.contrib.auth",
    "django.contrib.flatpages",
    "django.template.context_processors",
    "django.contrib.redirects",
    "django.contrib.sessions",
]

ROOT_URLCONF = 'kite.pkgexploration.importhooks.djangourlconf'
