package javascript

import (
	"bytes"
	"testing"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
)

type testCase struct {
	conf    Config
	in, out string
}

func TestPrettify_Semicolon(t *testing.T) {
	conf := Config{
		Semicolon: true,
	}
	cases := []testCase{
		{conf: conf, in: `a`, out: "a;"},
		{conf: conf, in: `for (i=0; i<10; i++) {i}`, out: "for(i=0;i<10;i++){i;}"},
		{conf: conf, in: `if(x) y`, out: "if(x) y;"},
		{conf: conf, in: `if(x){y}else{z}`, out: "if(x){y;}else{z;}"},
		{conf: conf, in: `return {}`, out: "return{};"},
		{conf: conf, in: `while(x) break`, out: "while(x) break;"},
		{conf: conf, in: "function(x){\n}\nx", out: "function(x){};x;"},
		{conf: conf, in: "function f(x){\n}\nx", out: "function f(x){};x;"},
		{conf: conf, in: "function f(x){\n};\nx", out: "function f(x){};x;"},
		{conf: conf, in: "class X{\n}\nx", out: "class X{};x;"},
		{conf: conf, in: "function f(x){\n}\n[x]", out: "function f(x){}[x];"},
		{conf: conf, in: "function f(x){\n};\n[x]", out: "function f(x){};[x];"},
		{conf: conf, in: "function f(x){\n}\n;[x]", out: "function f(x){};[x];"},
		{conf: conf, in: "class X{\n}\n[x]", out: "class X{}[x];"},
		{conf: conf, in: "function f(x){\n}\n(x)", out: "function f(x){}(x);"},
		{conf: conf, in: "class X{\n}\n(x)", out: "class X{}(x);"},
	}
	runTestCases(t, cases)

	conf = Config{
		Semicolon: false,
	}
	cases = []testCase{
		{conf: conf, in: `a`, out: "a"},
		{conf: conf, in: `for (i=0; i<10; i++) {i}`, out: "for(i=0;i<10;i++){i}"},
		{conf: conf, in: `if(x) y`, out: "if(x) y"},
		{conf: conf, in: `if(x){y}else{z}`, out: "if(x){y}else{z}"},
		{conf: conf, in: `return {}`, out: "return{}"},
		{conf: conf, in: `while(x) break`, out: "while(x) break"},
		{conf: conf, in: `super(props)`, out: "super(props)"},
		{conf: conf, in: "function(x){\n}\nx", out: "function(x){}\nx"},
		{conf: conf, in: "function f(x){\n}\nx", out: "function f(x){}x"},
		{conf: conf, in: "function f(x){\n};\nx", out: "function f(x){};x"},
		{conf: conf, in: "class X{\n}\nx", out: "class X{}x"},
		{conf: conf, in: "function f(x){\n}\n[x]", out: "function f(x){}[x]"},
		{conf: conf, in: "function f(x){\n};\n[x]", out: "function f(x){};[x]"},
		{conf: conf, in: "function f(x){\n}\n;[x]", out: "function f(x){};[x]"},
		{conf: conf, in: "class X{\n}\n[x]", out: "class X{}[x]"},
		{conf: conf, in: "function f(x){\n}\n(x)", out: "function f(x){}(x)"},
		{conf: conf, in: "class X{\n}\n(x)", out: "class X{}(x)"},
	}
	runTestCases(t, cases)
}

