package render

import (
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

var kiteDirectiveRE = regexp.MustCompile(`(?m)^ *# *(?:kite|KITE):([^\r\n]*)\r?$`)

func extractExecution(code []byte, raw Raw) (execution.Spec, []byte, error) {
	match := kiteDirectiveRE.FindSubmatchIndex(code)
	if match == nil || match[0] > 0 {
		return execution.Spec{}, code, nil
	}

	directive := code[match[2]:match[3]]

	code = code[match[1]:]
	if len(code) > 0 {
		// newline
		code = code[1:]
	}

	fields := strings.Fields(strings.TrimSpace(string(directive)))
	if len(fields) == 0 {
		return execution.Spec{}, nil, errors.Errorf("empty directive")
	}
	switch strings.ToLower(fields[0]) {
	case "environment":
		if len(fields) > 2 {
			return execution.Spec{}, nil, errors.Errorf("too many arguments to execution directive")
		}
		return raw.Environments[fields[1]], code, nil
	default:
		return execution.Spec{}, nil, errors.Errorf("unrecognized directive: %s", fields[0])
	}
}
