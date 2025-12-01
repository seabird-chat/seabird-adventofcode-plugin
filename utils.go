package adventofcode

import "strconv"

func Ordinal(x int) string {
	// Special cases
	switch x % 100 {
	case 11, 12, 13:
		return strconv.Itoa(x) + "th"
	}

	// Match on single digits
	switch x % 10 {
	case 1:
		return strconv.Itoa(x) + "st"
	case 2:
		return strconv.Itoa(x) + "nd"
	case 3:
		return strconv.Itoa(x) + "rd"
	default:
		return strconv.Itoa(x) + "th"
	}
}
