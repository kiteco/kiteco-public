package livemetrics

import (
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
)

// TODO parameterize this
// languageMetrics tracks metrics for which language the user is currently using
type languageMetrics struct {
	bashEdit, bashSelect     uint64
	cEdit, cSelect           uint64
	cppEdit, cppSelect       uint64
	csharpEdit, csharpSelect uint64
	cssEdit, cssSelect       uint64
	goEdit, goSelect         uint64
	htmlEdit, htmlSelect     uint64
	javaEdit, javaSelect     uint64
	jsEdit, jsSelect         uint64
	jsxEdit, jsxSelect       uint64
	ktEdit, ktSelect         uint64
	lessEdit, lessSelect     uint64
	objcEdit, objcSelect     uint64
	perlEdit, perlSelect     uint64
	phpEdit, phpSelect       uint64
	pythonEdit, pythonSelect uint64
	rubyEdit, rubySelect     uint64
	scalaEdit, scalaSelect   uint64
	tsEdit, tsSelect         uint64
	tsxEdit, tsxSelect       uint64
	vueEdit, vueSelect       uint64
}

func newLanguageMetrics() *languageMetrics {
	return &languageMetrics{}
}

func (l *languageMetrics) zero() {
	atomic.StoreUint64(&l.bashEdit, 0)
	atomic.StoreUint64(&l.bashSelect, 0)
	atomic.StoreUint64(&l.cEdit, 0)
	atomic.StoreUint64(&l.cSelect, 0)
	atomic.StoreUint64(&l.cppEdit, 0)
	atomic.StoreUint64(&l.cppSelect, 0)
	atomic.StoreUint64(&l.csharpEdit, 0)
	atomic.StoreUint64(&l.csharpSelect, 0)
	atomic.StoreUint64(&l.cssEdit, 0)
	atomic.StoreUint64(&l.cssSelect, 0)
	atomic.StoreUint64(&l.goEdit, 0)
	atomic.StoreUint64(&l.goSelect, 0)
	atomic.StoreUint64(&l.htmlEdit, 0)
	atomic.StoreUint64(&l.htmlSelect, 0)
	atomic.StoreUint64(&l.javaEdit, 0)
	atomic.StoreUint64(&l.javaSelect, 0)
	atomic.StoreUint64(&l.jsEdit, 0)
	atomic.StoreUint64(&l.jsSelect, 0)
	atomic.StoreUint64(&l.jsxEdit, 0)
	atomic.StoreUint64(&l.jsxSelect, 0)
	atomic.StoreUint64(&l.ktEdit, 0)
	atomic.StoreUint64(&l.ktSelect, 0)
	atomic.StoreUint64(&l.lessEdit, 0)
	atomic.StoreUint64(&l.lessSelect, 0)
	atomic.StoreUint64(&l.objcEdit, 0)
	atomic.StoreUint64(&l.objcSelect, 0)
	atomic.StoreUint64(&l.perlEdit, 0)
	atomic.StoreUint64(&l.perlSelect, 0)
	atomic.StoreUint64(&l.phpEdit, 0)
	atomic.StoreUint64(&l.phpSelect, 0)
	atomic.StoreUint64(&l.pythonEdit, 0)
	atomic.StoreUint64(&l.pythonSelect, 0)
	atomic.StoreUint64(&l.rubyEdit, 0)
	atomic.StoreUint64(&l.rubySelect, 0)
	atomic.StoreUint64(&l.scalaEdit, 0)
	atomic.StoreUint64(&l.scalaSelect, 0)
	atomic.StoreUint64(&l.tsEdit, 0)
	atomic.StoreUint64(&l.tsSelect, 0)
	atomic.StoreUint64(&l.tsxEdit, 0)
	atomic.StoreUint64(&l.tsxSelect, 0)
	atomic.StoreUint64(&l.vueEdit, 0)
	atomic.StoreUint64(&l.vueSelect, 0)
}

