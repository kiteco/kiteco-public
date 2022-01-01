from django.conf.global_settings import *
from .djangosettings import INSTALLED_APPS as _DEFAULT

INSTALLED_APPS = _DEFAULT + [
    "algoliasearch_django",
]

ALGOLIA = {
    'APPLICATION_ID': 'MyAppID',
    'API_KEY': 'MyApiKey',
}
