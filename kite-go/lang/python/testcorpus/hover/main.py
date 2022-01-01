import os.path
from requests import get


get_alias = get


def duplicate_function():
    x = "test"
    y = x.join(["foo", "bar", "baz"])
    # (z for z in y if z != x) # names defined in generators don't work
    f = lambda x: x
    # f(1) # names defined in lambdas don't work, unless the name's value is nil


def duplicate_function(var):
    return os.path.join("/foo", "bar/")


duplicate_function_alias = duplicate_function
result = duplicate_function_alias(None)
True, False
print(result.split())

class MyClass(object):
    def __init__(self, x):
        self.x = x

    def update(self, x):
        self.x = x

    def get(self):
        return self.x

c = MyClass(10)
print(c.get())
c.update("string")
print(c.get())
