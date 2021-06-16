package rules

import (
	"errors"
	"strings"

	"github.com/vulcand/predicate"
)

const (
	and = "and"
	or  = "or"
)

type treeBuilder func() *tree

// ParseDomains extract domains from rule.
func ParseDomains(rule string) ([]string, error) {
	parser, err := newParser()
	if err != nil {
		return nil, err
	}

	parse, err := parser.Parse(rule)
	if err != nil {
		return nil, err
	}

	buildTree, ok := parse.(treeBuilder)
	if !ok {
		return nil, errors.New("cannot parse")
	}

	return lower(parseDomain(buildTree())), nil
}

// ParseHostSNI extracts the HostSNIs declared in a rule.
// This is a first naive implementation used in TCP routing.
func ParseHostSNI(rule string) ([]string, error) {
	parser, err := newTCPParser()
	if err != nil {
		return nil, err
	}

	parse, err := parser.Parse(rule)
	if err != nil {
		return nil, err
	}

	buildTree, ok := parse.(treeBuilder)
	if !ok {
		return nil, errors.New("cannot parse")
	}

	return lower(parseDomain(buildTree())), nil
}

func lower(slice []string) []string {
	var lowerStrings []string
	for _, value := range slice {
		lowerStrings = append(lowerStrings, strings.ToLower(value))
	}
	return lowerStrings
}

func parseDomain(tree *tree) []string {
	switch tree.matcher {
	case and, or:
		return append(parseDomain(tree.ruleLeft), parseDomain(tree.ruleRight)...)
	case "Host", "HostSNI":
		return tree.value
	default:
		return nil
	}
}

func andFunc(left, right treeBuilder) treeBuilder {
	return func() *tree {
		return &tree{
			matcher:   and,
			ruleLeft:  left(),
			ruleRight: right(),
		}
	}
}

func orFunc(left, right treeBuilder) treeBuilder {
	return func() *tree {
		return &tree{
			matcher:   or,
			ruleLeft:  left(),
			ruleRight: right(),
		}
	}
}

func invert(t *tree) *tree {
	switch t.matcher {
	case or:
		t.matcher = and
		t.ruleLeft = invert(t.ruleLeft)
		t.ruleRight = invert(t.ruleRight)
	case and:
		t.matcher = or
		t.ruleLeft = invert(t.ruleLeft)
		t.ruleRight = invert(t.ruleRight)
	default:
		t.not = !t.not
	}

	return t
}

func notFunc(elem treeBuilder) treeBuilder {
	return func() *tree {
		return invert(elem())
	}
}

func newParser() (predicate.Parser, error) {
	parserFuncs := make(map[string]interface{})

	for matcherName := range funcs {
		matcherName := matcherName
		fn := func(value ...string) treeBuilder {
			return func() *tree {
				return &tree{
					matcher: matcherName,
					value:   value,
				}
			}
		}
		parserFuncs[matcherName] = fn
		parserFuncs[strings.ToLower(matcherName)] = fn
		parserFuncs[strings.ToUpper(matcherName)] = fn
		parserFuncs[strings.Title(strings.ToLower(matcherName))] = fn
	}

	return predicate.NewParser(predicate.Def{
		Operators: predicate.Operators{
			AND: andFunc,
			OR:  orFunc,
			NOT: notFunc,
		},
		Functions: parserFuncs,
	})
}

func newTCPParser() (predicate.Parser, error) {
	parserFuncs := make(map[string]interface{})

	// FIXME quircky way of waiting for new rules
	matcherName := "HostSNI"
	fn := func(value ...string) treeBuilder {
		return func() *tree {
			return &tree{
				matcher: matcherName,
				value:   value,
			}
		}
	}
	parserFuncs[matcherName] = fn
	parserFuncs[strings.ToLower(matcherName)] = fn
	parserFuncs[strings.ToUpper(matcherName)] = fn
	parserFuncs[strings.Title(strings.ToLower(matcherName))] = fn

	return predicate.NewParser(predicate.Def{
		Operators: predicate.Operators{
			OR: orFunc,
		},
		Functions: parserFuncs,
	})
}
