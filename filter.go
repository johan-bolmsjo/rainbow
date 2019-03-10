package main

import (
	"github.com/johan-bolmsjo/saft"
	"regexp"
	"strconv"
	"strings"
)

type filter struct {
	name       string
	regexp     *regexp.Regexp
	regexpFrom *filter
	props      map[int]properties // Properites indexed by regexp group
	filters    filterList
	state      *filterState
}

type properties struct {
	fgcolor, bgcolor color
	modifiers        modifierSet
}

func (props *properties) mergeWith(other properties) {
	if other.fgcolor != colorNone {
		props.fgcolor = other.fgcolor
	}
	if other.bgcolor != colorNone {
		props.bgcolor = other.bgcolor
	}
	props.modifiers |= other.modifiers
}

const filterSep = "/"

func elemParseFilter(elem saft.Elem, prog *program) (*filter, error) {
	assoc, err := elem.ExpectAssoc()
	if err != nil {
		return nil, err
	}
	if err = assocCheckDuplicates(assoc, parFilterName, parFilterRegexp, parFilterRegexpFrom); err != nil {
		return nil, err
	}
	if err = assocCheckExclusive(assoc, parFilterRegexp, parFilterRegexpFrom); err != nil {
		return nil, err
	}

	filter := filter{props: map[int]properties{}}
	var str *saft.String

	for _, p := range assoc.L {
		key := p.K.V
		switch key {
		case parFilterName:
			if str, err = elemExpectString(p.V, key); err != nil {
				return nil, err
			}
			if strings.Contains(str.V, filterSep) {
				return nil, posErrorf(str.Pos(), "filter name must not contain %q", filterSep)
			}
			filter.name = str.V

		case parFilterRegexp:
			if str, err = elemExpectString(p.V, key); err != nil {
				return nil, err
			}
			if filter.regexp, err = regexp.Compile(str.V); err != nil {
				return nil, posWrapError(err, str.Pos())
			}

		case parFilterRegexpFrom:
			if str, err = elemExpectString(p.V, key); err != nil {
				return nil, err
			}
			if filter.regexpFrom = prog.findFilter(str.V); filter.regexpFrom == nil {
				return nil, posErrorf(str.Pos(), "referenced filter %q does not exist", str.V)
			}
			if filter.regexpFrom.regexp == nil {
				return nil, posErrorf(str.Pos(), "referenced filter %q miss regexp", str.V)
			}

		case parFilterProperties:
			if err = elemParseFilterProperties(p.V, key, &filter); err != nil {
				return nil, err
			}

		case parFilter:
			nestedFilter, err := elemParseFilter(p.V, prog)
			if err != nil {
				return nil, err
			}
			if filter.filters.find(nestedFilter.name) != nil {
				return nil, posErrorf(p.V.Pos(), "duplicate filter %q", nestedFilter.name)
			}
			filter.filters = append(filter.filters, nestedFilter)

		default:
			return nil, unknownParameterError(&p)
		}
	}

	filter.state = prog.globalFilterState.allocState()
	return &filter, nil
}

func elemParseFilterProperties(elem saft.Elem, param string, filter *filter) error {
	assoc, err := elemExpectAssoc(elem, param)
	if err != nil {
		return err
	}

	for _, p := range assoc.L {
		var group int
		if group, err = strconv.Atoi(p.K.V); err != nil || group <= 0 {
			return posErrorf(p.K.Pos(), "invalid regexp group %q", p.K.V)
		}

		var props properties
		if props, err = elemParseProperties(p.V); err != nil {
			return err
		}
		filter.props[group] = props
	}
	return nil
}

func elemParseProperties(elem saft.Elem) (properties, error) {
	assoc, err := elem.ExpectAssoc()
	if err != nil {
		return properties{}, err
	}
	if err = assocCheckDuplicates(assoc, parPropertyColor, parPropertyBGColor, parPropertyModifiers); err != nil {
		return properties{}, err
	}

	var props properties

	for _, p := range assoc.L {
		key := p.K.V
		switch key {
		case parPropertyColor:
			if props.fgcolor, err = elemParseColor(p.V, key); err != nil {
				return properties{}, err
			}

		case parPropertyBGColor:
			if props.bgcolor, err = elemParseColor(p.V, key); err != nil {
				return properties{}, err
			}

		case parPropertyModifiers:
			var modifiers []modifier
			if modifiers, err = elemParseModifierList(p.V, key); err != nil {
				return properties{}, err
			}
			for _, modifier := range modifiers {
				props.modifiers.set(modifier)
			}

		default:
			return properties{}, unknownParameterError(&p)
		}
	}
	return props, nil
}

func elemParseColor(elem saft.Elem, param string) (color, error) {
	str, err := elemExpectString(elem, param)
	if err != nil {
		return colorNone, err
	}
	color, err := parseColor(str.V)
	if err != nil {
		return colorNone, posWrapError(err, str.Pos())
	}
	return color, nil
}

func elemParseModifierList(elem saft.Elem, param string) ([]modifier, error) {
	strList, err := elemExpectListOfString(elem, param)
	if err != nil {
		return nil, err
	}

	var modifiers []modifier
	for _, str := range strList {
		modifier, err := parseModifier(str.V)
		if err != nil {
			return nil, posWrapError(err, str.Pos())
		}
		modifiers = append(modifiers, modifier)
	}
	return modifiers, nil
}

type filterList []*filter

func (l *filterList) find(name string) *filter {
	if name != "" {
		for _, f := range *l {
			if name == f.name {
				return f
			}
		}
	}
	return nil
}

func (l *filterList) apply(f func(*filter)) {
	for _, v := range *l {
		f(v)
	}
}
