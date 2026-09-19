package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/johan-bolmsjo/errors"
	"github.com/johan-bolmsjo/rainbow/internal/igor"
	"github.com/johan-bolmsjo/saft"
)

// program is a parsed rainbow configuration.
type program struct {
	name              string
	globalFilterState globalFilterState
	filters           filterList
	stms              []*apply
	interp            *igor.Interpreter
}

// apply applies its filters when the condition evaluates to true. A nil
// condition always evaluates to true.
type apply struct {
	cond    *igor.Condition // Apply filters if expression evaluates to true
	filters filterList
}

// loadProgram reads and parses the configuration in filename.
func loadProgram(filename string) (*program, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	prog, err := createProgram(file)
	if err != nil {
		return nil, decorateErrorWithSource(err, filename)
	}
	prog.name = filename
	return prog, nil
}

// createProgram parses a program from reader.
func createProgram(reader io.Reader) (*program, error) {
	elems, err := saft.Parse(reader)
	if err != nil {
		return nil, err
	}

	if len(elems) == 0 {
		return nil, errors.New("expected one association list")
	}
	if len(elems) > 1 {
		return nil, formatErrorWithPosition(elems[1].Pos(), "trailing data")
	}
	root, err := elems[0].ExpectAssoc()
	if err != nil {
		return nil, err
	}

	prog := program{
		name:   "<stream>",
		interp: igor.NewInterpreter(),
	}

	prog.interp.RegisterFunction("filter-match?", func(args []igor.Object) igor.Object {
		for i, arg := range args {
			if str, ok := arg.(igor.ObjectString); ok {
				filter := prog.findFilter(string(str))
				if filter == nil {
					igor.Throw(igor.ExceptionInvalidArgument(i, fmt.Sprintf("missing filter %q", string(str))))
				}
				if filter.state.matched {
					return igor.ObjectBool(true)
				}
			} else {
				igor.Throw(igor.ExceptionTypeError(arg, i, igor.TypeString))
			}
		}
		return igor.ObjectBool(false)
	})

	prog.interp.RegisterFunction("filter-result", func(args []igor.Object) igor.Object {
		if len(args) != 2 {
			igor.Throw(igor.ExceptionInvalidNumberOfArguments(len(args), "2"))
		}
		var strArgs [2]string
		for i, arg := range args {
			if arg, ok := arg.(igor.ObjectString); ok {
				strArgs[i] = string(arg)
			} else {
				igor.Throw(igor.ExceptionTypeError(arg, i, igor.TypeString))
			}
		}

		filter := prog.findFilter(strArgs[0])
		if filter == nil {
			igor.Throw(igor.ExceptionInvalidArgument(0, fmt.Sprintf("missing filter %q", strArgs[0])))
		}

		idx, err := strconv.Atoi(strArgs[1])
		if err != nil {
			// Maybe generate a user visible error if a proper numeric type is introduced.
			return igor.ObjectStringList(nil)
		}

		return filter.state.valueMatchResult(idx)
	})

	for _, p := range root.L {
		switch p.K.V {
		case parFilter:
			if err := prog.parseFilter(p.V); err != nil {
				return nil, err
			}
		case parApply:
			if err := prog.parseApply(p.V); err != nil {
				return nil, err
			}
		default:
			return nil, unknownParameterError(&p)
		}
	}

	if len(prog.filters) == 0 {
		return nil, missingParameterError(root, parFilter)
	}
	if len(prog.stms) == 0 {
		return nil, missingParameterError(root, parApply)
	}

	return &prog, nil
}

// parseFilter parses and adds a top level filter to the program.
func (prog *program) parseFilter(elem saft.Elem) error {
	filter, err := elementParseFilter(elem, prog)
	if err == nil {
		if prog.findFilter(filter.name) != nil {
			return formatErrorWithPosition(elem.Pos(), "duplicate filter %q", filter.name)
		}
		prog.filters = append(prog.filters, filter)
	}
	return err
}

// findFilter returns the filter at the given slash separated path, or nil.
func (prog *program) findFilter(name string) *filter {
	var filter *filter
	filters := prog.filters
	for _, s := range strings.Split(name, filterSep) {
		if filter = filters.find(s); filter != nil {
			filters = filter.filters
		} else {
			break
		}
	}
	return filter
}

// parseApply parses and adds an apply statement to the program.
func (prog *program) parseApply(elem saft.Elem) error {
	association, err := elem.ExpectAssoc()
	if err != nil {
		return err
	}
	if err = associationCheckDuplicates(association, parApplyCond, parApplyFilters); err != nil {
		return err
	}

	var apply apply

	for _, p := range association.L {
		key := p.K.V
		switch key {
		case parApplyCond:
			if apply.cond, err = prog.interp.CompileCondition(p.V); err != nil {
				return err
			}

		case parApplyFilters:
			strList, err := elementExpectListOfString(p.V, key)
			if err != nil {
				return err
			}
			for _, str := range strList {
				filter := prog.findFilter(str.V)
				if filter == nil {
					return formatErrorWithPosition(str.Pos(), "referenced filter %q does not exist", str.V)
				}
				apply.filters = append(apply.filters, filter)
			}

		default:
			return unknownParameterError(&p)
		}
	}

	if len(apply.filters) == 0 {
		return missingParameterError(association, parApplyFilters)
	}

	prog.stms = append(prog.stms, &apply)
	return nil
}

const (
	parFilter            = "filter"
	parFilterName        = "name"
	parFilterRegexp      = "regexp"
	parFilterRegexpFrom  = "regexpFrom"
	parFilterProperties  = "properties"
	parPropertyColor     = "color"
	parPropertyBGColor   = "bgcolor"
	parPropertyModifiers = "modifiers"
	parApply             = "apply"
	parApplyCond         = "cond"
	parApplyFilters      = "filters"
)
