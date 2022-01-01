import sys

from ..foo import Foo
from ..bar import Bar
from ..runtime_helper import LARGE

_PY3 = sys.version >= '3'

foo = Foo()
bar = Bar()


class Test(object):
    if _PY3:
        foo = Foo.foo
    else:
        foo = Foo.foo.__func__  # broken on PY3
    large = LARGE
