

def test_requests_get():
    import requests

    url = ""
    data = {}
    '''TEST
    resp = requests.get($)
    @. url
    @. `url, params`
    @. `url, headers=dict)`
    status: ok
    '''


def test_requests_get_partial():
    import requests

    url = ""
    data = {}
    # The @0 is a bit strange, does it comes from EmptyCall?
    # Also this test got deactivated in subtoken-decoder branch because of race condition
    '''TEST
    resp = requests.get($
    @. ) …)
    @. url)
    status: ok

    '''


def test_requests_post():
    import requests

    url = ""
    data = {}

    '''TEST
    resp = requests.post($)
    @. `url, data`
    @. url
    @. `url, auth=data`
    status: ok
    '''


def test_matplotlib_plot():
    import numpy as np
    import matplotlib.pyplot as plt

    x = np.linspace(-1, 1)
    y = np.sin(x)

    '''TEST
    plt.plot($)
    @0 y
    @1 `x, y`
    @2 `x, y, label=str`
    @! `x, x`
    @! `y, y`
    @! `y, x`
    status: fail
    '''


def test_numpy_linspace():
    import numpy as np

    start = -1
    stop = 1
    num = 10

    '''TEST
    np.linspace($)
    @. `start, stop`
    @. `start, stop, num)`
    @. `start`
    @. `start, stop, num, endpoint=bool)`
    status: ok
    '''


def test_matplotlib_savefig():
    import matplotlib.pyplot as plt

    filename = ""
    title = ""

    '''TEST
    plt.savefig($)
    @. `filename, title`
    @. filename
    status: ok
    '''


def test_matplotlib_title():
    import matplotlib.pyplot as plt

    title = ""

    '''TEST
    plt.title($)
    @. title
    @. `label, fontdict`
    status: ok
    '''


def test_django_render():
    from django.shortcuts import render

    def my_render(req, name):
        '''TEST
        render($)
        @. `req, name`
        @. req
        status: ok
        '''


def test_csv_reader():
    import csv

    filename = ""
    delimiter = ""
    # The GGNN doesn't seem to support csv.reader
    # cf issue https://github.com/kiteco/kiteco/issues/9778
    '''TEST
    r = csv.reader($)

    @0 filename
    @1 `filename, delimiter=delimiter`
    status: fail
    '''


def test_json_dump():
    import json
    obj = {}
    file = open()

    # Bug grouper, the 2 and 3 should be merged but currently the extra closing parenthesis in 3 get the grouper lost
    '''TEST
    json.dump($)
    @. `obj, file`
    @. obj
    @. `obj, fp)`
    @. `obj, fp, indent=int)`
    @! `obj, obj`
    status: fail
    '''


def test_json_dumps():
    import json
    obj = {}

    '''TEST
    s = json.dumps($)
    @. obj
    @. `obj, ensure_ascii=obj`
    status: ok
    '''
