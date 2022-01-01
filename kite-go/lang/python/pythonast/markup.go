package pythonast

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
)

// MarkupFunc returns the strings that should be inserted before and after each AST node
type MarkupFunc func(n Node) (begin, end string)

type markupContext struct {
	w      io.Writer
	markup MarkupFunc
	src    []byte
	offset int
}

func (ctx *markupContext) consumeTo(pos int) {
	if pos > len(ctx.src) {
		// this happens because the parser adds a newline at the end
		pos = len(ctx.src)
	}
	if pos > ctx.offset {
		ctx.w.Write(ctx.src[ctx.offset:pos])
		ctx.offset = pos
	}
}

func (ctx *markupContext) finish() {
	ctx.w.Write(ctx.src[ctx.offset:])
}

type markupVisitor struct {
	cur Node
	end string
	ctx *markupContext
}

// Visit consumes each AST node
func (v *markupVisitor) Visit(n Node) Visitor {
	if n == nil && v.cur != nil {
		v.ctx.consumeTo(int(v.cur.End()))
		fmt.Fprint(v.ctx.w, v.end)
		return nil
	}

	begin, end := v.ctx.markup(n)
	v.ctx.consumeTo(int(n.Begin()))
	fmt.Fprint(v.ctx.w, begin)
	return &markupVisitor{
		cur: n,
		end: end,
		ctx: v.ctx,
	}
}

// Markup inserts strings around each AST node
func Markup(src []byte, root Node, markup MarkupFunc) template.HTML {
	var buf bytes.Buffer
	v := markupVisitor{ctx: &markupContext{
		w:      &buf,
		markup: markup,
		src:    src,
	}}
	Walk(&v, root)
	v.ctx.finish()
	return template.HTML(buf.String())
}
