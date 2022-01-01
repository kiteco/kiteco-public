package titleparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindAny tests findAny which also tests find.
func TestFindAny(t *testing.T) {
	candidates := []string{"abs", "array", "test"}
	targets := []string{"abs", "sqrt", "test1"}

	assert.Equal(t, true, strList(candidates).findAny(targets))
}

// TestCheckRulesCase1 tests checkRules with an example that violates
// 1. Constructing is used instead of Construct
// 2. one is used instead of a/an
// 3. missing brackets around "of int16"
// 4. missing brackets around "with a predefined array"
// 5. end the title with a period
// 6. 'float' is used in the code but not in the title
func TestCheckRulesCase1(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Constructing one matrix of int16 with a predefined array."
	parse := "(ROOT (S (VP (VBG Constructing) (NP (NP (CD one) (NN matrix)) (PP (IN of) (NP (NN int16)))) (PP (IN with) (NP (DT a) (JJ predefined) (NN array)))) (. .)))"
	tags := parseTags(parse)
	prelude := "from numpy import array as ay"
	code := `a = ay([3, 4, 5], dtype = 'float')`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 4, len(violations))
}

// TestCheckRulesCase2 tests checkRules with an example that violates
// 1. Return is used.
// 2. The parser thinks "Return is a noun", which causes a false linting error.
// 3. one is used instead of a/an
// 4. missing brackets around "of float"
func TestCheckRulesCase2(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Return one matrix of float"
	parse := "(ROOT (NP (NP (NN Return)) (NP (NP (CD one) (NN matrix)) (PP (IN of) (NP (NN float))))))"
	tags := parseTags(parse)
	prelude := "from numpy import array as ay"
	code := `a = ay([3, 4, 5], dtype = 'float')`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 3, len(violations))
}

