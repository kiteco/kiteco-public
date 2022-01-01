def numpy_sin():
    import numpy as np

    import matplotlib.pyplot as plt

    start = -1
    end = 1

    x = np.linspace(start, end)

    '''TEST
    y = np.sin($)
    @! end
    @! start
    status: ok
    '''

def test_requests_request():
    import requests

    url = 'https://foo.bar'
    payload = {'foo':'bar'}
    headers = None

    # This one is provided as first result after the last changes of the filtering model
    # We will need to fix that, cf github issue : https://github.com/kiteco/kiteco/issues/9562

    '''TEST
    resp = requests.request($)
    @! `url, url`
    @! `url, payload`
    @! `payload, url`
    status: ok
    '''

def test_pandas_from_records():
    import pandas as pd

    my_df = pd.read_csv("test.csv")

    '''TEST
    my_df.from_records($)
    @! `my_df, my_df`
    status: ok
    '''

def test_json_dumps():
    import json
    obj = {}

    '''TEST
    s = json.dumps($)
    @! `obj, obj`
    @! `obj, indent=obj`
    status: ok
    '''

def test_json_dump_file():
    import json
    obj = {}
    file = open()

    # obj,obj still shows up
    '''TEST
    json.dump($)
    @! `obj, obj`
    status: ok
    '''

def test_json_dump():
    import json
    obj = {}

    '''TEST
    json.dump($)
    @! `obj, obj`
    status: ok
    '''

def test_requests_get():
    import requests

    url = ""
    data = {}

    '''TEST
    resp = requests.get($)
    @! data
    @! `url, params=url`
    @! `data, headers=url`
    @! `url, timeout=url`
    @! `data, params=data`
    @! `url, headers=url`
    @! `url, timeout=data`
    @! `data, params=url`
    @! `data, timeout=url`
    status: ok
    '''

def test_requests_post():
    import requests

    url = ""
    data = {}

    '''TEST
    resp = requests.post($)
    @! data
    @! `data, data=data`
    @! `data, headers=data`
    @! `data, data=url`
    @! `data, headers=url`
    @! `url, data=url`
    @! `url, headers=url`
    @! `url, json=url`
    @! `data, json=data`
    @! `data, json=url`
    status: ok
    '''
