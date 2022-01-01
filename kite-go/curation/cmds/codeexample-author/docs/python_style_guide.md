<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Style Guide: Code Examples](#style-guide-code-examples)
  - [Introduction](#introduction)
    - [Goals: Readability, Concision, Simplicity, Consistency](#goals-readability-concision-simplicity-consistency)
  - [Basics](#basics)
    - [Follow PEP8 as a basic coding style guideline](#follow-pep8-as-a-basic-coding-style-guideline)
    - [Show only one concept per example](#show-only-one-concept-per-example)
    - [Duplicate code instead of using loops or extra variables](#duplicate-code-instead-of-using-loops-or-extra-variables)
    - [Minimize look-back distance](#minimize-look-back-distance)
    - [Use newlines only to separate functionality, and do not separate outputs](#use-newlines-only-to-separate-functionality-and-do-not-separate-outputs)
    - [Preserve structure across related code examples](#preserve-structure-across-related-code-examples)
    - [Choose which examples to write using Kite usage metrics, official documentation, then Google](#choose-which-examples-to-write-using-kite-usage-metrics-official-documentation-then-google)
  - [Titles](#titles)
    - [Start the title with a verb](#start-the-title-with-a-verb)
    - [Use verb roots instead of other verb forms](#use-verb-roots-instead-of-other-verb-forms)
    - [Capitalize only the first letter of the first word](#capitalize-only-the-first-letter-of-the-first-word)
    - [Use a verb phrase that captures the main behavior of the function](#use-a-verb-phrase-that-captures-the-main-behavior-of-the-function)
    - [Construct short but comprehensive titles](#construct-short-but-comprehensive-titles)
    - [Construct titles that reflect what a user might query](#construct-titles-that-reflect-what-a-user-might-query)
    - [Use proper English for titles](#use-proper-english-for-titles)
    - [Use verbs that are most commonly used with a given noun](#use-verbs-that-are-most-commonly-used-with-a-given-noun)
    - [Use the specification to describe various ways to use the function.](#use-the-specification-to-describe-various-ways-to-use-the-function)
    - [Don't include specification phrases for incidental complexity](#dont-include-specification-phrases-for-incidental-complexity)
    - [When a value in the title is essential, explicitly include the values; otherwise generalize](#when-a-value-in-the-title-is-essential-explicitly-include-the-values-otherwise-generalize)
    - [Spell out terminologies](#spell-out-terminologies)
    - [Avoid using parentheses](#avoid-using-parentheses)
    - [Avoid unnecessary prepositional phrases](#avoid-unnecessary-prepositional-phrases)
    - [Avoid using redundant `object` or `instance`](#avoid-using-redundant-object-or-instance)
    - [Avoid all apostrophes, either for contractions or for the possessive case](#avoid-all-apostrophes-either-for-contractions-or-for-the-possessive-case)
    - [Link the title to the code as much as possible](#link-the-title-to-the-code-as-much-as-possible)
    - [Use the suggested terminology if there is not a more appropriate term](#use-the-suggested-terminology-if-there-is-not-a-more-appropriate-term)
    - [Use consistent title structure across examples](#use-consistent-title-structure-across-examples)
    - [Use consistent vocabulary for interchangeable object types](#use-consistent-vocabulary-for-interchangeable-object-types)
    - [Use terms appropriate for the demonstrated concept](#use-terms-appropriate-for-the-demonstrated-concept)
    - [Use the articles "a" and "an" for general behavior, and "the" for specific behavior](#use-the-articles-a-and-an-for-general-behavior-and-the-for-specific-behavior)
    - [Use plurals to generalize behavior that applies to multiple objects](#use-plurals-to-generalize-behavior-that-applies-to-multiple-objects)
    - [Titles should never be duplicates](#titles-should-never-be-duplicates)
    - [Use backticks (`) to specify terminology that is not used as natural language](#use-backticks--to-specify-terminology-that-is-not-used-as-natural-language)
    - [Don't blend identifiers into natural language](#dont-blend-identifiers-into-natural-language)
    - [Prefer to use digits rather than spell out numbers](#prefer-to-use-digits-rather-than-spell-out-numbers)
    - [Use acronyms if they are typically used as such; otherwise, spell them out with each word capitalized](#use-acronyms-if-they-are-typically-used-as-such-otherwise-spell-them-out-with-each-word-capitalized)
    - [Do not put a period at the end of titles](#do-not-put-a-period-at-the-end-of-titles)
  - [Preludes and postludes](#preludes-and-postludes)
    - [Use the prelude and postlude for code that isn't directly relevant to the demonstrated concept](#use-the-prelude-and-postlude-for-code-that-isnt-directly-relevant-to-the-demonstrated-concept)
    - [Put import statements in the prelude](#put-import-statements-in-the-prelude)
    - [Use `import x` syntax by default](#use-import-x-syntax-by-default)
    - [Use `from a.b.c import d` when there are multiple subpackages](#use-from-abc-import-d-when-there-are-multiple-subpackages)
    - [In general, prefer to put code in the main code section](#in-general-prefer-to-put-code-in-the-main-code-section)
  - [Variables](#variables)
    - [Use concise and purposeful variable names](#use-concise-and-purposeful-variable-names)
    - [Avoid variables like `name` or `file` that could be confused with part of the API](#avoid-variables-like-name-or-file-that-could-be-confused-with-part-of-the-api)
    - [Follow language conventions for separating words in a variable name](#follow-language-conventions-for-separating-words-in-a-variable-name)
    - [Don't create variables that are only referenced once](#dont-create-variables-that-are-only-referenced-once)
    - [But do introduce temporary variables rather than split expressions across lines](#but-do-introduce-temporary-variables-rather-than-split-expressions-across-lines)
    - [Introduce temporary variables when the meaning of a value is not clear](#introduce-temporary-variables-when-the-meaning-of-a-value-is-not-clear)
  - [Values and Placeholders](#values-and-placeholders)
    - [Use simple placeholders](#use-simple-placeholders)
    - [When appropriate, use placeholders that are relevant to the package](#when-appropriate-use-placeholders-that-are-relevant-to-the-package)
    - [Minimize placeholders to only what is necessary](#minimize-placeholders-to-only-what-is-necessary)
    - [Use double quotes for string literals by default](#use-double-quotes-for-string-literals-by-default)
    - [Switch to single quotes if you need to include double-quotes inside a string](#switch-to-single-quotes-if-you-need-to-include-double-quotes-inside-a-string)
    - [Use triple-double-quotes for multi-line strings](#use-triple-double-quotes-for-multi-line-strings)
    - [Put a new line at the start and end of multi-line strings, if possible](#put-a-new-line-at-the-start-and-end-of-multi-line-strings-if-possible)
    - [Use the alphabet (a, b, c, ...) for string placeholders](#use-the-alphabet-a-b-c--for-string-placeholders)
    - [Use natural numbers (1, 2, 3, ...) for integer placeholders](#use-natural-numbers-1-2-3--for-integer-placeholders)
    - [Use natural numbers with a ".0" suffix for float placeholders](#use-natural-numbers-with-a-0-suffix-for-float-placeholders)
    - [Continue these sequences for hierarchies, sequences, or groups of placeholder content.](#continue-these-sequences-for-hierarchies-sequences-or-groups-of-placeholder-content)
    - [Use strings for dictionary keys](#use-strings-for-dictionary-keys)
    - [Use `C[n]` for placeholder classes and `f[n]` for placeholder functions](#use-cn-for-placeholder-classes-and-fn-for-placeholder-functions)
    - [Use `"/path/to/file"` for directory names](#use-pathtofile-for-directory-names)
    - [Don't use the same value twice unless for the same purpose each time](#dont-use-the-same-value-twice-unless-for-the-same-purpose-each-time)
    - [For placeholder functions and classes, balance "simple" with "natural"](#for-placeholder-functions-and-classes-balance-simple-with-natural)
    - [Use mock.kite.com for examples that demonstrate communication with a server](#use-mockkitecom-for-examples-that-demonstrate-communication-with-a-server)
  - [Files](#files)
    - [Use sample files provided by Kite whenever possible](#use-sample-files-provided-by-kite-whenever-possible)
    - [When an example requires a file to be created, create it in the prelude](#when-an-example-requires-a-file-to-be-created-create-it-in-the-prelude)
    - [Name files following the same guidelines for naming variables](#name-files-following-the-same-guidelines-for-naming-variables)
    - [Open file with a straightforward open](#open-file-with-a-straightforward-open)
    - [Always write to files in the current working directory](#always-write-to-files-in-the-current-working-directory)
  - [Output](#output)
    - [Generate the minimal output needed to clearly demonstrate the concept](#generate-the-minimal-output-needed-to-clearly-demonstrate-the-concept)
    - [Keep print statements as simple as possible](#keep-print-statements-as-simple-as-possible)
    - [Avoid unnecessary output formatting or explanations](#avoid-unnecessary-output-formatting-or-explanations)
    - [Prefer to print lists with a `for` statement](#prefer-to-print-lists-with-a-for-statement)
    - [Whenever possible, produce the same output every time](#whenever-possible-produce-the-same-output-every-time)
    - [Output binary data as raw strings](#output-binary-data-as-raw-strings)
  - [Miscellaneous](#miscellaneous)
    - [Write some examples before titling them](#write-some-examples-before-titling-them)
    - [Don't use advanced concepts unless necessary](#dont-use-advanced-concepts-unless-necessary)
    - [Don't write examples that demonstrate what not to do](#dont-write-examples-that-demonstrate-what-not-to-do)
    - [Don't write examples that simply construct an object](#dont-write-examples-that-simply-construct-an-object)
    - [Don't reimplement default behavior](#dont-reimplement-default-behavior)
    - [Use explicit keyword arguments when there are many arguments to a function](#use-explicit-keyword-arguments-when-there-are-many-arguments-to-a-function)
    - [Use keyword arguments when it is standard practice to do so](#use-keyword-arguments-when-it-is-standard-practice-to-do-so)
    - [Demonstrate the purpose, not simply the functionality](#demonstrate-the-purpose-not-simply-the-functionality)
    - [When copying existing examples, modify them to fit the style guidelines](#when-copying-existing-examples-modify-them-to-fit-the-style-guidelines)
    - [Only use comments when an example cannot be written in a self-explanatory way](#only-use-comments-when-an-example-cannot-be-written-in-a-self-explanatory-way)
    - [Bundle together examples that use the same function with different parameters](#bundle-together-examples-that-use-the-same-function-with-different-parameters)
    - [When using multi-variable assignment with long tuples, print the tuple first](#when-using-multi-variable-assignment-with-long-tuples-print-the-tuple-first)
    - [Use the `isoformat` method to format `datetime` objects, instead of `strftime`](#use-the-isoformat-method-to-format-datetime-objects-instead-of-strftime)
    - [For combinations of similar functions and similar parameters, choose one function as canonical](#for-combinations-of-similar-functions-and-similar-parameters-choose-one-function-as-canonical)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Style Guide: Code Examples

## Introduction

A code example should demonstrate how to use a function, class, or API in a way that is easy and quick to understand. For instance, here is a code example demonstrating how to write a simple object as JSON:

```good
# PRELUDE
import json

# CODE
print json.dumps({"a":1})

# OUTPUT
{"a": 1}
```

This example is excellent because:

- the main code section is short and quick to understand
- it shows one concept and does not try to show other concepts in the same example
- the import statement is in the prelude, because it is necessary to run the example but not central to the example

Here is code example demonstrating how to reverse a `deque` object:

```good
# PRELUDE
import collections

# CODE
d = collections.deque([1, 2, 3])
d.reverse()
print d

# OUTPUT
deque([3, 2, 1])
```

This code example is excellent because

- it is short and can be understood very quickly
- it is easy to match the input `[1, 2, 3]` with the output `[3, 2, 1]`, and so the effect of `reverse()` is clear
- it shows one concept and does not try to show other concepts in the same example
- the import statement is in the prelude, because it is necessary to run the example but not central to the example
- the print statement is in the main code section, so the user can match the print statement with the output

Here is a code example showing how to inspect the arguments of a function:

```good
# PRELUDE
import inspect

# CODE
def f(a, b=2, *args, **kwargs):
    return a

print getargspec(f)

# OUTPUT
ArgSpec(args=['a', 'b'], varargs='args', keywords='kwargs', defaults=(2,))
```

This code example is excellent because

- it is short and can be understood easily
- it shows one concept and does not try to show other concepts in the same example
- it uses simple placeholder variables and argument names
- it demonstrates all of the features of the function

Finally, here is a code example showing how to extract the query string from a request on a Werkzeug server:

```good
# PRELUDE
from werkzeug.wrappers import Request, Response
from werkzeug.serving import run_simple
from os import environ

# CODE
def application(environ, start_response):
    request = Request(environ)
    query = request.args.get("query")

    response = Response("You searched " + query)
    return response(environ, start_response)

# POSTLUDE
port = int(environ.get("PORT", "8000"))
run_simple("0.0.0.0", port, application)
```

This code example is excellent because

- the concept is demonstrated in the context of a minimal Werkzeug application
- the incidental code is limited to only what is necessary, while still showing the context of the example
- the main code section contains only server-related code; imports and running the server are outside of it

### Goals: Readability, Concision, Simplicity, Consistency

- **Readability**: how much working memory does it take to understand the example?
- **Concision**: how long does it take to read the example?
- **Simplicity**: how hard is it to interpolate this example with the reader's current code?
- **Consistency**: how hard is it to pick out the similarities and differences between this and other code examples?

Fortunately these goals are often aligned.

But do not follow these guidelines when they do not make sense.  Instead, use your best judgment about when to bend or break these rules.

In this document, good code examples are shown like this:

```good
print "This is a positive example of how to write good code examples."
```

And negative examples, showing what *not* to do, are shown like this:

```bad
print('This is a negative example showing how NOT to write code examples.')
```


## Basics

This section contains basic guidelines for code examples.

Note that these may vary from traditional coding style guidelines because examples are driven by different principles of construction than those used in software engineering.  For example, clarity and conciseness are important here, but modularity and reusability are not.

### Follow PEP8 as a basic coding style guideline

The standard Python coding conventions used by almost all Python developers are described in a document called PEP8: https://www.Python.org/dev/peps/pep-0008/. The most important of these conventions are listed below:

- use 4 spaces per indentation
- never use wildcard imports (e.g. `from x import *`)
- surround binary operators (`+`, `-`, `=`, etc.) with a space on each side
- do not surround an assignment (`=`) with spaces when used for keyword or default arguments
- do not put spaces immediately inside parenthesis and brackets (e.g. `spam( ham[ 1 ] )`)
- always pair a specific error with an `except` clause; never use the base `Exception` or leave it empty
- use `isinstance` for type comparison
- use the fact that empty sequences evaluate to `False` to check emptiness, instead of checking length

We diverge from PEP8 as follows:
- no two-line spaces around classes and top-level functions
- refer to previews to determine line length (should be 51)
- when splitting long function calls across lines, do not use extra whitespace to align with the opening delimiter
- we have our own set of rules for imports, as described in the prelude section

### Show only one concept per example

This example shows two independent concepts, which is bad:

```bad
data = [1, 2, 3]
print map(lambda x: x*2, data)
print filter(lambda x: x<3, data)
```

Instead, split this into two separate examples:

```good
data = [1, 2, 3]
print map(lambda x: x*2, data)
```

```good
data = [1, 2, 3]
print filter(lambda x: x<3, data)
```

However, use a single example to demonstrate the multiple ways to use a particular function.

```
# TITLE: Retrieve the timezone from a string

# CODE
print gettz()
print gettz("UTC")
print gettz("America/Los Angeles")
```

### Duplicate code instead of using loops or extra variables

It is perfectly fine to copy a line of code two or three times with small modifications. In fact this is often *better* than introducing a loop, because it takes less time for a human to  understand three duplicated lines with small changes than to understand a loop. This is a good example:

```good
d = defaultdict(list)

d["a"].append(1)
d["b"].append(2)
d["c"].append(3)

print d
```

Whereas this example takes longer to understand (despite technically having fewer lines of code):

```bad
d = defaultdict(list)

for i, v in enumerate(["a", "b", "c"]):
    d[v].append(i)

print d
```

### Minimize look-back distance

When referring to variables, try to keep them close to the line where they are used. If a variable is used multiple times, it may be worth replacing them with their literal values, as the more times it is used, the further away it gets from its definition and the more times the user must look back and forth for the value.

```bad
document = "document_a.txt"
print fnmatch(document, "*.txt")
print fnmatch(document, "document_?.txt")
print fnmatch(document, "document_[abc].txt")
print fnmatch(document, "document_[!xyz].txt")
```

```good
print fnmatch("document_a.txt", "*.txt")
print fnmatch("document_a.txt", "document_?.txt")
print fnmatch("document_a.txt", "document_[abc].txt")
print fnmatch("document_a.txt", "document_[!xyz].txt")
```

```bad
a = f.open("a.txt")
b = f.open("b.txt")

a.write("A")
b.write("B")

a.close()
b.close()
```

```good
a = f.open("a.txt")
a.write("A")
a.close()

b = f.open("b.txt")
b.write("B")
b.close()
```

### Use newlines only to separate functionality, and do not separate outputs

Aside from class and function definitions, newlines should only be used to separate a section of functionaily from another to make the example more readable. Do not add new lines before or after print statements, as they will naturally be separated by inline outputs.

```bad
q = Queue.Queue()

q.put("a")

q.put("b")

q.put("c")

print q.get()

print q.get()

print q.get()
```

```good
q = Queue.Queue()
q.put("a")
q.put("b")
q.put("c")

print q.get()
print q.get()
print q.get()
```

```bad
f = open("sample.json")

print json.load(f)
```

```good
f = open("sample.json")
print json.load(f)
```

### Preserve structure across related code examples

For example, these two examples show how to express different recurrence rules using `dateutil.rrule`:

```good
# TITLE: List the dates of the 100th day of every year

# CODE
for date in dateutil.rrule(YEARLY, byyearday=100, count=3):
    print date
```

```good
# TITLE: List the dates of the next 10th week of the year

# CODE
for date in dateutil.rrule(DAILY, byweekno=10, count=3):
    print date
```

Whereas if the second example was written as follows, it would be more difficult to understand the differences:

```bad
# TITLE: List the dates of the next 10th week of the year

# CODE
dates = dateutil.rrule(DAILY, byweekno=10, count=3)
print list(dates)
```

### Choose which examples to write using Kite usage metrics, official documentation, then Google

Typically, you want to prioritize the most popular classes, functions, and subpackages first. The curation tool provides usage metrics, which is a good baseline for evaluating what to cover. Using this in conjunction with the official documentation for the package will generally cover most, if not all of the major use cases. Supplement this with Google and StackOverflow search results.

General guideline:

1. Cover what is on both the curation tool and the quick start/tutorial/overview page of the documentation
2. Cover the rest of what is on the curation tool, until you reach niche or advanced functions
3. Cover the rest of what is on the official documentation, until you reach niche or advanced functions
4. Do searches online and cover any interesting use cases of the remaining content of the package

## Titles

Writing high quality titles is, in many ways, the hardest part of all.  You have to describe all of the essential parts of what the example does in one compact and easy-to-read sentence fragment.

**Template for writing titles: [verb phrase] [(opt.) specification phrase]**

Specification phrases are used to qualify or refine the verb phrase.  They are often prepositional phrases.

Examples of good titles that use the template:

Verb phrase only:
- "Construct a 1D array"
- "Construct a 2D array"

Verb phrase plus specification phrase:
- "Construct an array from a matrix"
- "Construct an array of \`int16\`s"
- "Construct an array of \`float\`s"
- "Construct an array from a matrix of \`int16\`s"

### Start the title with a verb

- Good: "**Multiply** every element of a matrix with a scalar"
- Bad: "*Scalar* multiplication of every element of a matrix"

### Use verb roots instead of other verb forms

Good: "**Construct** an array"
Bad: "*Constructing* an array"
Bad: "*Constructed* an array"

### Capitalize only the first letter of the first word

- Good: "**Construct an array**"
- Bad: "*Construct An Array*"
- Bad: "*construct an array*"

### Use a verb phrase that captures the main behavior of the function

The verb included in the function name is often a good place to start.

```
# Example
import itertools

for i in itertools.count(10):
    print i
    if i > 20: break
```
In this case, "**Count up from 10**" would be a good title

Sometimes the documentation for a function can provide good inspiration for a title.

### Construct short but comprehensive titles

If a word can be removed without changing the meaning or making the title nonsensical, it should be removed.
- Good: "**Construct an array of \`int16\`s**"
- Bad: "*Construct an array with explicit 16-bit integer type*"

### Construct titles that reflect what a user might query

Think about the kinds of queries that users might form to look for the code example, and use the most common query terms in your title.

```
# Example
for date in rrule(WEEKLY, byweekday=MO, count=3):
    print date
```

The above example uses an `rrule` to return the dates of every Monday. So while the example uses terms like `rrule`, `byweekday` and `weekly`, the most typical query for a script like this would contain terms like every, weekly, Monday, week day, and list, so you would want title it something like "**List the dates of every Monday**".

### Use proper English for titles

Do not skip articles or have grammatical, spelling, or capitalization errors.

- Good: "**Get the date of the first Monday of every month**"
- Bad: "*Get first monday of month*"

### Use verbs that are most commonly used with a given noun

- Good: "**Start** a thread"
- Bad: "*Run* a thread"
- Good: "**Compile** a regular expression"
- Bad: "*Build* a regular expression"

### Use the specification to describe various ways to use the function.

For example, the specification can be the input argument type, the secondary behavior of the function, or a particular condition.

Examples:
- Specify the argument type: "Construct an array **of \`int16\`s**"
- Specify the secondary behavior of the function: "Evaluate a template **and ignore missing keys**"
- Specify a particular condition: "Construct a 3D array **of random values between 0 and 1**"

### Don't include specification phrases for incidental complexity

How to determine whether a specification is essential or incidental:

Imagine you are writing a code example based on the title with a specification. The specification is **incidental** if the *same code (or a piece of similar code without substantial change) will be written without or with the specification*.

Example: `numpy.eye(3)`

Candidate titles:
- "Construct an identity matrix of a specified size"
- "Construct an identity matrix of size n"
- "Construct an identity matrix of size 3"
- "Construct an identity matrix"

The last one is preferred because by looking at the title "Construct an identity matrix",
the code you may write may be:

```
numpy.eye(2)
numpy.eye(3)
numpy.eye(4)
```

which are basically the same.

### When a value in the title is essential, explicitly include the values; otherwise generalize

```
# Example
expected = [1.0, 2.0, 3.0, 4.0]
array2 = [1.0, 2.0, 3.0, 4.01]
array3 = [1.0, 2.0, 3.0, 4.1]
try:
    testing.assert_array_almost_equal(array2, expected, 2)
    print "expected and array2 are equal"
    testing.assert_array_almost_equal(array3, expected, 2)
except AssertionError:
    print "AssertionError: expected and array3 are not equal"
```

Candidate titles:
- Bad: "Assert that *two arrays are equal within a given number of decimals*"
- Bad: "Assert that *arrays are equal within n decimals*"
- Good: "Assert that **two arrays are equal within 2 decimals**"

Both the number of arrays passed to the function and the decimal places are essential.

```
# Example
for i in itertools.repeat(10, 3):
    print i
```

Candidate titles:
- Bad: "Repeat *the value 10 three times*"
- Bad: "Repeat *a value n times*"
- Good: "Repeat **a value three times**"

In this case, because `repeat` by definition must repeat some number of times, the number of repetitions is essential; the value that it repeats is not.

```
# Example
print numpy.eye(3, dtype = int16)
```

Candidate titles:
- Bad: "Construct *an identity matrix of a specified type*"
- Bad: "Construct *a 3-by-3 identity matrix of int16s*""
- Good: "Construct **an identity matrix of int16s**""

The core concept being demonstrated here is the ability to specify a data type when initializing an identity matrix; therefore, the dimensions of the matrix is incidental, while the data type is essential.

### Spell out terminologies

- Bad: "Find the *R^2* of a fitted line"
- Good: "Find the **R-squared measure of a least-squares regression analysis**

### Avoid using parentheses

Titles that include parentheses, like "Compute the sum of the second dimension *(rows)* of an array", are unnecessarily verbose. If a title needs parentheses, it can be simplified to not include them.

### Avoid unnecessary prepositional phrases

- Good: "**Evaluate** a template"
- Bad: "*Substitute values into* a template"
- Good: "**Walk** a tree"
- Bad: "*Walk over* a tree"

### Avoid using redundant `object` or `instance`

Everything in Python is an object, so it is unnecessary to specify that something is an object or an instance.

- Bad: "Write a new compressed file *using a GzipFile object*"
- Bad: "Write a new compressed file *using an instance of GzipFile*"
- Good: "Write a new compressed file **using GzipFile**"

This is also true for JSON.

- Bad: "Encode a *dictionary as a JSON object*"
- Good: "Encode a **dictionary as JSON**"

You can, however, use object or instance to refer to Python objects in the general sense

- Good: "**Pickle a class object**"
- Good: "**Construct a Python class instance from a YAML document**"

### Avoid all apostrophes, either for contractions or for the possessive case

- Good: "...**should not**..."
- Bad: "...*shouldn't*..."
- Good: "...**the name of a thread**..."
- Bad: "...*a thread's name*..."

### Link the title to the code as much as possible

```
# Example
numpy.all(my_matrix > 0, axis = 0)
```

- Good: "Determine if every element in a matrix **is larger than 0** by columns"
- Bad: "Determine if every element in a matrix *is positive* by columns"

### Use the suggested terminology if there is not a more appropriate term

The following are preferred default terms that we would like all code examples to use for general cases. If there is not a better term based on the name of the function or what the documentation indicates, use these.

- Use **a** or **an** instead of *one*
- Prefer **construct** over *make, create, return, generate*
- Prefer **compute** over *calculate, return*
- Prefer **convert** over *map, translate*
- Prefer **equal** over *identical*
- Prefer **hex** over *hexadecimal*
- Prefer **get** over *return, print, show, view*, but use more specific words if possible
- Prefer **sequence** over *list, array, iterable*
- Prefer **element** over *item, value*
- Prefer **condition** over *predicate, if-statement*

Additionally, use the following common terms, even though they are not English words:
- **datetime**, not *date time*
- **filename**, not *file name*

### Use consistent title structure across examples

If there are multiple concise ways to express a title, pick one and stick with it.

- Good: "List the dates of Easter each year" and "List the 20th day of each month"
- Bad: "List the dates of Easter each year" and "Calculate the 20th day of each month"

### Use consistent vocabulary for interchangeable object types

For example, use "element" to refer to items of an array. Do not interchange between "element" and "item".

### Use terms appropriate for the demonstrated concept

For example, when writing an example that work with HTTP requests, use "**Send a GET/POST/etc. request...**" instead of "*Request a URL...*" or "*Make a request...*".

### Use the articles "a" and "an" for general behavior, and "the" for specific behavior

```
# Example
print mimetypes.guess_extension("text/html")
print mimetypes.guess_extension("audio/mpeg")
print mimetypes.guess_extension("fake/type")
```

- Good: "Guess **the** file extension from **a** MIME type" - In this example, we are demonstrating that the `guess_extension` function will return a *specific* file extension, and that it works for *various* MIME types. Therefore, we want to use the definite article "the" to refer to file extensions, and the indefinite article "a" to refer to MIME types.
- Bad: "Guess *a* file extension from *a* MIME type" - Implies that we are guessing *some* file extension, not a specific one
- Bad: "Guess *a* file extension from *the* MIME type" - Implies that we are guessing *some* file extension from a *specific* MIME type, which is not what the example shows
- Bad: "Guess *the* file extension from *the* MIME type" - Implies that we are only demonstrating the function for a *specific* MIME type, which is not what the example shows

### Use plurals to generalize behavior that applies to multiple objects

```
# Example
ints = array.array("i", [1, 2, 3])
print ints.pop()
print ints
print ints.pop(0)
print ints
```

- Bad: "Pop *an element* off of an array"
- Good: "Pop **elements** off of an array"

```
# Example
url = "http://mock.kite.com/text"
request = urllib2.Request(url)
request.add_header("custom-header", "header")
print request.header_items()
```

- Bad: "Define *a custom header* for an HTTP request"
- Good: "Define **custom headers** for an HTTP request"

In both examples, even though the example only works with one object at a time, the concept applies to multiple objects, so the plural forms are more appropriate.

Note that we don't pluralize the non-essential part ("array" and "request").

### Titles should never be duplicates

If two examples vary by a small amount, that variation is essential and therefore should be captured in the title.

### Use backticks (`) to specify terminology that is not used as natural language

When using terms such as `int`, `bytearray`, and `defaultdict`, surround the term with backticks. This also applies to terms such as `float` and `for`, which are English words but not used in their English meaning, and class names like `TextCalendar` and `ElementTree`, which are composed of English words but are not themselves English words.

Note that this does not apply to abbreviations; terms like MD5, SHA256, and HMAC should not have backticks around them.

### Don't blend identifiers into natural language

- When there is a concise English word that commonly corresponds to an identified, always use the English word ("**dictionary**" not "*dict*", "**sequence**" not "*list*", "**integer**" not "*int*"; however, use "**float**" instead of "*floating point number*")
- Prefer to not separate, expand, or otherwise modify the identifier into an English word ("**bytearray**" not "*byte array*", "**DictWriter**" not "*dictionary writer*")
- When an identifier is also exactly an English language word, use lower case if the meaning does not become ambiguous ("**Start a process**" not "*Start a Process*")
- If an identifier is exactly an English language word, but is not used in the same meaning or is only loosely related, Match capitalization and surround it in backticks ("**Share a list between processes using a \`Manager\`**", not "*Share a list between processes using a manager*")
- In general, when referring to a class name, match capitalization and use backticks, but when referring to the corresponding abstract concept that happens to be the same word, use lowercase and no backticks.
- When referring to functions in the title, do not include the parenthesis ("**Print a year calendar using \`prcal\`**" not "*Print a year calendar using \`prcal()\`*")

### Prefer to use digits rather than spell out numbers

Use your best judgment on whether to use digits/numerals (e.g. 5) or spell out the word of numbers (e.g. *five*).  Some general guidelines:

If a number is a parameter that is core to the example, use digits.

- Good: "Wrap a string of text to width **30**"
- Bad: "Wrap a string of text to width *thirty*"

If a number expresses something that is typically written with digits (e.g. measurements, dimensions, constants), use digits.

- Good: "Construct a **3-by-5** identity matrix"
- Bad: "Construct a *three-by-five* identity matrix"

If a number is none of the above, and is also less than 10, spell it out.

- Good: "Combine **two** arrays"
- Bad: "Combine *2* arrays"

When in doubt, use digits.

### Use acronyms if they are typically used as such; otherwise, spell them out with each word capitalized

Good:
- HMAC
- CSS
- Recurrent Neural Network
- Daylight Savings Time

Bad:
- LDA (should be Latent Direchlet Allocation)
- RNN (should be Recurrent Neural Network)
- HyperText Markup Language (should be HTML)
- Structured Query Language (should be SQL)

Exceptions:
- use "random number generator", not RNG or Random Number Generator

When in doubt, spell them out.

### Do not put a period at the end of titles

Titles are sentence fragments, not full sentences.


## Preludes and postludes

Code examples are divided into three sections: prelude, main code, and postlude.  These are combined at runtime to form the entire program, but only the main code is shown in the sidebar; the user only sees all three sections if they click on the example.  We therefore put setup and teardown code in the prelude and postlude, respectively, and reserve the main code section for demonstrating the core concept.

### Use the prelude and postlude for code that isn't directly relevant to the demonstrated concept

Preludes and postludes are not visible to a user until they expand a code example, so use them for code that is needed for the example to run, but is not immediately relevant to the core concept of the example.

```bad
# TITLE: Read a basic CSV file

# PRELUDE
import csv

# CODE
with open("sample.csv", "w") as f:
    f.write("a,1\n")
    f.write("b,2\n")

f = open("sample.csv")
csv_reader = csv.reader(f)

for row in csv_reader:
    print row
```

This example begins with setup code which creates `sample.csv`, a file used in the demonstration of `csv.reader`.  The setup code should not be included in the main code section.  A better division of the example would be:

```good
# TITLE: Read a basic CSV file

# PRELUDE
import csv

with open("sample.csv", "w") as f:
    f.write("a,1\n")
    f.write("b,2\n")

# CODE
f = open("sample.csv")
csv_reader = csv.reader(f)

for row in csv_reader:
    print row
```

Thus the Kite sidebar will only show code that opens and reads the file, which is the central concept of this example.

### Put import statements in the prelude

```good
# PRELUDE
import yaml

# CODE
print yaml.dump("abc")
```

### Use `import x` syntax by default

We want examples to be as easy to understand as possible, so for most packages, we want to import at the package level and access its functions from the package, rather than using `from x import y` to import functions directly.

```bad
# PRELUDE
from json import dumps

# CODE
print dumps({"a": 1})
```

```good
# PRELUDE
import json

# CODE
print json.dumps({"a": 1})
```

### Use `from a.b.c import d` when there are multiple subpackages

If a package contains subpackages, accessing them through the top-level package can make the example messy and hard to read. In these cases, it makes more sense to use `from`.

```good
# TITLE: Map a URL to a function using `getattr`

# PRELUDE
from werkzeug.wrappers import Response, Request
from werkzeug.routing import Map, Rule
from werkzeug.exceptions import HTTPException

# CODE
class HelloWorld(object):

    url_map = Map([
        Rule("/home", endpoint="home"),
    ])

    def dispatch_request(self, request):
        url_adapter = self.url_map.bind_to_environ(request.environ)
        try:
            endpoint, values = url_adapter.match()

            # Call the corresponding function by prepending "on_"
            return getattr(self, "on_" + endpoint)(request, **values)
        except HTTPException, e:
            return e

    def on_home(self, request):
        return Response("Hello, World!")

    def wsgi_app(self, environ, start_response):
        request = Request(environ)
        response = self.dispatch_request(request)
        return response(environ, start_response)

    def __call__(self, environ, start_response):
        return self.wsgi_app(environ, start_response)
```

### In general, prefer to put code in the main code section

If your example involves helper classes or methods that are central to the example then you should still include those in the main code section.

```good
# PRELUDE
import yaml

# CODE
class Dice(object):
    def __init__(self, a, b):
        self.a = a
        self.b = b
    def __repr__(self):
        return 'Dice(%d, %d)' % (self.a, self.b)

def dice_constructor(loader, node):
    value = yaml.loader.construct_scalar(node)
    a, b = map(int, value.split('d'))
    return Dice(a, b)

add_constructor('!dice', dice_constructor)

print yaml.load("gold: !dice 10d6")
```


## Variables

### Use concise and purposeful variable names

Use variable names that are short, and describe what a variable is going to be used for. For example, this is good because you can see what the purpose of each variables is:

```good
xdata = np.arange(10)
ydata = np.zeros(10)
plot(xdata, ydata)
```

Whereas this is bad because it's not clear what the variables do:

```bad
a = np.arange(10)
b = np.zeros(10)
plot(a, b)
```

**Avoid 'foo' 'bar' etc.** regardless of how/where you are considering using it.

### Avoid variables like `name` or `file` that could be confused with part of the API

Consider the following example using Jinja2:

```bad
template = Template("<div>{{name}}</div>")
print(template.render(name="abc"))  # unclear - is "name" somehow special?
```

For somebody not familiar with Jinja, it unclear whether `name` has some special meaning in the Jinja2 API, or whether it's used as an arbitrary placeholder. To make it clear, use a word that could not be confused for part of the API:

```good
template = Template("<div>{{person}}</div>")
print(template.render(person="abc"))
```

### Follow language conventions for separating words in a variable name

```good
# Python
my_variable = 1

# Java
int myVariable = 1
```

### Don't create variables that are only referenced once

This is unnecessarily verbose:

```bad
pattern = "abc .* def"
regex = re.compile(pattern)
```

Instead, put it all on one line:

```good
regex = re.compile("abc .* def")
```

### But do introduce temporary variables rather than split expressions across lines

This is difficult to understand:

```bad
yaml.dump({"name": "abc", "age": 7},
     open("myfile.txt", "w"),
     default_flow_style=False)
```

Instead, it would be better to introduce two temporary variables:

```good
data = {"name": "abc", "age": 7}
f = open("myfile.txt", "w")
yaml.dump(data, f, default_flow_style=False)
```

### Introduce temporary variables when the meaning of a value is not clear

This is difficult to understand:

```bad
print np.where([False, True, True], [1, 2, 3], [100, 200, 300])
```

Instead, introduce temporaries to indicate what the variables mean:

```good
condition = [False, True, True]
when_true = [1, 2, 3]
when_false = [100, 200, 300]
print np.where(condition, when_true, when_false)
```

This will sometimes conflict with the rule about not creating variables that are only referenced once. Use your best judgment.


## Values and Placeholders

### Use simple placeholders

This is unnecessarily long:

```bad
print json.dump({"first_name": "Graham", "last_name": "Johnson", "born_in": "Antarctica"})
```

Instead, use more concise data:

```good
print json.dump({"a": 1, "b": 2})
```

**Avoid 'foo' 'bar' etc.** regardless of how/where you are considering using it.

### When appropriate, use placeholders that are relevant to the package

This is simple, but makes no sense in the context of `shlex`:

```bad
print shlex.split("a b")
```

Instead, since `shlex` is used for parsing Unix shell commands, use a sample command:

```good
print shlex.split('tar -cvf kite_source.tar /home/kite/')
```

Similarly, since HMAC is used to hash messages using a key, express those semantics in the placeholders:

```good
h = hmac.new("key")
h.update("Hello, World!")
```

### Minimize placeholders to only what is necessary

**Use the smallest amount of placeholder content that still clearly demonstrates the concept.** You should rarely ever need more than 3.

When choosing between 1, 2, or 3 placeholders, consider the cost of incremental cost:
- For example, in `"abc".upper()`, the last `c`, while not strictly necessary, is low cost
- But `map(upper, ["abc", "def"])` is better than `map(upper, ["abc", "def", "ghi"])` because each incremental item element adds seven characters to the code, gets us to a less familiar part of the alphabet, and adds considerable length to the output
- In the case of `{"a": 1}.pop("a")`, one entry is sufficient because `pop` for dictionaries does not care about order

Be careful when using only one placeholder, as the example may become ambiguous
- if we show `"a".upper()`, it is unclear if it works for multiple characters
- if we show `[1].pop()`, it is unclear if `pop` is getting the first or last item, or even the whole list

Sometimes there are additional reasons to consider using two vs three items. For example, when multiplying two matrices it's required to use non-square dimensions to illustrate how the dimensions need to line up.

### Use double quotes for string literals by default

Rationale: we could have chosen either one, but it's important to have a consistent standard, and double quotes are more consistent with string representations in other languages.

```good
print "use double quotes by default"
```

### Switch to single quotes if you need to include double-quotes inside a string

This is ugly:

```bad
s = "Greg said \"hello\" to Phil"
```

Instead, switch to single quotes:

```good
s = 'Greg said "hello" to Phil'
```

### Use triple-double-quotes for multi-line strings

```good
document = """
{
  "a": 20,
  "b": [1,2,3,"a"],
  "c": {
    "d": [1,2,3],
    "e": 40
  }
}
"""
data = json.loads(document)
```

### Put a new line at the start and end of multi-line strings, if possible

```bad
data = """This is a
little hard to read"""
```

```good
data = """
This is a
lot easier to read
"""
```

### Use the alphabet (a, b, c, ...) for string placeholders

```good
my_string = "abc"
```

### Use natural numbers (1, 2, 3, ...) for integer placeholders

```good
numpy.array([1, 2, 3])
```

### Use natural numbers with a ".0" suffix for float placeholders

```good
my_list = [1.0, 2.0, 3.0]
```

### Continue these sequences for hierarchies, sequences, or groups of placeholder content.

```good
map(upper, ['abc', 'def'])
numpy.array([[1, 2, 3], [4, 5, 6]])
```

### Use strings for dictionary keys

If the key-value pairs are purposeful, use key names that correspond to the meaning of the value. Otherwise, use `"a", "b"...` as placeholder keys, and `1, 2...` as placeholder values.

```bad
json.dumps({1: "a", 2: "b"})
```

```good
json.dumps({"a": 1, "b": 2})
```

### Use `C[n]` for placeholder classes and `f[n]` for placeholder functions

Note that this only applies to *placeholder* classes and functions, i.e. classes and functions that have no functionality or purpose, such as those used in demonstrating `sys` and `inspect` functionality.

```bad
class Dog:
    def bark(self):
        return "Bark bark!"

class Cat:
    def meow(self):
        return "meow"
```

```good
class C:
    pass

class C:
    def f(self):
        pass

class C1:
    def f1(self):
        return 1

class C2:
    def f2(self):
        return 2
```

### Use `"/path/to/file"` for directory names

Again, only for non-purposeful placeholder directory names. Note that there is a `/` at the beginning.

```bad
os.path.split("/home/user/docs/sample.txt")
```

```good
os.path.split("/path/to/file")
```

### Don't use the same value twice unless for the same purpose each time

The following example creates an HMAC hash using a key, then updates it with a value:

```bad
h = hmac.new("abc")
h.update("abc")
```

This is confusing since it leaves the user wondering whether there was some important reason to use `"abc"` in both places. Instead, you should use different values so that there is no confusion:

```good
h = hmac.new("key")
h.update("Hello, World!")
```

On the other hand, sometimes the same value is being used *for the same purpose* in two different places. In this case you *should* use the same value in both cases, for example:

```good
data1 = numpy.zeros(8)
data2 = numpy.zeros(8)
```

### For placeholder functions and classes, balance "simple" with "natural"

This example has a bunch of incidental complexity:

```bad
# TITLE: Add test cases to a suite

# CODE
class MyTest(TestCase):
    def setUp(self):
        self.name = "abc"
        self.num = 123

    def test_name_equals(self):
        self.assertEqual(self.name, "abc")

    def test_num_equals(self):
        self.assertEqual(self.num, 123)

suite = TestSuite()
suite.addTest(MyTest("test_name_equals"))
suite.addTest(MyTest("test_num_equals"))
```

Here is a much simpler version, that forms a better example:

```good
# TITLE: Add test cases to a suite

# CODE
class MyTest(TestCase):
    def test_a(self):
        self.assertTrue(0 < 1)

suite = TestSuite()
suite.addTest(MyTest("test_a"))
```

First, we don't need to use `setUp` or instance variables.  We do have a decision between `assertTrue(True)` or `assertTrue(0 < 1)`.  Here we've decided in favor of the latter, though not strongly.

### Use mock.kite.com for examples that demonstrate communication with a server

A list of endpoints for mock.kite.com can be found [here](https://quip.com/qmbpAwyeKI56).


## Files

### Use sample files provided by Kite whenever possible

A list of sample files accessible from the examples can be found [here](https://quip.com/OCKRAUwgFL4x). The following example shows how to use these files:

```good
# CODE
f = open("sample.txt")
print f.read()

# POSTLUDE
'''
sample_files:
- sample.txt
'''
```

If the provided sample files are not enough, ask your correspondent about creating new sample files before explicitly creating files in new examples.

### When an example requires a file to be created, create it in the prelude

```good
# PRELUDE
import csv

with open("sample.csv", "w") as f:
    f.write("a,1\n")
    f.write("b,2\n")

# CODE
f = open("sample.csv")
csv_reader = csv.reader(f)

for row in csv_reader:
    print row
```

### Name files following the same guidelines for naming variables

File names that appear in the main code section should reflect their purpose, just like variables.

When a file is simple a placeholder file, use a short name with a familiar extension:

- `a.txt`
- `image.png`
- `file.zip`
- `page.html`
- etc.

### Open file with a straightforward open

Unless absolutely necessary, do not use a `with` statement for opening files (rationale: this is a difficult one to decide on but `open` works fine for short examples, and `with` is a language-level feature that some users may not be familiar with).

```good
f = open("input.txt")
```

### Always write to files in the current working directory

```good
f = open("output.txt", "w")
f.write("abc")
```

Never specify an explicit path (this would not run inside the sandbox environment):

```bad
f = open("/path/to/output.txt", "w")
f.write("abc")
```


## Output

### Generate the minimal output needed to clearly demonstrate the concept

Output must be read and understood by the user, to, so the more output there is, the more time it takes users to understand the example.

This code generates 24 lines of output, which is too much:

```bad
# CODE
for x in itertools.permutations([1, 2, 3, 4]):
    print x

# OUTPUT
(1, 2, 3, 4)
(1, 2, 4, 3)
(1, 3, 2, 4)
(1, 3, 4, 2)
(1, 4, 2, 3)
(1, 4, 3, 2)
(2, 1, 3, 4)
(2, 1, 4, 3)
(2, 3, 1, 4)
(2, 3, 4, 1)
(2, 4, 1, 3)
(2, 4, 3, 1)
(3, 1, 2, 4)
(3, 1, 4, 2)
(3, 2, 1, 4)
(3, 2, 4, 1)
(3, 4, 1, 2)
(3, 4, 2, 1)
(4, 1, 2, 3)
(4, 1, 3, 2)
(4, 2, 1, 3)
(4, 2, 3, 1)
(4, 3, 1, 2)
(4, 3, 2, 1)
```

Instead, do the following, which only generates six lines of output:

```good
# CODE
for x in itertools.permutations([1, 2, 3]):
    print x

# OUTPUT
(1, 2, 3)
(1, 3, 2)
(2, 1, 3)
(2, 3, 1)
(3, 1, 2)
(3, 2, 1)
```

However, the following generates *too little* output and does not show the concept clearly:

```bad
# CODE
for x in itertools.permutations([1, 2]):
    print x

# OUTPUT
(1, 2)
(2, 1)
```

### Keep print statements as simple as possible

Always use a simple `print` statement to output values. Note that we use Python 2-style `print value`, not `print(value)`. Don't use statements like these:

```bad
[print item for item in some_list]
```

```bad
print(value1, value2, value3)
```

```bad
print value1, value2, value3
```

```bad
from pprint import pprint
pprint(some_dict)
```

### Avoid unnecessary output formatting or explanations

If you feel the need to add expository, you probably need to simplify your example so that it does not create complex outputs.

```bad
print "This is the value for a: " + a
```

```bad
print "Person {name}: {sex}, age {age}".format(
    name = name,
    sex = sex,
    age = str(age)
)
```

```bad
def print_dict(d):
  output = ""
  for k, v in d.items():
    output += "Key: " + key + " Value: " + value + "\n"

print_dict(some_dict)
```

### Prefer to print lists with a `for` statement

This example requires a more advanced understanding of Python iterators and should be avoided:

```bad
print list(itertools.permutations([1, 2, 3]))
```

Instead, use this syntax:

```good
for x in itertools.permutations([1, 2, 3]):
    print x
```

### Whenever possible, produce the same output every time

When demonstrating functions such as random number generators, set a deterministic seed in the prelude if it is available:

```good
# PRELUDE
import random
random.seed(0)

# CODE
print random.randint(0, 10)
```

This is good because the user will only see the main code section, which will not be cluttered with the call to `random.seed`.

When using `numpy.random`, there is a similar seed function:

```good
# PRELUDE
from numpy.random import randn, seed
seed(0)

# CODE
print randn(2, 5)
```

For examples that involve time stamps, HTTP requests, and random number generators with no seed, this is not possible, which is okay.

### Output binary data as raw strings

Even though this can cause broken characters to appear, we still want to keep print statements as simple as possible.

```good
print binary_data
```

```bad
print repr(binary_data)
```

```bad
print hexlify(binary_data)
```

## Miscellaneous

### Write some examples before titling them

Often it is helpful to write some initial examples to get a sense of a package's classes and functions first. Coming up with titles is easier after you map out the different examples you want to write. Also, titling an example essentially finalizes its contents, and you may miss opportunities to improve the content if you write the title too early.

### Don't use advanced concepts unless necessary

This example works, but may be difficult for beginners who are not familiar with Python's dictionary unpacking syntax:

```bad
data = {"person": "abc", "age": 5}
print "{person} is age {age}".format(**data)
```

Instead, this example is easy for everyone to understand:

```good
print "{person} is age {age}".format(person="abc", age=5)
```

### Don't write examples that demonstrate what not to do

```bad
try:
    print pickle.loads("pickle")
except IndexError:
    print "String does not contain pickle data"
```

However, it is sometimes good to include a demonstration of common failure modes as part of a larger example.

```good
dictionary = {"a": 1}

print dictionary.pop("a")
print dictionary

try:
    print dictionary.pop("a")
except KeyError as e:
    print "KeyError: " + e.message

print dictionary.pop("a", None)
```

### Don't write examples that simply construct an object

Object construction on its own is not informative; it is much more helpful to see how an object is used.

```bad
pattern = re.compile("[a-z]+[0-9]+")
print pattern
```

```good
pattern = re.compile("[a-z]+[0-9]+")
print pattern.match("test123")
```

### Don't reimplement default behavior

Here is a bad example showing a class with an explicit pickling function:

```bad
import cPickle

class Foo(object):
    def __init__(self, value):
        self.value = value
    def __getstate__(self):
        return {'the_value': value}

f = Foo(123)
s = cPickle.dumps(f)
```

The problem with the code example above is that, by default, cPickle uses the `__dict__` attribute whenever there is no `getstate` function. So the code in the example above would have produced the exact same output even if the `getstate` function had been omitted. This is bad because it's not clear to the user why the `getstate` function is important, since the result is exactly what would have happened anyway if `getstate` had been omitted. Instead, a better example would implement `getstate` in a way that is different to the default behavior.

### Use explicit keyword arguments when there are many arguments to a function

```good
xdata = np.arange(10)
ydata = np.zeros(10)
plot(xdata, ydata, style='r-', label='my data')
```

### Use keyword arguments when it is standard practice to do so

This works but is non-standard:

```bad
a = array([1, 2, 3], float)
```

Whereas this is standard practice for `numpy` code:

```good
a = array([1, 2, 3], dtype=float)
```

### Demonstrate the purpose, not simply the functionality

Examples should demonstrate the **purpose** of its functions, not merely their *functionality*. Use functions in a way that mirrors their intended usage, and clearly show the purpose that the function serves.

```bad
print secure_filename("a b")
# OUTPUT: 'a_b'
```

```good
print secure_filename("../../../etc/passwd")
# OUTPUT: 'etc_passwd'
```

### When copying existing examples, modify them to fit the style guidelines

Many packages will provide examples in their official documentation pages, and it's okay to use these as examples. However, they will likely not abide by this style guide as is, so modify them as needed.

### Only use comments when an example cannot be written in a self-explanatory way

Strive to write examples that do not need comments to explain what they do. If an explanation is absolutely necessary, include a brief comment **above** the section that requires explanation; do not use inline comments.

This is obvious and does not need a comment:

```bad
# Dump to string
output_string = csv.dumps([1, 2, 3])
```

This is obvious and is also using an inline comment:

```bad
csv.loads(string) # load from string
```

This comment is helpful to include because the line is confusing on its own, but cannot be written more clearly because `has_header` cannot be called with a keyword argument:

```good
# Sample the first 256 bytes of the file
print sniffer.has_header(f.read(256))
```

### Bundle together examples that use the same function with different parameters

Examples that call the same function with different parameters can generally be bundled together into a "cheat sheet" example that provides users with a quick reference to the usages of the function, *if the function calls all demonstrate the same concept*.

```good
# TITLE: Construct a dictionary

# CODE
print dict(a=1, b=2)
print {"a": 1, "b": 2}
print dict([("a", 1), ("b", 2)])
print dict({"a": 1, "b": 2})
print dict(zip(["a", "b"], [1, 2]))
```

Functions like `dateutil.rrule` are exceptions because they provide conceptually different outputs based on the arguments provided.

### When using multi-variable assignment with long tuples, print the tuple first

This allows users to more easily match up the variables with their values and helps users who are not familiar with the syntax understand what is going on.

```good
st = os.stat(open("sample.txt"))
print st
mode, ino, dev, nlink, uid, gid, size, accessed, modified, created = st
```

### Use the `isoformat` method to format `datetime` objects, instead of `strftime`

`strftime` tends to be long and unwieldy, and is not completely standardized. `isoformat` has a standard output that is decently readable and makes the code example much more concise. Of course, when you can, you should print out the `datetime` object directly, so you can take advantage of the default `repr` method that outputs a nicely-formatted representation of the date and time.

### For combinations of similar functions and similar parameters, choose one function as canonical

When a package has `x` similar functions, each of which take in `y` similar parameters, we don't want to enumerate `x * y` examples that show all the different ways to call each of the functions with each of the parameters. For example, `csv` has two different readers, each of which take in various parameters for reading different delimiters, row indicators, quotes, etc. Showing examples for how to use each of these parameters for both readers would be redundant.

Instead, choose one of the function as the "canonical" example, and only show the different parameters or ways of calling the function for that function. For everything else, just provide one simple example for each function. In the case of `csv`, we would choose one reader and write an example for each of the different parameters, and only provide one simple example of using the other reader.

This is also true for objects - `hashlib` has 5 different hash objects, each of which can be initialized, updated, copied, converted to a hex value, or accessed as its raw binary value. In this case, we would choose one hash to show how to do each of these actions, and for all other hashes, simply have one example of each that shows how to initialize it.