// TestCheckRulesCase3 tests checkRules with an example that violates
// 1. Use ... to VB ...
func TestCheckRulesCase3(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Use zeros to construct an array"
	parse := "(ROOT (S (NP (NNP Use) (NNS zeros)) (VP (TO to) (VP (VB construct) (NP (DT an) (NN array))))))"
	tags := parseTags(parse)
	code := ""
	prelude := ""

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// TestCheckRulesCase4 tests checkRules with an example that violates nothing.
func TestCheckRulesCase4(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Use a computer [from 5 to 9]"
	parse := "(ROOT (S (VP (VB Use) (S (NP (DT a)) (VP (VB computer) (PP (IN from) (NP (QP (CD 5) (TO to) (CD 9)))))))))"
	tags := parseTags(parse)
	code := ""
	prelude := ""

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase5 tests checkRules with an example that violates
// 1. Missing brackets around 'floats'
func TestCheckRulesCase5(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Construct 1D array of floats"
	parse := "(ROOT (S (VP (VB Construct) (NP (NP (NN 1D) (NN array)) (PP (IN of) (FRAG (VP (VBZ floats))))))))"
	tags := parseTags(parse)
	code := `print(array([2, 3, 4, 5], dtype = 'float'))`
	prelude := "from numpy import array"

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase6 tests checkRules with an example that violates
// 1. 'float' is used as an argument in code, but not mentioned
func TestCheckRulesCase6(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Construct 1D array"
	parse := "(ROOT (S (VP (VB Construct) (NP (NN 1D) (NN array)))))"
	tags := parseTags(parse)
	code := `print(array([2, 3, 4, 5], dtype = 'float'))`
	prelude := "from numpy import array"

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// TestCheckRulesCase7 tests checkRules with an example for which there are
// no violations because the code is wrong.
func TestCheckRulesCase7(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Construct 1D array"
	parse := "(ROOT (S (VP (VB Construct) (NP (NN 1D) (NN array)))))"
	tags := parseTags(parse)
	code := `print(array([2, 3, 4, 5], dtype = 'float')`
	prelude := "from numpy import array"

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase8 tests checkRules with an example that violates
// 1. The title contains more than 1 sentence
func TestCheckRulesCase8(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Construct 1D array. Return the array"
	parse := "(ROOT (S (VP (VB Construct) (NP (NN 1D) (NN array))) (. .))) (ROOT (NP (NP (NN Return)) (NP (DT the) (NN array))))"
	tags := parseTags(parse)
	code := ""
	prelude := ""

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// -- Test cases from the Threading package

// TestCheckRulesCase9 tests checkRules with an example that violates
// 1. should use 'of thread' instead of "thread's"
// 2. Getting is used instead of Get
func TestCheckRulesCase9(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Getting the thread's name"
	parse := "(ROOT (S (VP (VBG Getting) (NP (NP (DT the) (NN thread) (POS 's)) (NN name)))))"
	tags := parseTags(parse)
	code := ""
	prelude := ""

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 2, len(violations))
}

// TestCheckRulesCase10 tests checkRules with an example that violates
// 1. Should use "Import" instead of "Importing"
// 2. Missing brackets around "at package level"
func TestCheckRulesCase10(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Importing at package level"
	parse := "(ROOT (S (VP (VBG Importing) (PP (IN at) (NP (NN package) (NN level))))))"
	tags := parseTags(parse)
	prelude := ""
	code := "import threading"

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// TestCheckRulesCase11 tests checkRules with an example that violates
// 1. Should use "Acquire" instead of "Acquiring"
// 2. Missing brackets around "with a nonblocking flag"
func TestCheckRulesCase11(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Acquiring with a nonblocking flag"
	parse := "(ROOT (S (VP (VBG Acquiring) (PP (IN with) (NP (DT a) (JJ nonblocking) (NN flag))))))"
	tags := parseTags(parse)
	prelude := "from threading import Semaphore"
	code := `semaphore = Semaphore()  # default value of 1
		semaphore.acquire()  # decrements value by 1
		semaphore.acquire(False) 
		print("This will always be printed.")`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// TestCheckRulesCase13 tests checkRules with an example that violates
// 1. Should use "Get" instead of "Getting"
// 2. Missing brackets
// 3. Arguments used in the code are not mentioned in the title
// [todo] should avoid using ','
func TestCheckRulesCase13(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Getting all threads currently running, with running threads"
	parse := "(ROOT (S (VP (VBG Getting) (NP (NP (DT all) (NNS threads)) (ADJP (RB currently) (VBG running)) (, ,) (PP (IN with) (NP (VBG running) (NNS threads)))))))"
	tags := parseTags(parse)
	prelude := "from threading import Thread, enumerate"
	code := `def threaded_count(start, end):
		    for i in range(start, end):
		    # do work
		    print(i)

		Thread(target=threaded_count, args=(0, 10)).start()
		print(enumerate())`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 2, len(violations))
}

// TestCheckRulesCase14 tests checkRules with an example that violates
// 1. Should use "Check" instead of "Checking"
// 2. Args not mentioned in the title
// 3. Missing brackets around [of condition]
func TestCheckRulesCase14(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Checking status of condition"
	parse := "(ROOT (S (VP (VBG Checking) (NP (NP (NN status)) (PP (IN of) (NP (NN condition)))))))"
	tags := parseTags(parse)
	prelude := "from threading import Lock, RLock, Condition, Thread"
	code := `lock = Lock()
		condition = Condition(lock=lock)
		condition.acquire()
		print(condition.locked())`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 2, len(violations))
}

// -- Test cases from inspect

// TestCheckRulesCase15 tests checkRules with an example that violates
// 1. Missing brackets around [of the source file]
func TestCheckRulesCase15(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Get the name of the source file in which an object was defined"
	parse := "(ROOT (S (VP (VB Get) (NP (NP (DT the) (NN name)) (PP (IN of) (NP (DT the) (NN source) (NN file))) (SBAR (WHPP (IN in) (WHNP (WDT which))) (S (NP (DT an) (NN object)) (VP (VBD was) (VP (VBN defined)))))))))"
	tags := parseTags(parse)
	prelude := "from inspect import getsourcefile"
	code := `def f(x, y):
		    """Adds x and y"""
		    return x + y
		print getsourcefile(f)`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase16 tests checkRules with an example that violates
// 1. Missing brackets around [about the arguments]
// 2. Missing brackets around [into a frame]
func TestCheckRulesCase16(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Get information about the arguments passed into a frame"
	parse := "(ROOT (S (VP (VB Get) (SBAR (S (NP (NP (NN information)) (PP (IN about) (NP (DT the) (NNS arguments)))) (VP (VBD passed) (PP (IN into) (NP (DT a) (NN frame)))))))))"
	tags := parseTags(parse)
	prelude := "from inspect import getargvalues, currentframe"
	code := `def f(x, y=2, *args, **kwargs):
		    print getargvalues(currentframe())
		 f(1)`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase17 tests checkRules with an example that violates nothing.
func TestCheckRulesCase17(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Format an argument spec"
	parse := "(ROOT (S (VP (VB Format) (NP (DT an) (NN argument) (NN spec)))))"
	tags := parseTags(parse)
	prelude := "from inspect import formatargspec, getargspec"
	code := `def f(a, b=2, *args, **kwargs):
		    return a
		print formatargspec(getargspec(f))`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase18 tests checkRules with an example that violates nothing.
func TestCheckRulesCase18(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Determine that the object is a method descriptor"
	parse := "(ROOT (S (VP (VB Determine) (SBAR (IN that) (S (NP (DT the) (NN object)) (VP (VBZ is) (NP (DT a) (NN method) (NN descriptor))))))))"
	tags := parseTags(parse)
	prelude := "from inspect import ismethoddescriptor"
	code := `class C(object):
		    def f(self):
			pass

		a = (getattr(C, d) for d in dir(C))
		m = a.next()

		while (ismethoddescriptor(m) == False):
		    print "False: ", m
		    m = a.next()

		print "True: ", m`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// -- Test cases from flask

// TestCheckRulesCase19 tests checkRules with an example that violates nothing.
func TestCheckRulesCase19(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Abort a request [with HTTP error code 404]"
	parse := "(ROOT (SINV (VP (VB Abort) (NP (DT a) (NN request)) (PP (IN with) (NP (NN HTTP) (NN error) (NN code)))) (NP (CD 404))))"
	tags := parseTags(parse)
	prelude := `import os
		    from flask import Flask, request
		    app = Flask(__name__)`
	code := `@app.errorhandler(404)
		 def page_not_found(error):
		     return "Aborted with 404", 404
    
		 @app.route('/abort_request')
		 def abort_request():
		     abort(404)
		     return "This should not be returned"`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRulesCase20 tests checkRules with an example that violates
// 1. 'hello.html' is used as an argument but not mentioned in the title.
// 2. myName=name is used as an argument but not mentioned in the title.
func TestCheckRulesCase20(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Render template [with variables]"
	parse := "(ROOT (S (VP (VB Render) (NP (NN template)) (PP (IN with) (NP (NNS variables))))))"
	tags := parseTags(parse)
	prelude := `import os
		    import sys
		    from flask import Flask, render_template

		    os.mkdir("templates")
		    f = open("templates/hello.html", "w")
		    f.write("""<!doctype html>
			    <h1>Hello {{myName}}!</h1>
			    """)
		    f.close()

		    app = Flask(__name__, template_folder=os.getcwd() + "/templates")`
	code := `@app.route('/hello/<name>')
		 def hello(name):
		    return render_template('hello.html', myName=name)`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}

// TestCheckRulesCase21 tests checkRules with an example that violates
// 1. Missing brackets around [from post request]
func TestCheckRulesCase21(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := "Receive data from post request"
	parse := "(ROOT (S (VP (VB Receive) (NP (NNS data)) (PP (IN from) (NP (NN post) (NN request))))))"
	tags := parseTags(parse)
	prelude := `import os
		    from flask import Flask, request

		    app = Flask(__name__)`
	code := `@app.route("/deposit_box", methods=["POST"])
		 def deposit_box():
		    data = request.get_data()
		 return data`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 0, len(violations))
}

// TestCheckRuleCase22 tests what happens when the title is empty
func TestCheckRuleCase22(t *testing.T) {
	validator := &TitleValidator{}
	validator.addRules(allRules)

	title := ""
	parse := ""
	tags := parseTags(parse)

	prelude := `import os
		    from flask import Flask, request

		    app = Flask(__name__)`
	code := `@app.route("/deposit_box", methods=["POST"])
		 def deposit_box():
		    data = request.get_data()
		 return data`

	violations, _ := validator.checkRules(newParsedTitle(title, title, parse, code, prelude, tags))
	assert.Equal(t, 1, len(violations))
}
