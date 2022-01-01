def test_else_block():
    a = 5
    s = "blibli"

    if s != "blabla":
        print("ok")

    '''TEST
    els$
    @0 else:
    status: ok
    '''


def test_import_keyword():
    import os
    '''TEST
    imp$
    @0 `import `
    status: ok
    '''


def test_no_import_as():
    import os
    '''TEST
    import nump$
    @0 numpy
    @! `numpy as np`
    status: ok
    '''


def test_only_empty_call():
    import requests

    my_url = "www.google.fr"

    '''TEST
    requests.ge$
    @0 get()
    @! get(<url>)
    @! get(my_url)
    @! get(<my_url>)
    status: ok
    '''

