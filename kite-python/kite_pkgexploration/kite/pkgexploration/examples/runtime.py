import functools
from .runtime_helper import PI, LARGE, empty

def broken_decorator(f):
    def wrapped(*args, **kwargs):
        return f(*args, **kwargs)
    return wrapped

def simple_decorator(f):
    @functools.wraps(f)
    def wrapped(*args, **kwargs):
        return f(*args, **kwargs)
    return wrapped


class Foo(object):
    @staticmethod
    @broken_decorator
    def f_broken():
        pass

    @staticmethod
    @simple_decorator
    def f_simple():
        pass

    @staticmethod
    def generate_id():
        def id(x):
            return x
        return id

    class Bar(object):
        @property
        def prop(self):
            pass

        def method(self):
            pass


class Baz(Foo.Bar):
    pass
