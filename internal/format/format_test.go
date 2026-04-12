package format

import "testing"

func TestFmtPriceAboveOne(t *testing.T) {
	got := FmtPrice(67234.56)
	want := "$67,234.56"
	if got != want {
		t.Errorf("FmtPrice(67234.56) = %q, want %q", got, want)
	}
}

func TestFmtPriceMillions(t *testing.T) {
	got := FmtPrice(1234567.89)
	want := "$1,234,567.89"
	if got != want {
		t.Errorf("FmtPrice(1234567.89) = %q, want %q", got, want)
	}
}

func TestFmtPriceExactlyOne(t *testing.T) {
	got := FmtPrice(1.0)
	want := "$1.00"
	if got != want {
		t.Errorf("FmtPrice(1.0) = %q, want %q", got, want)
	}
}

func TestFmtPriceBelowOne(t *testing.T) {
	got := FmtPrice(0.00012345)
	want := "$0.000123"
	if got != want {
		t.Errorf("FmtPrice(0.00012345) = %q, want %q", got, want)
	}
}

func TestFmtPriceSmallAboveOne(t *testing.T) {
	got := FmtPrice(1.50)
	want := "$1.50"
	if got != want {
		t.Errorf("FmtPrice(1.50) = %q, want %q", got, want)
	}
}

func TestFmtPriceDecimalOverflow(t *testing.T) {
	got := FmtPrice(99.995)
	want := "$100.00"
	if got != want {
		t.Errorf("FmtPrice(99.995) = %q, want %q", got, want)
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
	got := FmtMoney(0)
	want := "$0.00"
	if got != want {
		t.Errorf("FmtMoney(0) = %q, want %q", got, want)
	}
}

func TestFmtMoneySmall(t *testing.T) {
	got := FmtMoney(0.5)
	want := "$0.50"
	if got != want {
		t.Errorf("FmtMoney(0.5) = %q, want %q", got, want)
	}
}

func TestFmtMoneyThousands(t *testing.T) {
	got := FmtMoney(12345.678)
	want := "$12,345.68"
	if got != want {
		t.Errorf("FmtMoney(12345.678) = %q, want %q", got, want)
	}
}
