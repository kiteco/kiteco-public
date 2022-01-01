package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// walkSubclasses calls the walk function for each subclass of the given class
func walkSubclasses(ctx kitectx.Context, c *pythontype.SourceClass, f func(*pythontype.SourceClass) bool) {
	ctx.CheckAbort()

	if f(c) {
		for _, subclass := range c.Subclasses {
			walkSubclasses(ctx, subclass, f)
		}
	}
}
