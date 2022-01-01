# # can't resolve things defined in this file for now, since buffer-based symbols aren't yet supported
# def foobar():
#     pass

import main
duplicate_function_alias_alias = main.duplicate_function_alias

import json
dumps_alias = json.dumps

# enc = json.JSONEncoder() # instances don't work
# myInstance = main.MyClass() # instances don't work
main.MyClass()

# x = "foobar" # string literals don't work
