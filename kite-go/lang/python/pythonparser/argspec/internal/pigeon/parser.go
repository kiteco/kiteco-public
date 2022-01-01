//go:generate pigeon -o gen_parser.go parser.peg

package pigeon

import (
	"bytes"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func grammarAction(c *current, spec *pythonimports.ArgSpec) (*pythonimports.ArgSpec, error) {
	// process the optional delimiters: when we encounter a '[', all arguments within this
	// and the next ']' are optional, so we set DefaultValue to "..." unless one is already set.
	var optional, keywordOnly bool
	remove := make(map[int]bool)
	for i, arg := range spec.Args {
		if s := strings.TrimSpace(arg.Name); s == "/" || s == "*" {
			optional = true
			remove[i] = true
			continue
		}

		arg.KeywordOnly = keywordOnly
		if strings.HasPrefix(arg.Name, "[") {
			optional = true
			arg.Name = arg.Name[1:]
		} else if strings.HasPrefix(arg.Name, "]") {
			if len(remove) == 0 {
				optional = false
			}
			arg.Name = arg.Name[1:]
		}
		if optional && arg.DefaultValue == "" {
			arg.DefaultValue = "..."
		}
		if strings.HasSuffix(arg.Name, "[") {
			optional = true
			arg.Name = arg.Name[:len(arg.Name)-1]
		} else if strings.HasSuffix(arg.Name, "]") {
			if len(remove) == 0 {
				optional = false
			}
			arg.Name = arg.Name[:len(arg.Name)-1]
		}
		// set the arg back on the spec, as this is a struct by value, not a pointer
		spec.Args[i] = arg
	}

	if len(remove) > 0 {
		var keep []pythonimports.Arg
		for i, arg := range spec.Args {
			if remove[i] {
				continue
			}
			keep = append(keep, arg)
		}
		spec.Args = keep
	}

	// if the last arguments are vararg or kwarg, move them to the appropriate
	// fields on ArgSpec and remove them from the Arg array.
	for i := len(spec.Args) - 1; i >= 0; i-- {
		arg := spec.Args[i]
		if !strings.HasPrefix(arg.Name, "*") {
			// done processing star args
			break
		}
		if strings.HasPrefix(arg.Name, "**") {
			if spec.Kwarg != "" {
				// already processed kwargs, this one is invalid, leave it
				// as standard arg and stop processing.
				break
			}
			spec.Kwarg = arg.Name[2:]
			spec.Args = spec.Args[:len(spec.Args)-1]
		} else {
			spec.Vararg = arg.Name[1:]
			spec.Args = spec.Args[:len(spec.Args)-1]
			// done processing star args, kwarg cannot appear before vararg
			break
		}
	}
	return spec, nil
}

func argSpecAction(c *current, args []pythonimports.Arg) (*pythonimports.ArgSpec, error) {
	return &pythonimports.ArgSpec{
		Args: args,
	}, nil
}

func idListAction(c *current, first string, rest []interface{}) (string, error) {
	// fast path if no rest
	if len(rest) == 0 {
		return first, nil
	}

	var buf bytes.Buffer
	buf.WriteString(first)
	for _, v := range rest {
		// parts[0] == '.', parts[1] == ID
		parts := toIfaceSlice(v)
		if len(parts) != 2 {
			panic("expected len(parts) == 2")
		}
		buf.WriteString("." + parts[1].(string))
	}
	return buf.String(), nil
}

func argsDeclAction(c *current, list interface{}) ([]pythonimports.Arg, error) {
	if list == nil {
		return nil, nil
	}
	return list.([]pythonimports.Arg), nil
}

func argsListAction(c *current, first pythonimports.Arg, rest []interface{}) ([]pythonimports.Arg, error) {
	args := make([]pythonimports.Arg, 0, len(rest)+1)
	args = append(args, first)
	for _, v := range rest {
		// parts[0] == W*, parts[1] == ',', parts[2] == W*, parts[3] == Arg
		parts := toIfaceSlice(v)
		if len(parts) != 4 {
			panic("expected len(parts) == 4")
		}
		args = append(args, parts[3].(pythonimports.Arg))
	}
	return args, nil
}

func argDefAction(c *current, name interface{}, value []interface{},
	startDelim, endDelim string) (pythonimports.Arg, error) {

	var defName string
	switch n := name.(type) {
	case []interface{}:
		// n[0] == nil or []byte representing "*" or "**", n[1] == ID (string)
		if len(n) != 2 {
			panic("expected len(n) == 2")
		}

		var stars string
		if n[0] != nil {
			stars = string(n[0].([]byte))
		}
		id := n[1].(string)
		// put the stars in the name of the argument, will be moved out
		// to ArgSpec.Vararg or ArgSpec.Kwarg at the end of parsing.
		defName = stars + id
	case []byte:
		// this is the ellipsis
		defName = string(n)
	}
	defName = startDelim + defName + endDelim

	var defVal string
	if value != nil {
		// value[0] == W*, value[1] == '=', value[2] == Literal
		if len(value) != 3 {
			panic("expected len(value) == 3")
		}
		defVal = value[2].(string)
	}

	return pythonimports.Arg{
		Name:         defName,
		DefaultValue: defVal,
	}, nil
}

// toIfaceSlice is a helper function for the PEG grammar parser. It converts
// v to a slice of empty interfaces.
func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}
