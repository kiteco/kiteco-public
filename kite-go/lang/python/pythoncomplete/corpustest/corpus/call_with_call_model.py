

def test_requests_get():
    import requests

    url = ""
    data = {}
    # Call model broken
    '''TEST
    resp = requests.get($)
    @. url
    @. `url, data)`
    @. `url, headers=dict)`
    status: ok
    '''


def test_requests_get_partial():
    import requests

    url = ""
    data = {}

    # Call model broken (see equivalent test for subtoken decoder for expected results)
    '''TEST
    resp = requests.get($
    @. )
    @. url)
    @. `url, params=dict)`
    @. `url, headers=dict)`
    status: ok

    '''


def test_requests_post():
    import requests

    url = ""
    data = {}

    # Call model broken (see equivalent test for subtoken decoder for expected results)
    '''TEST
    resp = requests.post($)
    @. url
    @. `url, data`
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


def test_matplotlib_savefig():
    import matplotlib.pyplot as plt

    filename = ""
    title = ""

    # Call model broken (see equivalent test for subtoken decoder for expected results)
    # TODO: Check why filename= is present, it doesn't seem to be in savefig prototype (first arg is fname)
    # https://matplotlib.org/api/_as_gen/matplotlib.pyplot.savefig.html
    '''TEST
    plt.savefig($)
    @. filename
    @. `filename, title)`
    @. `filename, dpi=int)`
    status: ok
    '''


def test_matplotlib_title():
    import matplotlib.pyplot as plt

    filename = ""
    title = ""

    # Call model broken (see equivalent test for subtoken decoder for expected results)
    '''TEST
    plt.title($)
    @. title
    @. label)
    @. `label, fontsize=int)`
    status: ok
    '''


def test_django_render():
    from django.shortcuts import render

    def my_render(req, name):

        # Call model broken (see equivalent test for subtoken decoder for expected results)
        '''TEST
        render($)
        @. req
        @. `req, template_name)`
        @. `request, template_name, context)`
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

    # Call model broken (see equivalent test for subtoken decoder for expected results)
    '''TEST
    json.dump($)
    @. `obj, file`
    @. `obj, fp, indent=int)`
    @! `obj, obj`
    status: ok
    '''


def test_json_dumps():
    import json
    obj = {}
    # Call model broken (see equivalent test for subtoken decoder for expected results)
    '''TEST
    s = json.dumps($)
    @. obj
    @. `obj, obj)`
    @. `obj, indent=int)`
    @. `obj, cls=DjangoJSONEncoder)`
    status: ok
    '''
