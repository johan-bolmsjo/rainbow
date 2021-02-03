# Rainbow

Rainbow is a log file colorer that act as a stream processor. Match and action
rules are applied according to configuration to each line read from stdin,
outputting them to stdout.

One instance where this tool may be useful is when developing applications that
log lots of data to traditional log files, maybe an embedded system. When you
are looking for certain patterns, coloring may greatly speed up troubleshooting.
A custom configuration is quickly created to troubleshoot particular issues.

## Design Goals

* Low latency, output buffering is per line only.
* Good performance. Care has been taken to minimize memory allocations.
  Currently most time is spent in GOs regexp package.
* Easy to create customized log filters.

## Usage

Configuration files are searched for in `~/.config/rainbow/` with the suffix
`.rainbow` automatically added. e.g. the command `rainbow example` would try to
load the config file `~/.config/rainbow/example.rainbow`. Alternatively a full
path to a config file can be specified using the `-config` flag.

### Command-line Flags

    -help         Show help
    -color        Force color for non-TTY output
    -config FILE  Use config FILE
    CONFIG        Use config from ~/.config/rainbow/CONFIG.rainbow 

### Example Usage

    go build
    ./rainbow -config testdata/config/example.rainbow < testdata/logs/example.log

A rather silly example but gives an idea about what the tool can do.

[Screenshot](screenshot.png)

## Configuration

The overall configuration file syntax is described at
<https://github.com/johan-bolmsjo/saft/blob/master/README.md>.

### Filters

Filters specify line matching in the form of a regexp and what coloring to
perform uppon match. When an executed filter matches a line the fact that it
matched and the result of the match is saved. The match status can be used to
implement primitive flow control for what filters are to be applied.

Filters can be nested to group filters to apply. Nested filters can be
referred to using '/' as path separator (if they are named).

#### Definition

    filter: { ... }

#### Parameters

    name: NAME
      Optional filter name to be able to apply or reference filter.

    regexp: REGEXP
      Regular expression of filter.

    regexpFrom: REFERENCE
      Use regexp defined by other filter.
      The result is reused if the regexp has already been executed.

    properties: { REGEXP_GROUP: { PROPERTIES } }
      Properties to apply to individually matched regexp groups.

    filter: { ... }
      Nested filters

#### Parameter Values

    NAME:
      May not contain '/' as it is used to reference nested filters.

    REGEXP:
      See https://golang.org/pkg/regexp/syntax/ for syntax.

    REFERENCE:
      Reference to named filter using '/' as nested filter separator.
      e.g. 'logLevel/debug'.

    REGEXP_GROUP:
      Positive integer value identifying the matched regexp group counting
      from 1. Groups are counted from the left with each opening parenthesis.

    PROPERTIES:
      See [Filter Properties]

### Filter Properties

Regexp match properties.

#### Parameters

    color: COLOR
      Foreground color.

    bgcolor: COLOR
      Background color.

    modifiers: MODIFIER | [MODIFIER ... ]
      A single modifier or a list of modifiers.

#### Parameter Values

    COLOR:
       black  red  green  yellow  blue  magenta  cyan  white
      iblack ired igreen iyellow iblue imagenta icyan iwhite

      ANSI defines 8 terminal colors and intense versions of the same. Intense
      black yields a grayish color. Intense colors are prefixed with "i".

    MODIFIER:
      bold underline reverse blink

### Applying Filters

The order in which filters are applied are specified by apply clauses.

#### Definition

    apply: { ... }

#### Parameters

    cond: EXPR
      A lisp like expression that must evaluate to true for the apply clause
      to be applied.

    filters: FILTER | [FILTER ...]
      A single filter or a list of filters to apply.

#### Condition Expression

A number of built in functions are available to build an expression that
evaluates to true or false.

Functions are lisp like in that the first list element is the function and
the following elements its arguments.

    [not arg]
      Evaluates to true if arg is considered false, else false

    [and arg...]
      Evaluates to true with zero arguments.
      Evaluates to the first argument considered false or the last argument.

    [or arg...]
      Evaluates to false with zero arguments.
      Evaluates to the first argument considered true or the last argument.

    [equal? arg1 arg2]
      Evaluates to true if arg1 and arg2 are considered equal, else false.

    [filter-match? filterName...]
      Evaluates to true if any of the listed filters regexp matched, else false.

    [filter-result filterName idx]
      Get filter regexp match result as a list of strings of all matched regexp
      groups, idx=0 is current match, idx=1 is previous

### Example Config File

Config file for coloring "testdata/config/example.rainbow".

    {
        filter: {
            name: logLevel
    
            filter: {
                regexp: `\d \[((EMERG|ALERT|CRIT)|(ERROR)|(WARN))\]`
                properties: {
                    2: { // emerg...
                        color:     white
                        bgcolor:   red
                        modifiers: bold
                    }
                    3: { // error
                        color:   white
                        bgcolor: red
                    }
                    4: { // warn
                        color:   black
                        bgcolor: yellow
                    }
                }
            }
            // Color whole INFO logs gray as base color.
            filter: {
                name:   info
                regexp: `^([\d-:\. ]+ \[INFO\].*)$`
                properties: {
                    1: {
                        color: iblack
                    }
                }
            }
            // Color whole DEBUG logs cyan as base color.
            filter: {
                name:   debug
                regexp: `^([\d-:\. ]+ \[DEBUG\].*)$`
                properties: {
                    1: {
                        color: cyan
                    }
                }
            }
        }
    
        filter: {
            name:   time
            regexp: `^\d{4}-\d{2}-\d{2} (\d{2}:\d{2}:\d{2})`
        }
        filter: {
            name:       timeHighlight
            regexpFrom: time
            properties: {
                1: {
                    modifiers: bold
                }
            }
        }
        filter: {
            name:   domain
            regexp: `\[\w+\]\s+(\w+):`
            properties: {
                1: {
                    modifiers: bold
                }
            }
        }
        filter: {
            name:   variable
            regexp: `([\w]+)=`
            properties: {
                1: {
                    modifiers: bold
                }
            }
        }
    
        apply: {
            filters: [time logLevel]
        }
        apply: {
            cond:    [and [filter-match? time]
                          [not [equal? [filter-result time 0] [filter-result time 1]]]]
            filters: timeHighlight
        }
        apply: {
            cond:    [not [filter-match? logLevel/info logLevel/debug]]
            filters: domain
        }
        apply: {
            filters: variable
        }
    }