func (l *languageMetrics) get() map[string]uint64 {
	d := map[string]uint64{
		"bash_edit":         atomic.LoadUint64(&l.bashEdit),
		"bash_select":       atomic.LoadUint64(&l.bashSelect),
		"c_edit":            atomic.LoadUint64(&l.cEdit),
		"c_select":          atomic.LoadUint64(&l.cSelect),
		"cpp_edit":          atomic.LoadUint64(&l.cppEdit),
		"cpp_select":        atomic.LoadUint64(&l.cppSelect),
		"csharp_edit":       atomic.LoadUint64(&l.csharpEdit),
		"csharp_select":     atomic.LoadUint64(&l.csharpSelect),
		"css_edit":          atomic.LoadUint64(&l.cssEdit),
		"css_select":        atomic.LoadUint64(&l.cssSelect),
		"go_edit":           atomic.LoadUint64(&l.goEdit),
		"go_select":         atomic.LoadUint64(&l.goSelect),
		"html_edit":         atomic.LoadUint64(&l.htmlEdit),
		"html_select":       atomic.LoadUint64(&l.htmlSelect),
		"java_edit":         atomic.LoadUint64(&l.javaEdit),
		"java_select":       atomic.LoadUint64(&l.javaSelect),
		"javascript_edit":   atomic.LoadUint64(&l.jsEdit),
		"javascript_select": atomic.LoadUint64(&l.jsSelect),
		"jsx_edit":          atomic.LoadUint64(&l.jsxEdit),
		"jsx_select":        atomic.LoadUint64(&l.jsxSelect),
		"kotlin_edit":       atomic.LoadUint64(&l.ktEdit),
		"kotlin_select":     atomic.LoadUint64(&l.ktSelect),
		"less_edit":         atomic.LoadUint64(&l.lessEdit),
		"less_select":       atomic.LoadUint64(&l.lessSelect),
		"objectivec_edit":   atomic.LoadUint64(&l.objcEdit),
		"objectivec_select": atomic.LoadUint64(&l.objcSelect),
		"perl_edit":         atomic.LoadUint64(&l.perlEdit),
		"perl_select":       atomic.LoadUint64(&l.perlSelect),
		"php_edit":          atomic.LoadUint64(&l.phpEdit),
		"php_select":        atomic.LoadUint64(&l.phpSelect),
		"python_edit":       atomic.LoadUint64(&l.pythonEdit),
		"python_select":     atomic.LoadUint64(&l.pythonSelect),
		"ruby_edit":         atomic.LoadUint64(&l.rubyEdit),
		"ruby_select":       atomic.LoadUint64(&l.rubySelect),
		"scala_edit":        atomic.LoadUint64(&l.scalaEdit),
		"scala_select":      atomic.LoadUint64(&l.scalaSelect),
		"tsx_edit":          atomic.LoadUint64(&l.tsxEdit),
		"tsx_select":        atomic.LoadUint64(&l.tsxSelect),
		"typescript_edit":   atomic.LoadUint64(&l.tsEdit),
		"typescript_select": atomic.LoadUint64(&l.tsSelect),
		"vue_edit":          atomic.LoadUint64(&l.vueEdit),
		"vue_select":        atomic.LoadUint64(&l.vueSelect),
	}
	d["bash_events"] = d["bash_edit"] + d["bash_select"]
	d["c_events"] = d["c_edit"] + d["c_select"]
	d["cpp_events"] = d["cpp_edit"] + d["cpp_select"]
	d["csharp_events"] = d["csharp_edit"] + d["csharp_select"]
	d["css_events"] = d["css_edit"] + d["css_select"]
	d["go_events"] = d["go_edit"] + d["go_select"]
	d["html_events"] = d["html_edit"] + d["html_select"]
	d["java_events"] = d["java_edit"] + d["java_select"]
	d["javascript_events"] = d["javascript_edit"] + d["javascript_select"]
	d["jsx_events"] = d["jsx_edit"] + d["jsx_select"]
	d["kotlin_events"] = d["kotlin_edit"] + d["kotlin_select"]
	d["less_events"] = d["less_edit"] + d["less_select"]
	d["objectivec_events"] = d["objectivec_edit"] + d["objectivec_select"]
	d["perl_events"] = d["perl_edit"] + d["perl_select"]
	d["php_events"] = d["php_edit"] + d["php_select"]
	d["python_events"] = d["python_edit"] + d["python_select"]
	d["ruby_events"] = d["ruby_edit"] + d["ruby_select"]
	d["scala_events"] = d["scala_edit"] + d["scala_select"]
	d["tsx_events"] = d["tsx_edit"] + d["tsx_select"]
	d["typescript_events"] = d["typescript_edit"] + d["typescript_select"]
	d["vue_events"] = d["vue_edit"] + d["vue_select"]
	return d
}

