package markup

import (
	"bytes"
	"html/template"
	"sort"
)

type insertion struct {
	pos   int
	index int
	str   string
	open  bool
}

type byPos []insertion

func (xs byPos) Len() int      { return len(xs) }
func (xs byPos) Swap(i, j int) { xs[i], xs[j] = xs[j], xs[i] }
func (xs byPos) Less(i, j int) bool {
	// first sort by position
	if xs[i].pos != xs[j].pos {
		return xs[i].pos < xs[j].pos
	}

	// close tags always go before open tags when positions are equal
	if !xs[i].open && xs[j].open {
		return true
	} else if xs[i].open && !xs[j].open {
		return false
	}

	// otherwise sort by index: increasing for open tags, decreasing for close tags
	if xs[i].open {
		return xs[i].index < xs[j].index
	}
	return xs[i].index > xs[j].index
}

// Markupper marks up strings with begin and end tags
type Markupper struct {
	insertions []insertion
}

// Add appends an open and close tag to the list of insertions
func (m *Markupper) Add(openpos, closepos int, opentag, closetag string) {
	index := len(m.insertions)
	m.insertions = append(m.insertions, insertion{
		pos:   openpos,
		index: index,
		str:   opentag,
		open:  true,
	})
	m.insertions = append(m.insertions, insertion{
		pos:   closepos,
		index: index,
		str:   closetag,
		open:  false,
	})
}

// Render generates HTML by inserting tags into the given buffer
func (m *Markupper) Render(buf []byte) template.HTML {
	var b bytes.Buffer
	var pos int
	sort.Sort(byPos(m.insertions))
	for _, ins := range m.insertions {
		if pos < ins.pos {
			b.Write(buf[pos:ins.pos])
			pos = ins.pos
		}
		b.WriteString(ins.str)
	}
	if pos < len(buf) {
		b.Write(buf[pos:])
	}
	return template.HTML(b.String())
}