func TestPrettify_Indent(t *testing.T) {
	conf := Config{
		Indent:           2,
		StatementNewline: true,
	}
	cases := []testCase{
		{conf: conf, in: `class X{}`, out: "class X{\n}"},
		{conf: conf, in: `class X{a}`, out: "class X{\n  a\n}"},
		{conf: conf, in: `if(x)y;`, out: "if(x) y;"},
		{conf: conf, in: `if(x){y;}`, out: "if(x){\n  y;\n}"},
		{conf: conf, in: `if(x){y;}else if(v){w;}else{z;}`, out: "if(x){\n  y;\n}else if(v){\n  w;\n}else{\n  z;\n}"},
		{conf: conf, in: `switch(x){default:y;}`, out: "switch(x){\n  default:y;\n}"},
		{conf: conf, in: `for(i in x){y;z;}`, out: "for(i in x){\n  y;\n  z;\n}"},
		{conf: conf, in: `try{x;y;}catch(e){z;}finally{a;b;}`, out: "try{\n  x;\n  y;\n}catch(e){\n  z;\n}finally{\n  a;\n  b;\n}"},
	}
	runTestCases(t, cases)

	conf = Config{
		Indent:           -1,
		StatementNewline: true,
	}
	cases = []testCase{
		{conf: conf, in: `class X{}`, out: "class X{\n}"},
		{conf: conf, in: `class X{a}`, out: "class X{\n\ta\n}"},
		{conf: conf, in: `if(x)y;`, out: "if(x) y;"},
		{conf: conf, in: `if(x){y;}`, out: "if(x){\n\ty;\n}"},
		{conf: conf, in: `if(x){y;}else if(v){w;}else{z;}`, out: "if(x){\n\ty;\n}else if(v){\n\tw;\n}else{\n\tz;\n}"},
		{conf: conf, in: `switch(x){default:y;}`, out: "switch(x){\n\tdefault:y;\n}"},
		{conf: conf, in: `for(i in x){y;z;}`, out: "for(i in x){\n\ty;\n\tz;\n}"},
		{conf: conf, in: `try{x;y;}catch(e){z;}finally{a;b;}`, out: "try{\n\tx;\n\ty;\n}catch(e){\n\tz;\n}finally{\n\ta;\n\tb;\n}"},
	}
	runTestCases(t, cases)

	conf = Config{
		Indent:           0,
		StatementNewline: false,
	}
	cases = []testCase{
		{conf: conf, in: `class X{}`, out: "class X{}"},
		{conf: conf, in: `class X{a}`, out: "class X{a}"},
		{conf: conf, in: `if(x)y;`, out: "if(x) y;"},
		{conf: conf, in: `if(x){y;}`, out: "if(x){y;}"},
		{conf: conf, in: `if(x){y;}else if(v){w;}else{z;}`, out: "if(x){y;}else if(v){w;}else{z;}"},
		{conf: conf, in: `switch(x){default:y;}`, out: "switch(x){default:y;}"},
		{conf: conf, in: `for(i in x){y;z;}`, out: "for(i in x){y;z;}"},
		{conf: conf, in: `try{x;y;}catch(e){z;}finally{a;b;}`, out: "try{x;y;}catch(e){z;}finally{a;b;}"},
	}
	runTestCases(t, cases)
}

