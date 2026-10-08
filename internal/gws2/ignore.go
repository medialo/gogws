package gws2

import "regexp"

type IgnoreRules struct {
	patterns []*regexp.Regexp
}

func CompileIgnoreRules(patterns []string) IgnoreRules {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		compiled = append(compiled, re)
	}
	return IgnoreRules{patterns: compiled}
}

func (r IgnoreRules) IsEmpty() bool {
	return len(r.patterns) == 0
}

func (r IgnoreRules) Match(path string) bool {
	for _, re := range r.patterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
