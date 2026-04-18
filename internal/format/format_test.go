package format

import "testing"

func TestFmtPriceAboveOne(t *testing.T) {
	got := FmtPrice(67234.56, "usd")
	want := "USD 67,234.56"
	if got != want {
		t.Errorf("FmtPrice(67234.56, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceMillions(t *testing.T) {
	got := FmtPrice(1234567.89, "usd")
	want := "USD 1,234,567.89"
	if got != want {
		t.Errorf("FmtPrice(1234567.89, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceExactlyOne(t *testing.T) {
	got := FmtPrice(1.0, "usd")
	want := "USD 1.00"
	if got != want {
		t.Errorf("FmtPrice(1.0, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceBelowOne(t *testing.T) {
	got := FmtPrice(0.00012345, "usd")
	want := "USD 0.000123"
	if got != want {
		t.Errorf("FmtPrice(0.00012345, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceSmallAboveOne(t *testing.T) {
	got := FmtPrice(1.50, "usd")
	want := "USD 1.50"
	if got != want {
		t.Errorf("FmtPrice(1.50, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceDecimalOverflow(t *testing.T) {
	got := FmtPrice(99.995, "usd")
	want := "USD 100.00"
	if got != want {
		t.Errorf("FmtPrice(99.995, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtChangePositive(t *testing.T) {
	got := FmtChange(2.34)
	want := "+2.34%"
	if got != want {
		t.Errorf("FmtChange(2.34) = %q, want %q", got, want)
	}
}

func TestFmtChangeNegative(t *testing.T) {
	got := FmtChange(-1.23)
	want := "-1.23%"
	if got != want {
		t.Errorf("FmtChange(-1.23) = %q, want %q", got, want)
	}
}

func TestFmtChangeZero(t *testing.T) {
	got := FmtChange(0.0)
	want := "+0.00%"
	if got != want {
		t.Errorf("FmtChange(0.0) = %q, want %q", got, want)
	}
}

func TestFmtMoneyZero(t *testing.T) {
	got := FmtMoney(0, "usd")
	want := "USD 0.00"
	if got != want {
		t.Errorf("FmtMoney(0, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtMoneySmall(t *testing.T) {
	got := FmtMoney(0.5, "usd")
	want := "USD 0.50"
	if got != want {
		t.Errorf("FmtMoney(0.5, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtMoneyThousands(t *testing.T) {
	got := FmtMoney(12345.678, "usd")
	want := "USD 12,345.68"
	if got != want {
		t.Errorf("FmtMoney(12345.678, \"usd\") = %q, want %q", got, want)
	}
}

func TestFmtPriceWithEur(t *testing.T) {
	got := FmtPrice(1234.56, "eur")
	want := "EUR 1,234.56"
	if got != want {
		t.Errorf("FmtPrice(1234.56, \"eur\") = %q, want %q", got, want)
	}
}

func TestFmtPriceBelowOneWithEur(t *testing.T) {
	got := FmtPrice(0.000123, "eur")
	want := "EUR 0.000123"
	if got != want {
		t.Errorf("FmtPrice(0.000123, \"eur\") = %q, want %q", got, want)
	}
}

func TestFmtPriceWithUnknownCurrency(t *testing.T) {
	got := FmtPrice(1234.56, "brl")
	want := "BRL 1,234.56"
	if got != want {
		t.Errorf("FmtPrice(1234.56, \"brl\") = %q, want %q", got, want)
	}
}

func TestFmtMoneyWithEur(t *testing.T) {
	got := FmtMoney(12345.67, "eur")
	want := "EUR 12,345.67"
	if got != want {
		t.Errorf("FmtMoney(12345.67, \"eur\") = %q, want %q", got, want)
	}
}

func TestFmtMoneyWithJpy(t *testing.T) {
	got := FmtMoney(12345.67, "jpy")
	want := "JPY 12,345.67"
	if got != want {
		t.Errorf("FmtMoney(12345.67, \"jpy\") = %q, want %q", got, want)
	}
}

func TestCurrencyCodeLowerCase(t *testing.T) {
	got := currencyCode("usd")
	want := "USD"
	if got != want {
		t.Errorf("currencyCode(\"usd\") = %q, want %q", got, want)
	}
}

func TestCurrencyCodeUpperCase(t *testing.T) {
	got := currencyCode("EUR")
	want := "EUR"
	if got != want {
		t.Errorf("currencyCode(\"EUR\") = %q, want %q", got, want)
	}
}
