def test_no_elision_on_call():
    import requests
    '''TEST
    requests.get$
    @. get()
    @. get(url)
    @! ... …(url)
    status: ok
    '''
