package buildid

import (
	"fmt"
	"regexp"
	"strconv"
)

var copyReg = regexp.MustCompile(`(?i)^(.*?)-copy(\d*)$`)

// NextCopyID determines the next available unique ID when candidate already exists.
// If candidate already has a "-copy<N>" suffix, it auto-detects it and increments N.
// Otherwise, it starts with "-copy1" and increments until isTaken returns false.
func NextCopyID(candidate string, isTaken func(string) (bool, error)) (string, error) {
	base := candidate
	nextNum := 1

	if m := copyReg.FindStringSubmatch(candidate); len(m) == 3 {
		if m[1] != "" {
			base = m[1]
		} else {
			base = "build"
		}
		if m[2] != "" {
			if n, err := strconv.Atoi(m[2]); err == nil {
				nextNum = n + 1
			}
		}
	}

	for {
		target := fmt.Sprintf("%s-copy%d", base, nextNum)
		taken, err := isTaken(target)
		if err != nil {
			return "", err
		}
		if !taken {
			return target, nil
		}
		nextNum++
	}
}
