from kite import Kite

class Man(object):
    def __init__(self):
        self.code = "man"
    def car(self):
        print("star!")

k = Kite()
k.foo()

m = Man()
m.car()

def print_code():
    print(k.code + " " + m.code)

def some_kite():
    return Kite()

print_code()

q = some_kite()