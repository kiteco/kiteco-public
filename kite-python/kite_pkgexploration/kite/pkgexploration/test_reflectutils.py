import functools
import os
import sys
import unittest

import django
import django.conf
from django.db.models.query import QuerySet

from . import reflectutils
from .examples import foo, bar

_PY3 = sys.version >= '3'

if _PY3:
    import queue
    _BUILTINMOD = "builtins"
    _QUEUEMOD = "queue"
else:
    import Queue as queue
    _BUILTINMOD = "__builtin__"
    _QUEUEMOD = "Queue"


class Foo(object):
    """A mock class"""
    def bar(self):
        """Does nothing"""
        pass

class Bar(Foo):
    pass


class TypeutilsTest(unittest.TestCase):
    def test_get_kind(self):
        f = Foo()
        self.assertEqual(reflectutils.get_kind(f), "object")
        self.assertEqual(reflectutils.get_kind(Foo), "type")
        self.assertEqual(reflectutils.get_kind(reflectutils), "module")
        self.assertEqual(reflectutils.get_kind(Foo.bar), "function")
        self.assertEqual(reflectutils.get_kind(f.bar), "function")

    def test_get_kind_builtins(self):
        self.assertEqual(reflectutils.get_kind(str), "type")
        self.assertEqual(reflectutils.get_kind(str.join), "function")
        self.assertEqual(reflectutils.get_kind(type(str.join)), "type")
        self.assertEqual(reflectutils.get_kind(os), "module")
        self.assertEqual(reflectutils.get_kind(object), "type")
        self.assertEqual(reflectutils.get_kind(type), "type")

    def test_get_kind_numpy(self):
        import numpy
        self.assertEqual(reflectutils.get_kind(numpy), "module")
        self.assertEqual(reflectutils.get_kind(numpy.ndarray), "type")
        self.assertEqual(reflectutils.get_kind(numpy.ndarray.sum), "function")
        self.assertEqual(reflectutils.get_kind(numpy.zeros), "function")

    def test_boundmethod_get_classattr(self):
        self.assertEqual(reflectutils.boundmethod_get_classattr(queue.Queue().put), (queue.Queue, "put"))
        self.assertEqual(reflectutils.boundmethod_get_classattr(Bar().bar), (Foo, 'bar'))

    def test_approx_canonical_name(self):
        f = Foo()
        self.assertEqual(reflectutils.approx_canonical_name(unittest), "unittest")
        self.assertEqual(reflectutils.approx_canonical_name(unittest.TestCase), "unittest.case.TestCase")
        # this no longer works because pytest imports unittest before the runtime patch is applied
        # self.assertEqual(reflectutils.approx_canonical_name(unittest.TestCase.assertEqual), "unittest.case.TestCase.assertEqual")
        with self.assertRaises(Exception):
            reflectutils.approx_canonical_name(1)
        self.assertEqual(reflectutils.approx_canonical_name(queue.Queue().put), "{}.Queue.put".format(_QUEUEMOD))

    def test_approx_canonical_name_django(self):
         # this ensures that django has a settings module and an app
        django.conf.settings.configure()
        django.setup()
        with self.assertRaises(Exception):
            reflectutils.approx_canonical_name(QuerySet())

    def test_approx_canonical_name_builtins(self):
        self.assertEqual(reflectutils.approx_canonical_name(str), "{}.str".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(str.join), "{}.str.join".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(type(str.join)), "{}.method_descriptor".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(os), "os")
        self.assertEqual(reflectutils.approx_canonical_name(object), "{}.object".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(type), "{}.type".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(None), "{}.None".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(True), "{}.True".format(_BUILTINMOD))
        self.assertEqual(reflectutils.approx_canonical_name(NotImplemented), "{}.NotImplemented".format(_BUILTINMOD))
        if not _PY3:
            self.assertEqual(reflectutils.approx_canonical_name(type(None)), "types.NoneType")

    def test_approx_canonical_name_descriptor(self):
        import numpy
        self.assertEqual(reflectutils.approx_canonical_name(numpy.ndarray.ndim), "numpy.ndarray.ndim")

    def test_approx_canonical_name_unboundmethod(self):
        self.assertEqual(reflectutils.approx_canonical_name(bar.Bar.foo), "kite.pkgexploration.examples.foo.Foo.foo")
        self.assertEqual(reflectutils.approx_canonical_name(foo.Foo.foo), "kite.pkgexploration.examples.foo.Foo.foo")

    def test_approx_canonical_name_unboundmethod(self):
        self.assertEqual(reflectutils.approx_canonical_name(bar.Bar().foo), "kite.pkgexploration.examples.foo.Foo.foo")
        self.assertEqual(reflectutils.approx_canonical_name(foo.Foo().foo), "kite.pkgexploration.examples.foo.Foo.foo")

    def test_get_doc(self):
        self.assertEqual(reflectutils.get_doc(Foo), "A mock class")
        self.assertEqual(reflectutils.get_doc(Foo.bar), "Does nothing")

    def test_get_argspec(self):
        def foo(x, y, z=None, **bar):
            pass
        spec = reflectutils.get_argspec(foo)
        self.assertEqual(len(spec['args']), 3)
        self.assertEqual(spec['args'][0]['name'], 'x')
        self.assertEqual(spec['args'][1]['name'], 'y')
        self.assertEqual(spec['args'][2]['name'], 'z')
        self.assertEqual(spec['args'][2]['default_value'], 'None')
        self.assertEqual(spec['kwarg'], 'bar')

        spec = reflectutils.get_argspec(functools.partial(foo, 1, z=4, a=10))
        if sys.version >= '3':
            self.assertEqual(len(spec['args']), 1)
            self.assertEqual(len(spec['kwonly']), 1)
            self.assertEqual(spec['kwonly'][0]['name'], 'z')
            self.assertEqual(spec['kwonly'][0]['default_value'], '4')
        else:
            self.assertEqual(len(spec['args']), 2)
            self.assertEqual(len(spec['kwonly']), 0)  # this is technically wrong, but Py2 funcsigs doesn't support kwonly
            self.assertEqual(spec['args'][1]['name'], 'z')
            self.assertEqual(spec['args'][1]['default_value'], '4')

        self.assertEqual(spec['args'][0]['name'], 'y')
        self.assertEqual(spec['kwarg'], 'bar')
