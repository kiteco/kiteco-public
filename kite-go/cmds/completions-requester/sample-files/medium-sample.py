# this file serves as both a test suite and an illustration of how tracing
# responds to various AST node types

import re

from tracelib import macros, kite_trace, get_all_traced_ast_reprs

with kite_trace:
    for i in range(5):
        print i
    print len([1, 2, 3, 1 + 1])
    x = {1: 2}

    # AugAssign
    a = 0
    a += 1

    # FunctionDef
    def foo(): return 'foo!'
    foo()

    # ClassDef
    class A(object):

        def a_b(self):
            return 1

        @staticmethod
        def a_c():
            return 2
    print A().a_b()
    print A.a_c()

    # Delete
    del x[1]
    # x[1] = 3

    # Print with `dest`
    log = open("/tmp/hi", "w")
    print >>log, "test"

    # For with orelse, Continue
    for i in range(3):
        continue
    else:
        print 'orelse'

    # While, Break
    while True:
        break

    # If, Pass
    if True:
        print 'hi'
    if False:
        pass
    else:
        print 'hi'
    if False:
        pass
    elif False:
        pass
    else:
        print 'hi'

    # With
    with open('/tmp/hi') as f:
        pass

    # TryExcept, Raise
    try:
        raise ValueError('yo')
    except:  # no type, no name
        pass
    try:  # type, no name
        raise ValueError('')
    except ValueError:
        pass
    try:  # type, name
        raise ValueError('hi')
    except ValueError as e:
        pass
    try:  # orelse
        pass
    except ValueError as e:
        pass
    else:
        print 'reached'
    try:
        pass
    finally:
        print 'finally'

    # Assert
    assert True
    try:
        assert False, "hi"
    except:
        pass

    # Import / ImportFrom
    import collections
    from collections import OrderedDict
    from collections import deque as d
    import collections as c

    # Exec
    exec "print a" in {}, {"a": 1}

    # Global
    foo = 1
    global foo

    # BoolOp
    a = True and True
    b = False or True

    # BinOp
    x = 1 + 2
    x = 1 - 2
    x = 1 * 2
    x = '' * 10
    x = 1.5 / 2
    x = 2 / 1.5

    # UnaryOp
    x = 2
    x = ~x

    # Lambda
    x = map(lambda x: x - 2, range(3))

    # IfExp
    if True:
        pass
    if False:
        pass
    else:
        pass
    if False:
        pass
    elif False:
        pass
    else:
        pass

    # Set
    x = set([1, 2, 3])

    # ListComp
    x = [a for a in range(3) if a < 2]

    # SetComp
    y = [1, 3]
    w = [1, 2, 3]
    x = {x + 1 for x in y if True if True for z in w}

    # DictComp
    x = {k: v for (k, v) in [(1, 2), (3, 4)]}

    # GeneratorExp
    x = list(x + 1 for x in [1, 2, 3])

    # Yield
    def foo2():
        yield 2, 3
    x = [x for x in foo2()]

    # Compare
    x = x < 4 < 3
    x = (x < 4) < 3

    # Call
    def foo(a, b, c=2):
        return a + b + c
    print foo(1, 2, 3)
    print foo(1, 2)
    print foo(1, 2, c=1)
    print foo(*(1, 2, 3))
    print foo(1, 2, **{'c': 10})
    print foo(*(1, 2), **{'c': 10})

    # Repr
    print repr([1, 10])

    # assignment context
    class B(object):

        def __init__(self): self.a = self.b = 10
    x = B()
    x.a = 5
    x.b = x.a
    x = {}
    x[1] = 2
    x, y = [1, 2]
    x, y = (1, 2)
    [x, y] = (1, 2)
    [x, y] = [1, 2]

    # list of differing lengths
    for i in []:
        print 'hi'
    for i in [1]:
        print 'hi'
    for i in [1, 2]:
        print 'hi'
    for i in [{1: 2}]:
        print 'hi'

    print re.compile('.*')

    print 'hi'
    print 'TARGET CODE 2'
    print {1: 2}[1]
    y = [1, 2, 3]
    z = y[1:]
    z = y[:1]
    z = y[1:2]
    z = y[1:2:1]

    print 'hi' * 10

    # changing "type" over time
    for i in ('string', 1):
        print i

# test kite_trace on expressions rather than code blocks
kite_trace[repr([1, 1 + 1, 3])]
kite_trace[str(1)]
kite_trace[[1, 2]]
kite_trace[(2, 3, 1 + 5)]
kite_trace[int("5")]
kite_trace[re.compile('.*')]
kite_trace[{1: 2}[1]]

print '\n\n'.join(get_all_traced_ast_reprs(indent='  ', include_field_names=True))
