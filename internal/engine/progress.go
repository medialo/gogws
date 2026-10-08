package engine

import (
	"regexp"
	"strconv"
	"strings"
)

type Progress struct {
	Phase   string
	Percent int
	Current int
	Total   int
}

var progressRe = regexp.MustCompile(`^(?:remote: )?([A-Za-z][A-Za-z ]*?):\s+(\d+)% \((\d+)/(\d+)\)`)

func parseProgress(line string) (Progress, bool) {
	match := progressRe.FindStringSubmatch(strings.TrimSpace(line))
	if match == nil {
		return Progress{}, false
	}
	percent, _ := strconv.Atoi(match[2])
	current, _ := strconv.Atoi(match[3])
	total, _ := strconv.Atoi(match[4])
	return Progress{Phase: match[1], Percent: percent, Current: current, Total: total}, true
}
