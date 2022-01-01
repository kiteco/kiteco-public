# Description
This package contains types used to represent corpus test cases and their expected completions results. It also contains code common to parsing the test case description format used by [python](../../../kite-go/lang/python/pythoncomplete/corpustest/README.md) and [lexical](../../../kite-go/lang/lexical/lexicalcomplete/corpustest/README.md) and running the actual corpus tests.
NOTE: this does not currently support placeholders

# Test Cases Description Format
In the case of semantic completions (`python`), the test case description is contained in a python multiline string. For for lexical completions, it is defined using single-line comments.

Test case descriptions have the following format:
```
<FUNC_DEF>
    <CONTEXT_1>

    *START TEST CASE DELIMITED*
    TEST
    <PRE_LINE>$<POST_LINE>
    <RESULT>+
    status: ok/slow/fail
    *END TEST CASE DELIMITED*

    <CONTEXT_2>
```
Where
- `<FUNC_DEF>` includes the name of the test according to [python](../../../kite-go/lang/python/pythoncomplete/corpustest/README.md) or [lexical](../../../kite-go/lang/lexical/lexicalcomplete/corpustest/README.md) rules, and the test function must be at the top level 
  of the module
- `<CONTEXT_1>` and `<CONTEXT_2>` define arbitrary code that is used to set up context for the 
  prediction task, either or both can be empty
- the `'''TEST...'''` snippet defines the actual information about the test
  - `<PRE_LINE>` defines the line content before the cursor location where we are making a prediction
    and `<POST_LINE>` defines the line content after the cursor location where we are making a prediciton,
    either or both can be empty. The content `<PRE_LINE>$<POST_LINE>` must NOT contain any newlines otherwise
    we can get weird indentation issues when we try to insert the line content into the rest of the function
    body as code (see below).
  
  - The `<RESULT>` component defines the expected results, there must be atleast one of these, possibly more.
  - `<RESULT> := @<RANK> <EXPECTED> <EXPECTED DISPLAY>? <EXPECTED HINT>? | @EXACT | @! <EXPECTED>` where:
    - the `@<RANK>` component defines the expected rank that we should see the `<EXPECTED>` result in the completions
      list. There can be multiple of these 
    - the `<EXPECTED>` component defines the actual output we expect to see in the `Insert` field of the resulting completions.
    - the optional `<EXPECTED DISPLAY>` component defines the expected in the `Display` field of the resulting completion.
    - the optional `<EXPECTED HINT>` component defines the expected in the `Hint` field of the resulting completion.
    - the `@EXACT` notation is used to indicate that no other results should be shown besides the expected ones
    - the `@! <EXPECTED>` notation is used to make sure none of the provided completions have the `<EXPECTED>` insert text
    - use `` (backquote) to delimit completions containing whitespaces
  
  - the `status` component determines how we should interpret the results of running the test case:
    - a `status: ok` indicates that if the test fails then we should trigger a failed CI build
    - a `status: fail` means that we will not trigger a failed CI build

A code snippet for testing is constructed from the above test case by:
    - gathering the `<PRE_LINE>$<POST_LINE>` content and removing the `$` and placing the resulting line
      such that the beginning of the line of inserted code is at the SAME position as the 
      beginning of test case description (`TEST...`), the description itself
      is removed to avoid corrupting the prediction context. Because of this insertion procedure
      it is critical that the string constant defining the test case be placed at the correct indendation
      level and that it not contain new lines.
    - placing the cursor at the appropriate position in the function body after placing the completion line
      as described above and removing the description.

