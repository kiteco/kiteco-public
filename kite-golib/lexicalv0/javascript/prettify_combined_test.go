package javascript

import (
	"bytes"
	"strings"
	"testing"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
)

var defaultCombinedConf = Config{
	ArrayElementNewline:        -1,
	ArrayBracketNewline:        -1,
	ArrayBracketSpacing:        true,
	ArrowSpacingBefore:         true,
	ArrowSpacingAfter:          true,
	CommaSpacingAfter:          true,
	FuncParenNewline:           -1,
	FuncParamArgumentNewline:   -1,
	Indent:                     2,
	KeySpacingAfterColon:       true,
	KeywordSpacingBefore:       true,
	KeywordSpacingAfter:        true,
	ObjectCurlyNewline:         -1,
	ObjectCurlySpacing:         true,
	ObjectPropertyNewline:      -1,
	SpaceBeforeBlocks:          true,
	SpaceInfixOps:              true,
	SpaceUnaryOpsWords:         true,
	StatementNewline:           true,
	SwitchColonNewLine:         true,
	JsxFragmentChildrenNewline: true,
	JsxElementChildrenNewline:  -1,
}

func TestPrettify_Combined_For(t *testing.T) {
	src := `
function fibonacci(num) {
  if (num <= 1) return 1;
  return fibonacci(num - 1) + fibonacci(num - 2);
}
`
	want := `
function fibonacci(num) {
  if (num <= 1) return 1;
  return fibonacci(num - 1) + fibonacci(num - 2);
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Function(t *testing.T) {
	src := `
function fibonacci(num){if(num<=1)return 1;return fibonacci(num-1)+fibonacci(num-2)}
`
	want := `
function fibonacci(num) {
  if (num <= 1) return 1;
  return fibonacci(num - 1) + fibonacci(num - 2)
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Function_2(t *testing.T) {
	src := `
function fibonacci(num) {
if(num<=1)
return 1
return fibonacci(num-1)+fibonacci(num-2)
}
`
	want := `
function fibonacci(num) {
  if (num <= 1) return 1
  return fibonacci(num - 1) + fibonacci(num - 2)
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_InfixSpace(t *testing.T) {
	src := `
console.log(i + +2)
`
	want := `
console.log(i + +2)
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_InfixSpace_2(t *testing.T) {
	src := `
console.log(i + +2)
`
	want := `
console.log(i+ +2)
`
	conf := defaultCombinedConf
	conf.SpaceInfixOps = false
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Statement(t *testing.T) {
	// from error found with https://raw.githubusercontent.com/zeit/next.js/canary/packages/next/client/head-manager.js
	src := `
export default class HeadManager {
  constructor() {
    this.updatePromise = null
  }

  updateHead = head => {
    const promise = (this.updatePromise = Promise.resolve().then(() => {
      if (promise !== this.updatePromise) return

      this.updatePromise = null
      this.doUpdateHead(head)
    }))
  }
}
`
	want := `
export default class HeadManager {
  constructor() {
    this.updatePromise = null
  }
  updateHead = head => {
    const promise = (this.updatePromise = Promise.resolve().then(() => {
      if (promise !== this.updatePromise) return
      this.updatePromise = null
      this.doUpdateHead(head)
    }))
  }
}
`
	conf := defaultCombinedConf
	conf.FuncParenNewline = 0
	conf.FuncParamArgumentNewline = 0
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Statement_2(t *testing.T) {
	// from error found with https://raw.githubusercontent.com/zeit/next.js/canary/packages/next/client/head-manager.js
	src := `
export default class HeadManager {
  constructor() {
    this.updatePromise = null
  }

  updateHead = head => {
    const promise = (this.updatePromise = Promise.resolve().then(() => {
      if (promise !== this.updatePromise) return

      this.updatePromise = null
      this.doUpdateHead(head)
    }))
  }
}
`
	want := `
export default class HeadManager {constructor() {this.updatePromise = null;};updateHead = head => {const promise = (this.updatePromise = Promise.resolve().then(() => {if (promise !== this.updatePromise) return; this.updatePromise = null; this.doUpdateHead(head);}));};};
`
	conf := defaultCombinedConf
	conf.FuncParenNewline = 0
	conf.Semicolon = true
	conf.FuncParamArgumentNewline = 0
	conf.StatementNewline = false
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Space(t *testing.T) {
	src := `
(({
x:[[{
a:b,
c:d,
}]],
y:{
z:1,
}
})
)
`
	want := `
(({
  x: [ [ {
    a: b,
    c: d,
  } ] ],
  y: {
    z: 1,
  }
}))
`
	conf := defaultCombinedConf
	conf.ArrayElementNewline = 0
	conf.ArrayBracketNewline = 0
	conf.ObjectCurlyNewline = 1
	conf.ObjectPropertyNewline = 1
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Map(t *testing.T) {
	src := `
export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props: {language: language}
  }
})
`
	want := `
export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props: { language: language }
  }
})
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Empty_For(t *testing.T) {
	src := `
for(;;){}
`
	want := `
for (;;) {}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_JSX_Element(t *testing.T) {
	src := `
render( < Provider store = { store } > < Router history = { history } > < Route path="/" component = { App } / >  < / Router > < / Provider > , document . getElementById('root'));
`
	want := `
render(
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={App}/>
    </Router>
  </Provider>,
  document.getElementById('root')
);
`
	conf := defaultCombinedConf
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_JSX_SelfClosing(t *testing.T) {
	src := `
< Route path = "/settings" name = "name" / >
`
	want := `
<Route path="/settings" name="name"/>
`
	conf := defaultCombinedConf
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_JSX_Fragment(t *testing.T) {
	src := `
<>
	<p>Q. What is React?</p>
	<p>A. A JavaScript library for building user interfaces</p>
	<p>Q. How do I render sibling elements?</p>
	<p>A. Use Fragments</p>
</>;
`
	want := `
<>
  <p>Q. What is React?</p>
  <p>A. A JavaScript library for building user interfaces</p>
  <p>Q. How do I render sibling elements?</p>
  <p>A. Use Fragments</p>
</>;
`
	conf := defaultCombinedConf
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_JSX_Element_2(t *testing.T) {
	src := `
render ( < div > The user is < b > { isLoggedIn ? 'currently' : 'not' } < / b > logged in. < / div > );
`
	want := `
render(
  <div>
    The user is
    <b>
      {isLoggedIn ? 'currently' : 'not'}
    </b>
    logged in.
  </div>
);
`
	conf := defaultCombinedConf
	conf.JsxElementChildrenNewline = 1
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Regex(t *testing.T) {
	src := `
Object.keys(meta.dependencies || {}).filter(key => /^d3-/.test(key))
`
	want := `
Object.keys(meta.dependencies || {}).filter(
  key => /^d3-/.test(key)
)
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Import(t *testing.T) {
	src := `
import {export1} from "module-name";
import {export1 as alias1} from "module-name";
import {export1,export2} from "module-name";
import {foo,bar} from "module-name";
import {export1,export2 as alias2,export3} from "module-name";
import defaultExport, {export1,export2 as alias2} from "module-name";
`
	want := `
import { export1 } from "module-name";
import { export1 as alias1 } from "module-name";
import { export1, export2 } from "module-name";
import { foo, bar } from "module-name";
import { export1, export2 as alias2, export3 } from "module-name";
import defaultExport, { export1, export2 as alias2 } from "module-name";
`
	conf := defaultCombinedConf
	conf.ObjectPropertyNewline = 0
	conf.ObjectCurlyNewline = 0
	conf.ObjectCurlySpacing = true
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Export(t *testing.T) {
	src := `
export{name1,name2,name3};
export{variable1 as name1,variable2 as name2,name3};
export const{name1,name2:bar}=o;
export{import1 as name1,import2 as name2,name3}from "x";
`
	want := `
export { name1, name2, name3 };
export { variable1 as name1, variable2 as name2, name3 };
export const { name1, name2: bar } = o;
export { import1 as name1, import2 as name2, name3 } from "x";
`

	conf := defaultCombinedConf
	conf.ObjectPropertyNewline = 0
	conf.ObjectCurlyNewline = 0
	conf.ObjectCurlySpacing = true
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_For_2(t *testing.T) {
	src := `
for(let i=0;i<10;i++){}
`
	want := `
for (let i = 0;i < 10;i++) {}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_For_3(t *testing.T) {
	src := `
for(let i=0;i<10;i++) return x;
	`
	want := `
for (let i = 0;i < 10;i++) return x;
	`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_For_4(t *testing.T) {
	src := `
for(let i=0;i<10;i++) return x;
	`
	want := `
for (let i = 0;i < 10;i++)
  return x;
	`
	conf := defaultCombinedConf
	conf.NonBlockStatementBodyLinebreak = true
	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Import_2(t *testing.T) {
	src := `
import ascii from"rollup-plugin-ascii";
import node from"rollup-plugin-node-resolve";
import{terser}from"rollup-plugin-terser";
import*as meta from"./package.json";
export*as name1 from"x";
`
	want := `
import ascii from "rollup-plugin-ascii";
import node from "rollup-plugin-node-resolve";
import { terser } from "rollup-plugin-terser";
import * as meta from "./package.json";
export * as name1 from "x";
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Func_Auto_1(t *testing.T) {
	src := `
foo ( bar , baz , bat , bay , bao ) ;
`
	want := `
foo(
  bar,
  baz,
  bat,
  bay,
  bao
);
`
	conf := defaultCombinedConf
	conf.FuncParenNewline = -1
	conf.FuncParamArgumentNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Func_Auto_2(t *testing.T) {
	src := `
render ( < div > The user is < b > { isLoggedIn ? 'currently' : 'not' } < / b > logged in. < / div > );
`
	want := `
render(
  <div>
    The user is
    <b>{isLoggedIn ? 'currently' : 'not'}</b>
    logged in.
  </div>
);
`
	conf := defaultCombinedConf
	conf.FuncParenNewline = -1
	conf.FuncParamArgumentNewline = -1
	conf.JsxElementChildrenNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Func_Auto_3(t *testing.T) {
	src := `
path.split(separator).filter((f, i) => f || !i)
`
	want := `
path.split(separator).filter(
  (f, i) => f || !i
)
`
	conf := defaultCombinedConf
	conf.FuncParenNewline = -1
	conf.FuncParamArgumentNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Object_Auto_1(t *testing.T) {
	src := `
const loadExamples = (state, action) => {
  return {...state, status: "loading", language: action.language, identifiers: action.identifiers,
  };
};
`
	want := `
const loadExamples = (state, action) => {
  return {
    ...state,
    status: "loading",
    language: action.language,
    identifiers: action.identifiers,
  };
};
`
	conf := defaultCombinedConf
	conf.ObjectCurlyNewline = -1
	conf.ObjectPropertyNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Object_Auto_2(t *testing.T) {
	src := `const 
{ 
  loading, stage, message 
} = this.state;`
	want := `const {
  loading,
  stage,
  message
} = this.state;`

	conf := defaultCombinedConf
	conf.ObjectCurlyNewline = -1
	conf.ObjectPropertyNewline = -1
	conf.ObjectCurlySpacing = true

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Jsx_Auto_1(t *testing.T) {
	src := `
<a href="#" className="home__link__settings" onClick={this.setEmailStage} >Change email</a>
`
	want := `
<a
  href="#"
  className="home__link__settings"
  onClick={this.setEmailStage}
>
  Change email
</a>
`
	conf := defaultCombinedConf
	conf.JsxAttributeNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Jsx_Auto_2(t *testing.T) {
	src := `
<div className="login__row login__start">
	<input name="email" type="email" className="login__input" placeholder="Email" value={this.state.email}/> 
</div>
`
	want := `
<div className="login__row login__start">
  <input
    name="email"
    type="email"
    className="login__input"
    placeholder="Email"
    value={this.state.email}
  />
</div>
`
	conf := defaultCombinedConf
	conf.JsxAttributeNewline = -1

	runPrettifyCase(t, conf, src, want)
}

func TestPrettify_Combined_Jsx_Unclosed_01(t *testing.T) {
	// issue #10452
	src := `
render(< Provider store={} >)
`
	want := `
render(<Provider store={}>)
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Jsx_Unclosed_02(t *testing.T) {
	// issue #10452
	src := `
render(< Provider store={})
`
	want := `
render(<Provider store={})
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_String_Unclosed(t *testing.T) {
	src := `
render('abc)
`
	want := `
render('abc)
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Call_Unclosed(t *testing.T) {
	src := `
render(
`
	want := `
render(
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_ArrayArgAndFunc_Unclosed(t *testing.T) {
	src := `
render(x[1
`
	want := `
render(x[1
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Superfluous_Paren(t *testing.T) {
	src := `
render(x))
`
	want := `
render(x))
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_StatementBlock(t *testing.T) {
	src := `
const loadExamples = (state, action) => { switch (action) }
`
	want := `
const loadExamples = (state, action) => {
  switch (action)
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_EmptyStatementBlock(t *testing.T) {
	src := `if (success) {}`
	want := `if (success) {}`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Switch(t *testing.T) {
	src := `
const history = (state = defaultState, action) => {
  switch (action.type) {
    case actions.DO_NOTHING:
    case actions.ADD_PAGE_TO_HISTORY:
      x = addPageToHistory(state, action);
      return x;
    default:
      return state;
  }
};`
	want := `
const history = (state = defaultState, action) => {
  switch (action.type) {
    case actions.DO_NOTHING:
    case actions.ADD_PAGE_TO_HISTORY:
      x = addPageToHistory(state, action);
      return x;
    default:
      return state;
  }
};`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_SwitchBracket(t *testing.T) {
	src := `
const action = 'say_hello';
switch (action) {
  case 'say_hello': {
    let message = 'hello';
    console.log(message);
    break;
  }
  default: {
    console.log('Empty action received.');
    break;
  }
}
`
	want := `
const action = 'say_hello';
switch (action) {
  case 'say_hello': {
    let message = 'hello';
    console.log(message);
    break;
  }
  default: {
    console.log('Empty action received.');
    break;
  }
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_TemplateString(t *testing.T) {
	src := "console.log(`Fifteen is ${a+b} and\nnot ${2*a+b}.`);"
	want := "console.log(`Fifteen is ${a + b} and\nnot ${2 * a + b}.`);"
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_TemplateString_Nested(t *testing.T) {
	src := "const classes = `header ${isLargeScreen()?'':\n`icon-${item.isCollapsed?'expander':'collapser'}`}`;"
	want := "const classes = `header ${isLargeScreen() ? '' : `icon-${item.isCollapsed ? 'expander' : 'collapser'}`}`;"
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_TemplateString_Tag(t *testing.T) {
	src := "console.log(tag`Hello ${x}`);"
	want := "console.log(\n  tag`Hello ${x}`\n);"
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Switch_Blocks(t *testing.T) {
	src := `
const todo = (state, action) => {
  switch (action.type) {
    case 'ADD_TODO':
      return { id: action.id, text: action.text }
    case 'TOGGLE_TODO':
      if (state.id !== action.id) {
        return todo
      }
  }
}
`
	want := `
const todo = (state, action) => {
  switch (action.type) {
    case 'ADD_TODO':
      return { id: action.id, text: action.text }
    case 'TOGGLE_TODO':
      if (state.id !== action.id) {
        return todo
      }
  }
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Parenthesized_Expression(t *testing.T) {
	src := `
if (headingPath) {
  return (
    <Heading className='something'>
      <Link className='anything' to={headingPath}>
        {heading}
      </Link>
    </Heading>
  )
}
`
	want := `
if (headingPath) {
  return (
    <Heading className='something'>
      <Link className='anything' to={headingPath}>
        {heading}
      </Link>
    </Heading>
  )
}
`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func TestPrettify_Combined_Vue_At(t *testing.T) {
	src := `
<template>
  <draggable class="list-group" :move="onMove" @end="...">
  </draggable>
</template>`
	want := `
<template>
  <draggable class="list-group" :move="onMove" @end="...">
  </draggable>
</template>`
	runPrettifyCase(t, defaultCombinedConf, src, want)
}

func runPrettifyCase(t *testing.T, conf Config, src, want string) {
	src, want = strings.TrimSpace(src), strings.TrimSpace(want)

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(javascript.GetLanguage())
	b := []byte(src)
	tree := parser.Parse(b)
	defer tree.Close()

	var buf bytes.Buffer
	if _, err := Prettify(&buf, conf, b, 0, len(src), tree.RootNode()); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != want {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}
