import unittest

from .runtime.decorated import get_decorated
from .runtime.qualname import get_qualname

from .examples import runtime


def test_qualname():
    assert get_qualname(runtime.Foo) == 'Foo'
    assert get_qualname(runtime.Foo.generate_id()) == 'Foo.generate_id.<locals>.id'
    assert get_qualname(runtime.Foo.Bar.method) == 'Foo.Bar.method'
    assert get_qualname(runtime.Baz.method) == 'Foo.Bar.method'

def test_qualname_with_decorated():
    assert get_qualname(get_decorated(runtime.Foo.f_broken)) == 'Foo.f_broken'
    assert get_qualname(get_decorated(runtime.Foo.f_simple)) == 'Foo.f_simple'
    assert get_qualname(get_decorated(runtime.Foo.Bar.prop)) == 'Foo.Bar.prop'

def test_refmap(runtime_refmap):
    # it would be nice if we got small integers to work, but it probably requires hacking CPython internals
    # assert runtime_refmap.lookup(runtime, runtime.PI) == 'kite.pkgexploration.examples.runtime_helper.PI'

    # test with module object
    assert runtime_refmap.lookup(runtime, runtime.LARGE) == 'kite.pkgexploration.examples.runtime_helper.LARGE'
    assert runtime_refmap.lookup(runtime, runtime.empty) == 'kite.pkgexploration.examples.runtime_helper.empty'
    # test with module path
    assert runtime_refmap.lookup(runtime.__name__, runtime.LARGE) == 'kite.pkgexploration.examples.runtime_helper.LARGE'
    # test with large int
    assert runtime.LARGE == 12345
    assert runtime_refmap.lookup(runtime, 12345) is None
    # test with small int
    assert runtime.PI == 3
    assert runtime_refmap.lookup(runtime, 3) is None

def test_jsonschema():
    import jsonschema  # this calls pkgutil.get_data(...)
