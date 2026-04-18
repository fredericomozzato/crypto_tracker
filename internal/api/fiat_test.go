package api

import (
	"sort"
	"testing"
)

func TestFilterFiatMatchesKnownCodes(t *testing.T) {
	apiCodes := []string{"usd", "eur", "btc", "gbp", "eth"}
	result := FilterFiat(apiCodes)

	// Should contain only fiat codes: usd, eur, gbp
	if len(result) != 3 {
		t.Fatalf("expected 3 fiat codes, got %d: %v", len(result), result)
	}

	// Result should be sorted alphabetically
	expected := []string{"eur", "gbp", "usd"}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i])
		}
	}
}

func TestFilterFiatEmptyInput(t *testing.T) {
	result := FilterFiat([]string{})

	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestFilterFiatNoMatches(t *testing.T) {
	// All crypto codes
	result := FilterFiat([]string{"btc", "eth", "xrp", "ada"})

	if len(result) != 0 {
		t.Errorf("expected empty result for all-crypto input, got %v", result)
	}
}

func TestFilterFiatAllFiat(t *testing.T) {
	// Build list of all fiat codes from the map
	allFiat := make([]string, 0, len(FiatCurrencies))
	for code := range FiatCurrencies {
		allFiat = append(allFiat, code)
	}

	result := FilterFiat(allFiat)

	if len(result) != len(FiatCurrencies) {
		t.Errorf("expected %d fiat codes, got %d", len(FiatCurrencies), len(result))
	}

	// Sort both for comparison
	sort.Strings(result)
	sort.Strings(allFiat)

	for i, exp := range allFiat {
		if result[i] != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, result[i])
		}
	}
}
