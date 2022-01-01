def test_post_only_url():
    import requests

    url = ""

    '''TEST
    resp = requests.post($)
    @. `url, data`
    @. url
    status: ok
    '''


def test_post_only_data():
    import requests

    data = {}

    '''TEST
    resp = requests.post($)
    @. `url, data`
    @. `url, auth=data`
    @! data
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
    @. `url, data`
    @. url
    status: ok
    '''


def test_post_with_noise_no_data():
    import requests

    url = ""
    counter = 5
    flag = True

    '''TEST
    resp = requests.post($)
    @. `url, data`
    @. url
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
