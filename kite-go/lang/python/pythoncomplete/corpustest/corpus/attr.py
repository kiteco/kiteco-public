def test_request_get():
    import requests
    url = ""
    '''TEST
    resp = requests.$
    @. get()
    status: ok
    '''


def test_json_dumps():
    import json
    obj = {}
    '''TEST
    s = json.$
    @. dumps()
    @. loads()
    @. load()
    @. dump()
    status: ok
    '''


def test_json_dump():
    import json
    obj = {}
    f = open()

    '''TEST
    json.$
    @0 `dump(obj, f)`
    @1 dump(obj)
    @2 dump()
    @. dumps()
    @. loads()
    @. load()
    status: fail
    '''


def test_dict_items():
    d = {}

    '''TEST
    for k, v in d.$
    @. get()
    @. keys()
    @. items()
    status: ok
    '''


def test_sys_exit():
    import sys

    # TODO(juan) @0 exit()
    # after fixing attr model.
    if some_condition:
        '''TEST
        sys.$
        @. exit()
        status: ok
        '''


def test_time_time():
    import time

    '''TEST
    cur_time = time.$

    @. time()
    status: ok
    '''
    # TODO(juan) @1 localtime()
    # after fixing attr model.


def test_matplotlib_pyplot_plot():
    import matplotlib.pyplot as plt

    x = [1, 2, 3]
    y = [4, 5, 6]

    '''TEST
    plt.p$
    @. plot()
    status: ok
    '''

def test_matplotlib_pyplot_plot_slow():
    import matplotlib.pyplot as plt

    x = [1,2,3]
    y = [4,5,6]

    '''TEST
    plt.$
    @. plot()
    status: slow
    '''
def test_sqlalchemy():
    import sqlalchemy

    '''TEST
    sqlalchemy.$
    @. sql
    @. orm
    @. Column Column
    status: ok
    '''
