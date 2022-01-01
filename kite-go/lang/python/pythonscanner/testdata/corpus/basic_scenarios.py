# Test import scenarios
import os
import json
import requests

# from imports
from json import dumps
from os import path

# Test basic completions
x = os.path.join("hello", "world")
x = path.join("hello", "world")

# Type inference
print x.upper()

# Local namespace completions (currently doesn't work)
dumps({"a": 1, "b": 2})

# Test type-inference + completions
r = requests.get("http://www.kite.com/")
body = r.text
print r.status_code

# Test known canonicalization issues
y = numpy.max([1,2,3,4])
