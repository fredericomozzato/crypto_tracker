package api

import "sort"

// fiatCurrencies is a hardcoded map of fiat currency codes to their display names.
// These are filtered against the CoinGecko API response to determine which
// currencies the app supports.
var fiatCurrencies = map[string]string{
	"aed": "UAE Dirham",
	"ars": "Argentine Peso",
	"aud": "Australian Dollar",
	"brl": "Brazilian Real",
	"cad": "Canadian Dollar",
	"chf": "Swiss Franc",
	"clp": "Chilean Peso",
	"cny": "Chinese Yuan",
	"czk": "Czech Koruna",
	"dkk": "Danish Krone",
	"eur": "Euro",
	"gbp": "British Pound",
	"hkd": "Hong Kong Dollar",
	"idr": "Indonesian Rupiah",
	"ils": "Israeli Shekel",
	"inr": "Indian Rupee",
	"jpy": "Japanese Yen",
	"krw": "South Korean Won",
	"mxn": "Mexican Peso",
	"myr": "Malaysian Ringgit",
	"nok": "Norwegian Krone",
	"nzd": "New Zealand Dollar",
	"php": "Philippine Peso",
	"pln": "Polish Zloty",
	"rub": "Russian Ruble",
	"sar": "Saudi Riyal",
	"sek": "Swedish Krona",
	"sgd": "Singapore Dollar",
	"thb": "Thai Baht",
	"try": "Turkish Lira",
	"twd": "Taiwan Dollar",
	"usd": "US Dollar",
	"vnd": "Vietnamese Dong",
	"zar": "South African Rand",
}

// FilterFiat returns the intersection of apiCodes with fiatCurrencies.
// Only codes present in both the API response and our fiat map are returned.
// Results are sorted alphabetically by currency code.
func FilterFiat(apiCodes []string) []string {
	result := make([]string, 0, len(apiCodes))
	for _, code := range apiCodes {
		if _, ok := fiatCurrencies[code]; ok {
			result = append(result, code)
		}
	}
	sort.Strings(result)
	return result
}

// FiatCurrencyName returns the display name for a fiat currency code.
// The second return value indicates whether the code is a known fiat currency.
func FiatCurrencyName(code string) (string, bool) {
	name, ok := fiatCurrencies[code]
	return name, ok
}
