package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/johan-bolmsjo/saft"
)

// filter matches a line with a regexp and applies properties to the matched
// groups. It may also reference the regexp of another filter and contain nested
// filters.
type filter struct {
	name       string
	regexp     *regexp.Regexp
	regexpFrom *filter
	props      map[int]properties // Properties indexed by regexp group
	filters    filterList
	state      *filterState
}

// properties is the coloring applied to a matched regexp group.
type properties struct {
	fgcolor, bgcolor color
	modifiers        modifierSet
}

// mergeWith merges the set fields of other into the properties. A color of colorNone
// leaves the existing color unchanged.
func (props *properties) mergeWith(other properties) {
	if other.fgcolor != colorNone {
		props.fgcolor = other.fgcolor
	}
	if other.bgcolor != colorNone {
		props.bgcolor = other.bgcolor
	}
	props.modifiers |= other.modifiers
}

// filterSep separates the names of nested filters.
const filterSep = "/"

// elementParseFilter parses a filter association list.
func elementParseFilter(elem saft.Elem, prog *program) (*filter, error) {
	association, err := elem.ExpectAssoc()
	if err != nil {
		return nil, err
	}
	if err = associationCheckDuplicates(association, parFilterName, parFilterRegexp, parFilterRegexpFrom, parFilterProperties); err != nil {
		return nil, err
	}
	if err = associationCheckExclusive(association, parFilterRegexp, parFilterRegexpFrom); err != nil {
		return nil, err
	}

	filter := filter{props: map[int]properties{}}
	var str *saft.String

	for _, p := range association.L {
		key := p.K.V
		switch key {
		case parFilterName:
			if str, err = elementExpectString(p.V, key); err != nil {
				return nil, err
			}
			if strings.Contains(str.V, filterSep) {
				return nil, formatErrorWithPosition(str.Pos(), "filter name must not contain %q", filterSep)
			}
			filter.name = str.V

		case parFilterRegexp:
			if str, err = elementExpectString(p.V, key); err != nil {
				return nil, err
			}
			if filter.regexp, err = regexp.Compile(str.V); err != nil {
				return nil, wrapErrorWithPosition(err, str.Pos())
			}

		case parFilterRegexpFrom:
			if str, err = elementExpectString(p.V, key); err != nil {
				return nil, err
			}
			if filter.regexpFrom = prog.findFilter(str.V); filter.regexpFrom == nil {
				return nil, formatErrorWithPosition(str.Pos(), "referenced filter %q does not exist", str.V)
			}
			if filter.regexpFrom.regexp == nil {
				return nil, formatErrorWithPosition(str.Pos(), "referenced filter %q has no regexp", str.V)
			}

		case parFilterProperties:
			if err = elementParseFilterProperties(p.V, key, &filter); err != nil {
				return nil, err
			}

		case parFilter:
			nestedFilter, err := elementParseFilter(p.V, prog)
			if err != nil {
				return nil, err
			}
			if filter.filters.find(nestedFilter.name) != nil {
				return nil, formatErrorWithPosition(p.V.Pos(), "duplicate filter %q", nestedFilter.name)
			}
			filter.filters = append(filter.filters, nestedFilter)

		default:
			return nil, unknownParameterError(&p)
		}
	}

	// Validate that all property group numbers exist in the regexp.
	// Parameters are order-independent, so validation is performed after
	// parsing all of them.
	if len(filter.props) > 0 {
		re := filter.regexp
		if re == nil && filter.regexpFrom != nil {
			re = filter.regexpFrom.regexp
		}
		if re == nil {
			return nil, formatErrorWithPosition(association.Pos(), "properties set but filter has no regexp")
		}
		for group := range filter.props {
			if group > re.NumSubexp() {
				return nil, formatErrorWithPosition(association.Pos(),
					"invalid regexp group %d, regexp has %d group(s)", group, re.NumSubexp())
			}
		}
	}

	filter.state = prog.globalFilterState.allocateState()
	return &filter, nil
}

// elementParseFilterProperties parses the properties of a filter, keyed by
// regexp group number.
func elementParseFilterProperties(elem saft.Elem, param string, filter *filter) error {
	association, err := elementExpectAssociation(elem, param)
	if err != nil {
		return err
	}

	seenGroup := map[int]bool{}
	for _, p := range association.L {
		var group int
		if group, err = strconv.Atoi(p.K.V); err != nil || group <= 0 {
			return formatErrorWithPosition(p.K.Pos(), "invalid regexp group %q", p.K.V)
		}
		if seenGroup[group] {
			return formatErrorWithPosition(p.K.Pos(), "duplicate regexp group %q", p.K.V)
		}
		seenGroup[group] = true

		var props properties
		if props, err = elementParseProperties(p.V); err != nil {
			return err
		}
		filter.props[group] = props
	}
	return nil
}

// elementParseProperties parses a property association list.
func elementParseProperties(elem saft.Elem) (properties, error) {
	association, err := elem.ExpectAssoc()
	if err != nil {
		return properties{}, err
	}
	if err = associationCheckDuplicates(association, parPropertyColor, parPropertyBGColor, parPropertyModifiers); err != nil {
		return properties{}, err
	}

	var props properties

	for _, p := range association.L {
		key := p.K.V
		switch key {
		case parPropertyColor:
			if props.fgcolor, err = elementParseColor(p.V, key); err != nil {
				return properties{}, err
			}

		case parPropertyBGColor:
			if props.bgcolor, err = elementParseColor(p.V, key); err != nil {
				return properties{}, err
			}

		case parPropertyModifiers:
			var modifiers []modifier
			if modifiers, err = elementParseModifierList(p.V, key); err != nil {
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

// elementParseColor parses a color value.
func elementParseColor(elem saft.Elem, param string) (color, error) {
	str, err := elementExpectString(elem, param)
	if err != nil {
		return colorNone, err
	}
	color, err := parseColor(str.V)
	if err != nil {
		return colorNone, wrapErrorWithPosition(err, str.Pos())
	}
	return color, nil
}

// elementParseModifierList parses a modifier or a list of modifiers.
func elementParseModifierList(elem saft.Elem, param string) ([]modifier, error) {
	strList, err := elementExpectListOfString(elem, param)
	if err != nil {
		return nil, err
	}

	var modifiers []modifier
	for _, str := range strList {
		modifier, err := parseModifier(str.V)
		if err != nil {
			return nil, wrapErrorWithPosition(err, str.Pos())
		}
		modifiers = append(modifiers, modifier)
	}
	return modifiers, nil
}

// filterList is a list of filters.
type filterList []*filter

// find returns the filter with the given name, or nil. An empty name never
// matches.
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

// apply calls f for every filter in the list.
func (l *filterList) apply(f func(*filter)) {
	for _, v := range *l {
		f(v)
	}
}
