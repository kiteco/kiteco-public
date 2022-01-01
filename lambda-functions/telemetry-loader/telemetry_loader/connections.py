import certifi
import contextvars
import functools

from elasticsearch_async import AsyncElasticsearch
from elasticsearch import Elasticsearch
from elasticsearch.connection.http_urllib3 import create_ssl_context
from elasticsearch.connection import Connection


def get_connection(func):
    @property
    @functools.wraps(func)
    def wrapper(self):
        if func.__name__ not in self._conns:
            self._conns[func.__name__] = func(self)
        return self._conns[func.__name__]
    return wrapper


connections_var = contextvars.ContextVar('connections')


es_host = 'https://XXXXXXX.us-east-1.aws.found.io:9243'


class Connections(object):
    def __init__(self):
        self._conns = {}

    def get(self, conn_name):
        return self._conns.get(conn_name)

    @get_connection
    def elasticsearch_async(self):
        context = create_ssl_context(cafile=certifi.where())

        return AsyncElasticsearch(
            hosts=[es_host],
            headers={'authorization': Connection(api_key=(
                'DU-XXXXXXX', 'XXXXXXX')).headers['authorization']},
            ssl_context=context
        )

    @get_connection
    def elasticsearch(self):
        return Elasticsearch(
            hosts=[es_host],
            api_key=('DU-XXXXXXX', 'XXXXXXX'),
            use_ssl=True,
            ca_certs=certifi.where(),
            verify_certs=True
        )