func TestPrettify_BlockSpacing(t *testing.T) {
	conf := Config{
		BlockSpacing: true,
	}
	cases := []testCase{
		{conf: conf, in: "if(true){return;}else{return;}", out: `if(true){ return; }else{ return; }`},
		{conf: conf, in: `for(x in y){break;}`, out: `for(x in y){ break; }`},
		{conf: conf, in: `function(x){}`, out: `function(x){ }`},
		{conf: conf, in: `try{}catch{}finally{}`, out: `try{ }catch{ }finally{ }`},
		{conf: conf, in: `switch(x){}`, out: `switch(x){ }`},
		{conf: conf, in: `class X {}`, out: `class X{ }`},
	}
	runTestCases(t, cases)

	conf = Config{
		BlockSpacing: false,
	}
	cases = []testCase{
		{conf: conf, in: "if(true){return;}else{return;}", out: `if(true){return;}else{return;}`},
		{conf: conf, in: `for(x in y){break;}`, out: `for(x in y){break;}`},
		{conf: conf, in: `function(x){}`, out: `function(x){}`},
		{conf: conf, in: `try{}catch{}finally{}`, out: `try{}catch{}finally{}`},
		{conf: conf, in: `switch(x){}`, out: `switch(x){}`},
		{conf: conf, in: `class X {}`, out: `class X{}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_ArrayBracketNewline(t *testing.T) {
	conf := Config{
		ArrayElementNewline: 1,
		ArrayBracketNewline: 1,
	}
	cases := []testCase{
		{conf: conf, in: `[]`, out: `[]`},
		{conf: conf, in: `[a]`, out: "[\na\n]"},
		{conf: conf, in: `[a,b]`, out: "[\na,\nb\n]"},
		{conf: conf, in: `[a,b,[c,d]]`, out: "[\na,\nb,\n[\nc,\nd\n]\n]"},
	}
	runTestCases(t, cases)

	conf = Config{
		ArrayBracketNewline: 0,
		ArrayElementNewline: 0,
	}
	cases = []testCase{
		{conf: conf, in: `[]`, out: `[]`},
		{conf: conf, in: `[a]`, out: "[a]"},
		{conf: conf, in: `[a,b]`, out: "[a,b]"},
		{conf: conf, in: `[a,b,[c,d]]`, out: "[a,b,[c,d]]"},
	}
	runTestCases(t, cases)
}

func TestPrettify_ArrayBracketSpacing(t *testing.T) {
	conf := Config{
		ArrayBracketSpacing: true,
	}
	cases := []testCase{
		{conf: conf, in: `[]`, out: `[]`},
		{conf: conf, in: `[a]`, out: "[ a ]"},
		{conf: conf, in: `[a,b]`, out: "[ a,b ]"},
		{conf: conf, in: `[a,b,[c,d]]`, out: "[ a,b,[ c,d ] ]"},
	}
	runTestCases(t, cases)

	conf = Config{
		ArrayBracketSpacing: false,
	}
	cases = []testCase{
		{conf: conf, in: `[]`, out: `[]`},
		{conf: conf, in: `[a]`, out: "[a]"},
		{conf: conf, in: `[a,b]`, out: "[a,b]"},
		{conf: conf, in: `[a,b,[c,d]]`, out: "[a,b,[c,d]]"},
	}
	runTestCases(t, cases)
}

func TestPrettify_CommaSpacing(t *testing.T) {
	conf := Config{
		CommaSpacingBefore: true,
		CommaSpacingAfter:  true,
	}
	cases := []testCase{
		{conf: conf, in: `1,2,3`, out: `1 , 2 , 3`},
		{conf: conf, in: `{a:1,b:2}`, out: `{a:1 , b:2}`},
		{conf: conf, in: `[1,2]`, out: `[1 , 2]`},
		{conf: conf, in: `f(a,b,c)`, out: `f(a , b , c)`},
		{conf: conf, in: `[,,1]`, out: `[ , , 1]`},
	}
	runTestCases(t, cases)

	conf = Config{
		CommaSpacingBefore: false,
		CommaSpacingAfter:  true,
	}
	cases = []testCase{
		{conf: conf, in: `1,2,3`, out: `1, 2, 3`},
		{conf: conf, in: `{a:1,b:2}`, out: `{a:1, b:2}`},
		{conf: conf, in: `[1,2]`, out: `[1, 2]`},
		{conf: conf, in: `f(a,b,c)`, out: `f(a, b, c)`},
		{conf: conf, in: `[,,1]`, out: `[, , 1]`},
	}
	runTestCases(t, cases)

	conf = Config{
		CommaSpacingBefore: false,
		CommaSpacingAfter:  false,
	}
	cases = []testCase{
		{conf: conf, in: `1,2,3`, out: `1,2,3`},
		{conf: conf, in: `{a:1,b:2}`, out: `{a:1,b:2}`},
		{conf: conf, in: `[1,2]`, out: `[1,2]`},
		{conf: conf, in: `f(a,b,c)`, out: `f(a,b,c)`},
		{conf: conf, in: `[,,1]`, out: `[,,1]`},
	}
	runTestCases(t, cases)
}

func TestPrettify_CompPropSpacing(t *testing.T) {
	conf := Config{
		ComputedPropertySpacing: true,
	}
	cases := []testCase{
		{conf: conf, in: `{[x]:1}`, out: `{[ x ]:1}`},
		{conf: conf, in: `{[x[y]]:1}`, out: `{[ x[y] ]:1}`}, // the inner [y] is parsed as a subscript op, controlled by array bracket spacing
	}
	runTestCases(t, cases)

	conf = Config{
		ComputedPropertySpacing: false,
	}
	cases = []testCase{
		{conf: conf, in: `{[x]:1}`, out: `{[x]:1}`},
		{conf: conf, in: `{[x[y]]:1}`, out: `{[x[y]]:1}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_FuncCallSpacing(t *testing.T) {
	conf := Config{
		FuncCallSpacing: true,
	}
	cases := []testCase{
		{conf: conf, in: `f()`, out: "f ()"},
		{conf: conf, in: `f(a)`, out: "f ( a )"},
		{conf: conf, in: `f(a,b,c)`, out: "f ( a,b,c )"},
	}
	runTestCases(t, cases)

	conf = Config{
		FuncCallSpacing: false,
	}
	cases = []testCase{
		{conf: conf, in: `f()`, out: "f()"},
		{conf: conf, in: `f(a)`, out: "f(a)"},
		{conf: conf, in: `f(a,b,c)`, out: "f(a,b,c)"},
	}
	runTestCases(t, cases)
}

func TestPrettify_FuncParamArgNewline(t *testing.T) {
	conf := Config{
		FuncParenNewline:         1,
		FuncParamArgumentNewline: 1,
	}
	cases := []testCase{
		{conf: conf, in: `function f()`, out: "function f()"},
		{conf: conf, in: `function f(a)`, out: "function f(\na\n)"},
		{conf: conf, in: `function f(a,b)`, out: "function f(\na,\nb\n)"},
		{conf: conf, in: `()=>1`, out: "()=> 1"},
		{conf: conf, in: `(a)=>1`, out: "(\na\n)=> 1"},
		{conf: conf, in: `(a,b)=>1`, out: "(\na,\nb\n)=> 1"},
		{conf: conf, in: `var f=function(a,b)`, out: "var f=function(\na,\nb\n)"},
		{conf: conf, in: `f()`, out: "f()"},
		{conf: conf, in: `f(a)`, out: "f(\na\n)"},
		{conf: conf, in: `f(a,b,c)`, out: "f(\na,\nb,\nc\n)"},
	}
	runTestCases(t, cases)

	conf = Config{
		FuncParenNewline:         0,
		FuncParamArgumentNewline: 0,
	}
	cases = []testCase{
		{conf: conf, in: `function f(a,b)`, out: "function f(a,b)"},
		{conf: conf, in: `(a,b)=>1`, out: "(a,b)=> 1"},
		{conf: conf, in: `var f=function(a,b)`, out: "var f=function(a,b)"},
		{conf: conf, in: `f(a,b,c)`, out: "f(a,b,c)"},
	}
	runTestCases(t, cases)
}

func TestPrettify_ImplicitArrowLinebreak(t *testing.T) {
	conf := Config{
		ArrowSpacingBefore:     true,
		ImplicitArrowLinebreak: true,
	}
	cases := []testCase{
		{conf: conf, in: `()=>a`, out: "() =>\na"},
		{conf: conf, in: `()=>{}`, out: "() =>\n{}"}, // here {} is not a block statement, but a returned empty object!
		{conf: conf, in: `()=>{return x;}`, out: "() =>{return x;}"},
	}
	runTestCases(t, cases)

	conf = Config{
		ArrowSpacingBefore:     false,
		ImplicitArrowLinebreak: false,
	}
	cases = []testCase{
		{conf: conf, in: `()=>a`, out: "()=> a"},
		{conf: conf, in: `()=>{}`, out: "()=> {}"}, // here {} is not a block statement, but a returned empty object!
		{conf: conf, in: `()=>{return x;}`, out: "()=>{return x;}"},
	}
	runTestCases(t, cases)
}

func TestPrettify_KeySpacing(t *testing.T) {
	conf := Config{
		KeySpacingBeforeColon: true,
		KeySpacingAfterColon:  true,
	}
	cases := []testCase{
		{conf: conf, in: `{x:1}`, out: `{x : 1}`},
		{conf: conf, in: `{x:1,y:true,z:{a:1}}`, out: `{x : 1,y : true,z : {a : 1}}`},
	}
	runTestCases(t, cases)

	conf = Config{
		KeySpacingBeforeColon: false,
		KeySpacingAfterColon:  true,
	}
	cases = []testCase{
		{conf: conf, in: `{x:1}`, out: `{x: 1}`},
		{conf: conf, in: `{x:1,y:true,z:{a:1}}`, out: `{x: 1,y: true,z: {a: 1}}`},
	}
	runTestCases(t, cases)

	conf = Config{
		KeySpacingBeforeColon: false,
		KeySpacingAfterColon:  false,
	}
	cases = []testCase{
		{conf: conf, in: `{x:1}`, out: `{x:1}`},
		{conf: conf, in: `{x:1,y:true,z:{a:1}}`, out: `{x:1,y:true,z:{a:1}}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_KeywordSpacing(t *testing.T) {
	conf := Config{
		KeywordSpacingBefore: true,
		KeywordSpacingAfter:  true,
	}
	cases := []testCase{
		{conf: conf, in: `if(x)y;`, out: `if (x) y;`},
		{conf: conf, in: `if(x){y;}else{z;}`, out: `if (x){y;} else {z;}`},
		{conf: conf, in: `while(x)y;`, out: `while (x) y;`},
		{conf: conf, in: `for(x in y)z;`, out: `for (x in y) z;`},
		{conf: conf, in: `try{x;}catch(e){}finally{}`, out: `try {x;} catch (e){} finally {}`},
	}
	runTestCases(t, cases)

	conf = Config{
		KeywordSpacingBefore: false,
		KeywordSpacingAfter:  false,
	}
	cases = []testCase{
		{conf: conf, in: `if(x)y;`, out: `if(x) y;`},
		{conf: conf, in: `if(x){y;}else{z;}`, out: `if(x){y;}else{z;}`},
		{conf: conf, in: `while(x)y;`, out: `while(x) y;`},
		{conf: conf, in: `for(x in y)z;`, out: `for(x in y) z;`},
		{conf: conf, in: `try{x;}catch(e){}finally{}`, out: `try{x;}catch(e){}finally{}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_NonBlockNewline(t *testing.T) {
	conf := Config{
		NonBlockStatementBodyLinebreak: true,
	}
	cases := []testCase{
		{conf: conf, in: `do x;while (y);`, out: "do\nx;while(y);"},
		{conf: conf, in: `for(i=0;i<1;i++)x;`, out: "for(i=0;i<1;i++)\nx;"},
		{conf: conf, in: `for(v in obj)x;`, out: "for(v in obj)\nx;"},
		{conf: conf, in: `for(v of obj)x;`, out: "for(v of obj)\nx;"},
		{conf: conf, in: `while(x)y`, out: "while(x)\ny"},
		{conf: conf, in: `with(x)y`, out: "with(x)\ny"},
		{conf: conf, in: `if(x)y`, out: "if(x)\ny"},
		{conf: conf, in: `if(x)y;else z`, out: "if(x)\ny;else\nz"},
		{conf: conf, in: `if(x){y}else z`, out: "if(x){y}else\nz"},
	}
	runTestCases(t, cases)

	conf = Config{
		NonBlockStatementBodyLinebreak: false,
	}
	cases = []testCase{
		{conf: conf, in: `do x;while (y);`, out: "do x;while(y);"},
		{conf: conf, in: `for(i=0;i<1;i++)x;`, out: "for(i=0;i<1;i++) x;"},
		{conf: conf, in: `for(v in obj)x;`, out: "for(v in obj) x;"},
		{conf: conf, in: `for(v of obj)x;`, out: "for(v of obj) x;"},
		{conf: conf, in: `while(x)y`, out: "while(x) y"},
		{conf: conf, in: `with(x)y`, out: "with(x) y"},
		{conf: conf, in: `if(x)y`, out: "if(x) y"},
		{conf: conf, in: `if(x)y;else z`, out: "if(x) y;else z"},
		{conf: conf, in: `if(x){y}else z`, out: "if(x){y}else z"},
		{conf: conf, in: `if(x){y}else{z}`, out: "if(x){y}else{z}"},
	}
	runTestCases(t, cases)
}

func TestPrettify_ObjPropNewline(t *testing.T) {
	conf := Config{
		ObjectCurlyNewline:    1,
		ObjectPropertyNewline: 1,
	}
	cases := []testCase{
		{conf: conf, in: `{}`, out: `{}`},
		{conf: conf, in: `{a:2}`, out: "{\na:2\n}"},
		{conf: conf, in: `{a:2,b:3}`, out: "{\na:2,\nb:3\n}"},
		{conf: conf, in: `{a:2,b:{c:true,d:false}}`, out: "{\na:2,\nb:{\nc:true,\nd:false\n}\n}"},
	}
	runTestCases(t, cases)

	conf = Config{
		ObjectCurlyNewline:    0,
		ObjectPropertyNewline: 0,
	}
	cases = []testCase{
		{conf: conf, in: `{}`, out: `{}`},
		{conf: conf, in: `{a:2}`, out: "{a:2}"},
		{conf: conf, in: `{a:2,b:3}`, out: "{a:2,b:3}"},
		{conf: conf, in: `{a:2,b:{c:true,d:false}}`, out: "{a:2,b:{c:true,d:false}}"},
	}
	runTestCases(t, cases)
}

func TestPrettify_ObjCurlySpacing(t *testing.T) {
	conf := Config{
		ObjectCurlySpacing: true,
	}
	cases := []testCase{
		{conf: conf, in: `{}`, out: `{}`},
		{conf: conf, in: `{a:2}`, out: "{ a:2 }"},
		{conf: conf, in: `{a:2,b:3}`, out: "{ a:2,b:3 }"},
		{conf: conf, in: `{a:2,b:{c:true,d:false}}`, out: "{ a:2,b:{ c:true,d:false } }"},
	}
	runTestCases(t, cases)

	conf = Config{
		ObjectCurlySpacing: false,
	}
	cases = []testCase{
		{conf: conf, in: `{}`, out: `{}`},
		{conf: conf, in: `{a:2}`, out: "{a:2}"},
		{conf: conf, in: `{a:2,b:3}`, out: "{a:2,b:3}"},
		{conf: conf, in: `{a:2,b:{c:true,d:false}}`, out: "{a:2,b:{c:true,d:false}}"},
	}
	runTestCases(t, cases)
}

func TestPrettify_BeforeBlocks(t *testing.T) {
	conf := Config{
		SpaceBeforeBlocks: true,
	}
	cases := []testCase{
		{conf: conf, in: `function(){}`, out: `function() {}`},
		{conf: conf, in: `function a(){}`, out: `function a() {}`},
		{conf: conf, in: `x=()=>{}`, out: `x=()=> {}`},
		{conf: conf, in: `switch(x){}`, out: `switch(x) {}`},
		{conf: conf, in: `x={y:{}}`, out: `x= {y: {}}`},
		{conf: conf, in: `class X{}`, out: `class X {}`},
	}
	runTestCases(t, cases)

	conf = Config{
		SpaceBeforeBlocks: false,
	}
	cases = []testCase{
		{conf: conf, in: `function(){}`, out: `function(){}`},
		{conf: conf, in: `function a(){}`, out: `function a(){}`},
		{conf: conf, in: `x=()=>{}`, out: `x=()=> {}`}, // the ImplicitArrowLinebreak=false rule triggers this
		{conf: conf, in: `switch(x){}`, out: `switch(x){}`},
		{conf: conf, in: `x={y:{}}`, out: `x={y:{}}`},
		{conf: conf, in: `class X{}`, out: `class X{}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_BeforeFuncParen(t *testing.T) {
	conf := Config{
		SpaceBeforeFuncParen: true,
	}
	cases := []testCase{
		{conf: conf, in: `function(){}`, out: `function (){}`},
		{conf: conf, in: `function a(){}`, out: `function a (){}`},
		{conf: conf, in: `x=()=>{}`, out: `x= ()=> {}`},
	}
	runTestCases(t, cases)

	conf = Config{
		SpaceBeforeFuncParen: false,
	}
	cases = []testCase{
		{conf: conf, in: `function(){}`, out: `function(){}`},
		{conf: conf, in: `function a(){}`, out: `function a(){}`},
		{conf: conf, in: `x=()=>{}`, out: `x=()=> {}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_Parens(t *testing.T) {
	conf := Config{
		SpaceInParens: true,
	}
	cases := []testCase{
		{conf: conf, in: `(a)`, out: `( a )`},
		{conf: conf, in: `()`, out: `( )`},
		{conf: conf, in: `(1+(2*a.x.(y, z)))`, out: `( 1+( 2*a.x.( y,z ) ) )`},
	}
	runTestCases(t, cases)

	conf.SpaceInParens = false
	cases = []testCase{
		{conf: conf, in: `(a)`, out: `(a)`},
		{conf: conf, in: `()`, out: `()`},
		{conf: conf, in: `(1+(2*a.x.(y, z)))`, out: `(1+(2*a.x.(y,z)))`},
	}
	runTestCases(t, cases)
}

func TestPrettify_Template(t *testing.T) {
	conf := Config{
		TemplateTagSpacing: true,
	}
	cases := []testCase{
		// only applies to tagged templated strings, not to explicit func calls
		{conf: conf, in: "tag`a`", out: "tag `a`"},
		{conf: conf, in: "tag(`a`)", out: "tag(`a`)"},
	}
	runTestCases(t, cases)

	conf.TemplateTagSpacing = false
	cases = []testCase{
		{conf: conf, in: "tag`a`", out: "tag`a`"},
	}
	runTestCases(t, cases)
}

func TestPrettify_Unary(t *testing.T) {
	conf := Config{
		SpaceUnaryOpsWords:    true,
		SpaceUnaryOpsNonWords: true,
	}
	cases := []testCase{
		{conf: conf, in: `typeof{}`, out: `typeof {}`},
		{conf: conf, in: `new A`, out: `new A`},
		{conf: conf, in: `yield x`, out: `yield x`},
		{conf: conf, in: `void 0`, out: `void 0`},
		{conf: conf, in: `delete a`, out: `delete a`},

		{conf: conf, in: `-a`, out: `- a`},
		{conf: conf, in: `a++`, out: `a ++`},
		{conf: conf, in: `!!x`, out: `! ! x`},
		{conf: conf, in: `!!-+~x++`, out: `! ! - + ~ x ++`},
	}
	runTestCases(t, cases)

	conf = Config{
		SpaceUnaryOpsWords:    false,
		SpaceUnaryOpsNonWords: false,
	}
	cases = []testCase{
		{conf: conf, in: `typeof{}`, out: `typeof{}`},
		{conf: conf, in: `new A`, out: `new A`},
		{conf: conf, in: `yield x`, out: `yield x`},
		{conf: conf, in: `void 0`, out: `void 0`},
		{conf: conf, in: `delete a`, out: `delete a`},

		{conf: conf, in: `-a`, out: `-a`},
		{conf: conf, in: `a++`, out: `a++`},
		{conf: conf, in: `!!x`, out: `!!x`},
		{conf: conf, in: `!!-+~x++`, out: `!!- +~x++`}, // space added due to potential operator conflict rule
	}
	runTestCases(t, cases)
}

func TestPrettify_Infix(t *testing.T) {
	conf := Config{
		SpaceInfixOps: true,
	}
	cases := []testCase{
		{conf: conf, in: `a`, out: `a`},
		{conf: conf, in: `a+b`, out: `a + b`},
		{conf: conf, in: `a.x+"s"`, out: `a.x + "s"`},
		{conf: conf, in: `a.y-b.x+"s"+'z'`, out: `a.y - b.x + "s" + 'z'`},
		{conf: conf, in: `a?b:c`, out: `a ? b : c`},
		{conf: conf, in: `a.x.y+b.v().w>123?(x?y:z):c.d().e`, out: `a.x.y + b.v().w > 123 ? (x ? y : z) : c.d().e`},
		{conf: conf, in: `a=1`, out: `a = 1`},
		{conf: conf, in: `a<<=1`, out: `a <<= 1`},
		{conf: conf, in: `let a=1`, out: `let a = 1`},
		{conf: conf, in: `var a=1`, out: `var a = 1`},
		{conf: conf, in: `const a=1,b=2`, out: `const a = 1,b = 2`},
		{conf: conf, in: `var {a=0}=bar`, out: `var{a = 0} = bar`},
		{conf: conf, in: `function foo(a=0){}`, out: `function foo(a = 0){}`},
	}
	runTestCases(t, cases)

	conf = Config{
		SpaceInfixOps: false,
	}
	cases = []testCase{
		{conf: conf, in: `a`, out: `a`},
		{conf: conf, in: `a+b`, out: `a+b`},
		{conf: conf, in: `a.x+"s"`, out: `a.x+"s"`},
		{conf: conf, in: `a.y-b.x+"s"+'z'`, out: `a.y-b.x+"s"+'z'`},
		{conf: conf, in: `a?b:c`, out: `a?b:c`},
		{conf: conf, in: `a.x.y+b.v().w>123?(x?y:z):c.d().e`, out: `a.x.y+b.v().w>123?(x?y:z):c.d().e`},
		{conf: conf, in: `a=1`, out: `a=1`},
		{conf: conf, in: `a<<=1`, out: `a<<=1`},
		{conf: conf, in: `let a=1`, out: `let a=1`},
		{conf: conf, in: `var a=1`, out: `var a=1`},
		{conf: conf, in: `const a=1,b=2`, out: `const a=1,b=2`},
		{conf: conf, in: `var {a=0}=bar`, out: `var{a=0}=bar`},
		{conf: conf, in: `function foo(a=0){}`, out: `function foo(a=0){}`},
	}
	runTestCases(t, cases)
}

func TestPrettify_IncompleteSwitch(t *testing.T) {
	var conf Config
	cases := []testCase{
		{
			conf: conf,
			in: `
switch (action.type) {
	c
}
`,
			out: `switch(action.type){c}`,
		},
	}

	runTestCases(t, cases)
}

func runTestCases(t *testing.T, cases []testCase) {
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(javascript.GetLanguage())
			src := []byte(c.in)
			tree := parser.Parse(src)
			defer tree.Close()

			var buf bytes.Buffer
			if _, err := Prettify(&buf, c.conf, src, 0, len(src), tree.RootNode()); err != nil {
				t.Fatal(err)
			}
			want, got := c.out, buf.String()
			if want != got {
				t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
			}
		})
	}
}
