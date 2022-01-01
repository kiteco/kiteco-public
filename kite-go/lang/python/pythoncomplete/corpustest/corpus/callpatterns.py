def test_format():
    '''TEST
    "".format($)
    @. name)
    status: ok
    '''


def test_list_append():
    '''TEST
    [].append($)
    @0 object)
    status: ok
    '''


def test_dict_get():
    '''TEST
    {}.get($)
    @0 key)
    @1 `key, default)`
    status: ok
    '''


def test_dict_pop():
    '''TEST
    {}.pop($)
    @0 `k, d)`
    @1 k)
    status: ok
    '''


def test_json_dumps_order():
    import json
    '''TEST
    json.dumps($)
    @0 obj)
    @1 `obj, indent=int)`
    status: ok
    '''


def test_json_dump_call_pattern_limit():
    # make sure we don't reconsider completions and bypass the limit
    import json
    '''TEST
    json.dump$
    @. dump() dump(…)
    @. `dump(obj, fp)`
    @. `dump(obj, fp, indent=int)`
    @. dumps()
    status: ok
    '''
