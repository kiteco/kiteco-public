def test_empty_call():
    import requests
    url = "www.google.fr"

    # Flack test, alternate between  get(url,[params]) and  get([url],url)
    '''TEST
    requests.get$
    @0 get
    @1 get(url,[params]) get(<url>,<params>)
    @! get()
    status: fail
    '''


def test_empty_call_on_type():
    class ClassA(object):
        def __init__(self):
            print("That a really nice class, of Class A, at least!!!")
    '''TEST
    class MyClass(Cla$
    @0 ClassA
    @! ClassA()
    
    status: ok
    
    '''
