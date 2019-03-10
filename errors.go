package main

import (
	"fmt"
	"github.com/johan-bolmsjo/errors"
	"github.com/johan-bolmsjo/saft"
)

func posErrorf(pos saft.LexPos, format string, a ...interface{}) error {
	return fmt.Errorf(pos.String()+": "+format, a...)
}

func posWrapError(err error, pos saft.LexPos) error {
	return errors.Wrap(err, pos.String())
}

func assocCheckDuplicates(assoc *saft.Assoc, keys ...string) error {
	checkDup := map[string]bool{}
	for _, v := range keys {
		checkDup[v] = false
	}

	for _, p := range assoc.L {
		if seen, check := checkDup[p.K.V]; check {
			if seen {
				return posErrorf(p.K.Pos(), "duplicate parameter %q", p.K.V)
			}
			checkDup[p.K.V] = true
		}
	}
	return nil
}

func assocCheckExclusive(assoc *saft.Assoc, keys ...string) error {
	checkExcl := map[string]bool{}
	for _, v := range keys {
		checkExcl[v] = true
	}
	var seen string

	for _, p := range assoc.L {
		if checkExcl[p.K.V] {
			if len(seen) > 0 {
				return posErrorf(p.K.Pos(), "parameters %q and %q are mutually exclusive", p.K.V, seen)
			}
			seen = p.K.V
		}
	}
	return nil
}

func elemExpectString(elem saft.Elem, param string) (*saft.String, error) {
	str, err := elem.ExpectString()
	if err != nil {
		return nil, fmt.Errorf("%s when parsing %q", err, param)
	}
	return str, nil
}

func elemExpectListOfString(elem saft.Elem, param string) (list []*saft.String, err error) {
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

func elemExpectAssoc(elem saft.Elem, param string) (*saft.Assoc, error) {
	assoc, err := elem.ExpectAssoc()
	if err != nil {
		return nil, fmt.Errorf("%s when parsing %q", err, param)
	}
	return assoc, nil
}

func unknownParameterError(pair *saft.Pair) error {
	return posErrorf(pair.K.Pos(), "unknown parameter %q", pair.K.V)
}

func missingParameterError(assoc *saft.Assoc, param string) error {
	return posErrorf(assoc.Pos(), "missing parameter %q", param)
}

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
