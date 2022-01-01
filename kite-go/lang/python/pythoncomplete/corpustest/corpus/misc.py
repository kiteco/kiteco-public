def test_import_dots():
    '''TEST
    import numpy as n$
    @0 ... np
    status: ok
    '''

def test_import_alias():
    # This test also tests having multiple back-quoted blocks in one expected completion
    '''TEST
    import num$
    @0 `numpy as np` `numpy as np`
    @! numpy
    status: ok
    '''

def test_noarg_placeholder():
    import datetime
    '''TEST
    t = datetime.datetime.utcn$
    @0 utcnow() utcnow()
    status: ok
    '''

def test_arg_placeholder():
    import datetime
    '''TEST
    t = datetime.datetime.now$
    @. now() now(…)
    @. now(tz)
    status: ok
    '''

def test_noarg_SourceFunction():
    class Foo:
        def func(self):
            pass

    foo = Foo()
    '''TEST
    foo.fun$
    @0 func() func()
    @! func() func(…)
    status: ok
    '''
