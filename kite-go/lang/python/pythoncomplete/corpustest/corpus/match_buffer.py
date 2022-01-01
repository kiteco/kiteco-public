def test_exact_match():
    import json

    def pretty_print():
        data = {'foo':'bar'}
        num_spaces = 2
        '''TEST
        output = json.dumps(data, indent=$num_spaces$)
        @. num_spaces
        status: ok
        '''

def test_partial_match():
    import requests
    uri = "bla"
    up = "baa"
    '''TEST
    resp = requests.get(url=$uri$)
    @. uri
    status: ok
    '''
