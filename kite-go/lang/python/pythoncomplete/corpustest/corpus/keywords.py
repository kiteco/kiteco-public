def test_else_block():
    a = 5
    s = "blibli"

    if s != "blabla":
        print("ok")

    '''TEST
    els$
    @. else:
    @! else
    status: ok
    '''


def test_else_inline():
    a = 5
    s = "blibli"

    '''TEST
    result = "bloblo" if s != "blabla" els$
    @. `else `
    @! else:
    status: ok
    '''

