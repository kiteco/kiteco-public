def test_post_only_url():
    import requests

    url = ""

    '''TEST
    resp = requests.post($)
    @. url
    @. `url, data)`
    @. `url, data=dict, headers=dict)`
    status: ok
    '''


def test_post_only_data():
    import requests

    data = {}

    '''TEST
    resp = requests.post($)
    @. `url, data)`
    @. data)
    @. `url, data=dict, headers=dict)`
    @. `url, json=dict)`
    status: ok
    '''


def test_post_with_noise():
    import requests

    url = ""
    data = {}
    counter = 5
    flag = True

    '''TEST
    resp = requests.post($)
    @. url
    @. `url, data)`
    @. `url, data=dict, headers=dict)`
    status: ok
    '''


def test_post_with_noise_no_data():
    import requests

    url = ""
    counter = 5
    flag = True

    '''TEST
    resp = requests.post($)
    @. url
    @. `url, data)`
    status: ok
    '''


def test_existing_variable():
    abc = 'abc'
    xyz = 'xyz'

    '''TEST
    foo($xyz$)
    @. xyz
    status: ok
    '''
