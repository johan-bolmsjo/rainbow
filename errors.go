package main

import (
	"fmt"

	"github.com/johan-bolmsjo/errors"
	"github.com/johan-bolmsjo/saft"
)

// formatErrorWithPosition returns an error prefixed with the source position.
func formatErrorWithPosition(position saft.LexPos, format string, a ...interface{}) error {
	return fmt.Errorf(position.String()+": "+format, a...)
}

// wrapErrorWithPosition wraps err with the source position.
func wrapErrorWithPosition(err error, position saft.LexPos) error {
	return errors.Wrap(err, position.String())
}

// associationCheckDuplicates reports an error if any of the keys occurs more
// than once in the association list.
func associationCheckDuplicates(association *saft.Assoc, keys ...string) error {
	checkDup := map[string]bool{}
	for _, v := range keys {
		checkDup[v] = false
	}

	for _, p := range association.L {
		if seen, check := checkDup[p.K.V]; check {
			if seen {
				return formatErrorWithPosition(p.K.Pos(), "duplicate parameter %q", p.K.V)
			}
			checkDup[p.K.V] = true
		}
	}
	return nil
}

// associationCheckExclusive reports an error if more than one of the keys is
// present in the association list.
func associationCheckExclusive(association *saft.Assoc, keys ...string) error {
	checkExcl := map[string]bool{}
	for _, v := range keys {
		checkExcl[v] = true
	}
	var seen string

	for _, p := range association.L {
		if checkExcl[p.K.V] {
			if len(seen) > 0 {
				return formatErrorWithPosition(p.K.Pos(), "parameters %q and %q are mutually exclusive", p.K.V, seen)
			}
			seen = p.K.V
		}
	}
	return nil
}

// elementExpectString expects element to be a string and decorates any error
// with the parameter name.
func elementExpectString(elem saft.Elem, param string) (*saft.String, error) {
	str, err := elem.ExpectString()
	if err != nil {
		return nil, fmt.Errorf("%s when parsing %q", err, param)
	}
	return str, nil
}

// elementExpectListOfString expects element to be a string or a list of
// strings and decorates any error with the parameter name. A single string is
// returned as a one element list.
func elementExpectListOfString(elem saft.Elem, param string) (list []*saft.String, err error) {
	if str, ok := elem.IsString(); ok {
		return []*saft.String{str}, nil
	}
	var tmpList *saft.List
	if tmpList, err = elem.ExpectList(); err != nil {
		return nil, fmt.Errorf("%s when parsing %q", err, param)
	}
	for _, elem = range tmpList.L {
		str, err := elem.ExpectString()
		if err != nil {
			return nil, fmt.Errorf("%s when parsing %q", err, param)
		}
		list = append(list, str)
	}
	return list, nil
}

// elementExpectAssociation expects element to be an association list and
// decorates any error with the parameter name.
func elementExpectAssociation(elem saft.Elem, param string) (*saft.Assoc, error) {
	association, err := elem.ExpectAssoc()
	if err != nil {
		return nil, fmt.Errorf("%s when parsing %q", err, param)
	}
	return association, nil
}

// unknownParameterError returns an error for an unknown association parameter.
func unknownParameterError(pair *saft.Pair) error {
	return formatErrorWithPosition(pair.K.Pos(), "unknown parameter %q", pair.K.V)
}

// missingParameterError returns an error for a missing association parameter.
func missingParameterError(association *saft.Assoc, param string) error {
	return formatErrorWithPosition(association.Pos(), "missing parameter %q", param)
}

// decorateErrorWithSource prefixes err with the source name. A space is added
// after the source name unless err already starts with a source position.
func decorateErrorWithSource(err error, source string) error {
	isDigit := func(c byte) bool {
		return c >= '0' && c <= '9'
	}
	errStr := err.Error()
	if len(errStr) > 0 && !isDigit(errStr[0]) {
		errStr = " " + errStr
	}
	return fmt.Errorf("%s:%s", source, errStr)
}
