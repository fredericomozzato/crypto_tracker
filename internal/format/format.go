package format

import (
	"fmt"
	"strings"
)

// currencyCode returns the uppercase 3-letter currency code.
func currencyCode(currency string) string {
	return strings.ToUpper(currency)
}

// FmtPrice formats a price in the given currency:
//   - v >= 1: "USD X,XXX.XX" (2 dp, comma thousands separator)
//   - v < 1:  "USD 0.XXXXXX" (6 dp)
func FmtPrice(v float64, currency string) string {
	prefix := currencyCode(currency)
	if v >= 1 {
		parts := strings.SplitN(fmt.Sprintf("%.2f", v), ".", 2)
		return prefix + " " + addCommas(parts[0]) + "." + parts[1]
	}
	return fmt.Sprintf("%s %.6f", prefix, v)
}

// FmtChange formats a 24 h percentage change as "+X.XX%" or "-X.XX%".
func FmtChange(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f%%", v)
	}
	return fmt.Sprintf("%.2f%%", v)
}

// FmtMoney formats a holding value as "USD X,XXX.XX" (always 2 dp, thousands separator).
func FmtMoney(v float64, currency string) string {
	parts := strings.SplitN(fmt.Sprintf("%.2f", v), ".", 2)
	return currencyCode(currency) + " " + addCommas(parts[0]) + "." + parts[1]
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
