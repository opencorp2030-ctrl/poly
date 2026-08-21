// Package version compares version strings the way package managers
// mean "newer": numerically across dotted segments, so "10.0.0" ranks
// above "9.9.9" (a plain string comparison would say the opposite).
package version

import (
	"strconv"
	"strings"
)

// Compare returns -1, 0, or +1 depending on whether a is older than,
// equal to, or newer than b.
//
// Each version is split on ".", "-", and "_"; a leading "v" is ignored
// (Go module versions like "v1.2.3"). Segments that are purely numeric
// compare numerically and rank above non-numeric ones ("1.0.0" > "1.0.0-rc1");
// non-numeric segments compare lexicographically. Segments missing from
// one side count as empty.
func Compare(a, b string) int {
	as, bs := split(a), split(b)
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv string
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if c := cmpSegment(av, bv); c != 0 {
			return c
		}
	}
	return 0
}

// Newer reports whether a is a strictly newer version than b.
func Newer(a, b string) bool { return Compare(a, b) > 0 }

func split(v string) []string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	return strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
}

func cmpSegment(a, b string) int {
	// A missing segment is the empty string; treat it as "0" so that
	// "1.2" == "1.2.0" and, crucially, a release ("1.0.0") outranks its
	// own prerelease ("1.0.0-rc1") the way semver says it should.
	if a == "" {
		a = "0"
	}
	if b == "" {
		b = "0"
	}
	if a == b {
		return 0
	}
	an, aErr := strconv.Atoi(a)
	bn, bErr := strconv.Atoi(b)
	switch {
	case aErr == nil && bErr == nil:
		switch {
		case an < bn:
			return -1
		case an > bn:
			return 1
		}
		return 0
	case aErr == nil:
		return 1 // numeric > non-numeric
	case bErr == nil:
		return -1
	default:
		if a < b {
			return -1
		}
		return 1
	}
}
