# Description
This package contains corpus based integration tests
NOTE: this does not currently support placeholders

# Directories
- `./` contains the go files that actuall run the tests defined in `./corpus`
- `./corpus` contains `.py` files which define test cases

# Test Cases
Test cases are defined in a `.py` file and have the following format:

```
def test_*():
    <CONTEXT_1>

    '''
    <TEST_CASE_DESCRIPTION>
    '''

    <CONTEXT_2>
```
Where
- `test_*` is the name of the test, and the `test_*` function must be at the top level 
  of the module
- `<CONTEXT_1>` and `<CONTEXT_2>` define arbitrary python code that is used to set up context for the prediction task, either or both can be empty 
- `<TEST_CASE_DESCRIPTION>` follows the format described in the [corpustests library](../../../../../kite-golib/complete/corpustests/README.md)

An example snippet is:
```
def test_requests_get():
    import requests

    url = ""
    body = {}

    '''TEST
    resp = r$
    @2 requests.get(url)
    status: ok
    '''
```
Where
- `<CONTEXT_1>` is:
```
    import requests

    url = ""
    body = {}
```
- `CONTEXT_2>` is empty
- `<PRE_LINE>` is `resp = r`
- `<POST_LINE>` is empty
- `<RANK`> is `2`
- `<EXPECTED>` is `requests.get(url)`
