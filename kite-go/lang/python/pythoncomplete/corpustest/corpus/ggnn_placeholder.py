
def test_dict_pop():
    open_mode="rb"

    def readFile(filename:str, counter):
        '''TEST
            with open($[filename]$,[mode])
        @0 ... filename str
        @1 ... counter
        status: fail
        '''
