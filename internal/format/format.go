package format

import (
	"fmt"
	"strings"
)

// FmtPrice formats a USD price:
//   - v >= 1: "$X,XXX.XX" (2 dp, comma thousands separator)
//   - v < 1:  "$0.XXXXXX" (6 dp)
func FmtPrice(v float64) string {
	if v >= 1 {
		intPart := int64(v)
		fracPart := v - float64(intPart)
		return "$" + addCommas(fmt.Sprintf("%d", intPart)) + fmt.Sprintf(".%02d", int64(fracPart*100+0.5))
	}
	return fmt.Sprintf("$0.%06d", int64(v*1e6+0.5))
}

// FmtChange formats a 24 h percentage change as "+X.XX%" or "-X.XX%".
func FmtChange(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f%%", v)
	}
	return fmt.Sprintf("%.2f%%", v)
}

func addCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	var b strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
