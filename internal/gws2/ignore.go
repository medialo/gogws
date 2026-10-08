package gws2

import "regexp"

func IsPathIgnored(path string, patterns []string) bool {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
