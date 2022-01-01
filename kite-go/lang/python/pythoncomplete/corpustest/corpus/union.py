def test_union_completions():
    import requests
    resp = requests.get("www.impots.gouv.fr")
    '''TEST
    resp.$
    @. headers
    @. status_code
    status: fail
    '''

def test_restricted_union_completions():
    import requests
    resp = requests.get("www.impots.gouv.fr")
    k = resp.keys()
    '''TEST
    resp.$
    @. get()
    @. copy()
    @. keys()
    @. items()
    @. values()
    status: ok
    '''
