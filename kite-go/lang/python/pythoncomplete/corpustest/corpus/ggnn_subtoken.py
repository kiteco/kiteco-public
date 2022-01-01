def test_partial_get():
    import requests
    my_url = "www.world-wide-web.wa"
    my_data = {'content':"blip"}

    '''TEST
    resp = requests.get($
    @. )
    @. my_url
    @. `my_url, params)`
    status: ok
    '''

def test_ggnn_partial_from_attr():
    import requests
    my_url = "www.world-wide-web.wa"
    my_data = {'content':"blip"}

    '''TEST
    resp = requests.g$
    @. get(my_url) get(my_url
    @. `get(my_url, params)`
    @. `get(my_url, auth=OAuth1)`
    status: ok
    '''

def test_ggnn_partial_no_comma_before():
    import requests
    my_url = "www.world-wide-web.wa"
    my_data = {'content':"blip"}
    # We don't get any completion from ggnn for this one
    # Currently GGNN is not triggered if an argument is partially typed (or fully, we don't make a difference)
    # GGNN is only triggered right after the opening parenthesis or after a comma
    '''TEST
    resp = requests.post(my_url$
    @0 my_url
    status: ok
    '''


def test_ggnn_partial_comma_before():
    import requests
    my_url = "www.world-wide-web.wa"
    my_data = {'content':"blip"}

    # fail because of empty call flooding
    # and it seems the comma is actually added
    # Empty call flooding : https://github.com/kiteco/kiteco/issues/9654
    '''TEST
    resp = requests.post(my_url,$
    @0 ) ‸)
    @1 get(url,headers=dict …(url,headers=dict
    @10 get(my_url …(my_url
    status: fail
    '''


def test_ggnn_partial_no_comma_before_parent_present():
    import requests
    my_url = "www.world-wide-web.wa"
    my_data = {'content':"blip"}

    '''TEST
    resp = requests.post(my_url, $)
    @. `data, json=my_data`
    @. data=my_data
    status: ok
    '''
