# Description
This package contains corpus based integration tests
NOTE: this does not currently support placeholders

# Directories
- `./` contains the go files that actuall run the tests defined in `./corpus`
- `./corpus/<lang>` contain test files which define test cases (e.g. `./corpus/go` contains `.go` test files). Note: for go, test file names should end in `_test.go` for linting purposes.

# Test Cases
Each test case must contain only one test function. Test cases are defined in a language-specific file and have the following format:

```
<FUNC_DEF>
    <CONTEXT_1>

    <TEST CASE DESCRIPTION>
    
    <CONTEXT_2>
```
Where
- `<FUNC_DEF>` is the language-specific function definition
- `<CONTEXT_1>` and `<CONTEXT_2>` define arbitrary code that defines the context for which completions will be generated, either or both can be empty
- `<TEST_CASE_DESCRIPTION>` follows the format described in the [corpustests library](../../../../../kite-golib/complete/corpustests/README.md) with each line begins with a single-line comment for the language being tested

An example go snippet is:
```
import (
    _ "strings" // Note: make fmt, make vet, make test are all run for this file
)

func testBasic() {
    // TEST
    // strings.Ha$
    // @0 HasPrefix(
    // @1 HasSuffix(
    // status: ok
}
```

Where
- `<CONTEXT_1>` is:
```
import (
    _ "strings"
)
```
- `CONTEXT_2>` is empty
- `<PRE_LINE>` is `strings.Ha`
- `<POST_LINE>` is empty
- `<EXPECTED>` is `HasPrefix` and `HasSuffix`