func (l *languageMetrics) dump() map[string]uint64 {
	d := l.get()
	l.zero()
	return d
}

// TrackEvent implements the Tracker interface
func (l *languageMetrics) TrackEvent(evt *event.Event) {
	if evt.GetSource() == "terminal" {
		return
	}

	action := evt.GetAction()
	if action == "skip" {
		action = evt.GetInitialAction()
	}
	if action != "edit" && action != "selection" {
		return
	}

	var incr *uint64
	switch lang.FromFilename(evt.GetFilename()) {
	case lang.Golang:
		if action == "edit" {
			incr = &l.goEdit
		} else {
			incr = &l.goSelect
		}
	case lang.JavaScript:
		if action == "edit" {
			incr = &l.jsEdit
		} else {
			incr = &l.jsSelect
		}
	case lang.Cpp:
		if action == "edit" {
			incr = &l.cppEdit
		} else {
			incr = &l.cppSelect
		}
	case lang.Java:
		if action == "edit" {
			incr = &l.javaEdit
		} else {
			incr = &l.javaSelect
		}
	case lang.Python:
		if action == "edit" {
			incr = &l.pythonEdit
		} else {
			incr = &l.pythonSelect
		}
	case lang.PHP:
		if action == "edit" {
			incr = &l.phpEdit
		} else {
			incr = &l.phpSelect
		}
	case lang.ObjectiveC:
		if action == "edit" {
			incr = &l.objcEdit
		} else {
			incr = &l.objcSelect
		}
	case lang.Scala:
		if action == "edit" {
			incr = &l.scalaEdit
		} else {
			incr = &l.scalaSelect
		}
	case lang.C:
		if action == "edit" {
			incr = &l.cEdit
		} else {
			incr = &l.cSelect
		}
	case lang.CSharp:
		if action == "edit" {
			incr = &l.csharpEdit
		} else {
			incr = &l.csharpSelect
		}
	case lang.Perl:
		if action == "edit" {
			incr = &l.perlEdit
		} else {
			incr = &l.perlSelect
		}
	case lang.Ruby:
		if action == "edit" {
			incr = &l.rubyEdit
		} else {
			incr = &l.rubySelect
		}
	case lang.Bash:
		if action == "edit" {
			incr = &l.bashEdit
		} else {
			incr = &l.bashSelect
		}
	case lang.HTML:
		if action == "edit" {
			incr = &l.htmlEdit
		} else {
			incr = &l.htmlSelect
		}
	case lang.Less:
		if action == "edit" {
			incr = &l.lessEdit
		} else {
			incr = &l.lessSelect
		}
	case lang.CSS:
		if action == "edit" {
			incr = &l.cssEdit
		} else {
			incr = &l.cssSelect
		}
	case lang.JSX:
		if action == "edit" {
			incr = &l.jsxEdit
		} else {
			incr = &l.jsxSelect
		}
	case lang.Vue:
		if action == "edit" {
			incr = &l.vueEdit
		} else {
			incr = &l.vueSelect
		}
	case lang.TypeScript:
		if action == "edit" {
			incr = &l.tsEdit
		} else {
			incr = &l.tsSelect
		}
	case lang.Kotlin:
		if action == "edit" {
			incr = &l.ktEdit
		} else {
			incr = &l.ktSelect
		}
	case lang.TSX:
		if action == "edit" {
			incr = &l.tsxEdit
		} else {
			incr = &l.tsxSelect
		}
	}
	if incr != nil {
		atomic.AddUint64(incr, 1)
	}
}
