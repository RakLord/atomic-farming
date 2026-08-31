package bignum

import (
	"encoding/json"
	"testing"
)

func TestParseStringRoundTrip(t *testing.T) {
	input := MustParse("12345.678")
	text := input.String()
	parsed, err := Parse(text)
	if err != nil {
		t.Fatalf("Parse(String()): %v", err)
	}
	if !parsed.Eq(input) {
		t.Fatalf("round-trip mismatch: got %v want %v", parsed, input)
	}
}

func TestAddDifferentExponentsKeepsLargeValue(t *testing.T) {
	large := MustParse("1e30")
	small := MustParse("1e5")
	got := large.Add(small)
	if !got.Eq(large) {
		t.Fatalf("expected large value to dominate, got %v want %v", got, large)
	}
}

func TestAddSameExponent(t *testing.T) {
	got := MustParse("2.5e3").Add(MustParse("7.5e2"))
	want := MustParse("3.25e3")
	if !got.Eq(want) {
		t.Fatalf("Add mismatch: got %v want %v", got, want)
	}
}

func TestSubToZero(t *testing.T) {
	got := MustParse("9.99e12").Sub(MustParse("9.99e12"))
	if !got.IsZero() {
		t.Fatalf("expected zero, got %v", got)
	}
}

func TestCmp(t *testing.T) {
	if !MustParse("1e9").GT(MustParse("9e8")) {
		t.Fatalf("expected GT to report true")
	}
	if !MustParse("-2e3").LT(MustParse("-1e3")) {
		t.Fatalf("expected LT to report true for negatives")
	}
	if MustParse("5e4").Cmp(MustParse("5e4")) != 0 {
		t.Fatalf("expected equal values to compare as zero")
	}
}

func TestMarshalJSON(t *testing.T) {
	type payload struct {
		Value Decimal `json:"value"`
	}

	blob, err := json.Marshal(payload{Value: MustParse("2.5e6")})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(blob) != `{"value":"2.5e6"}` {
		t.Fatalf("unexpected JSON: %s", blob)
	}

	var out payload
	if err := json.Unmarshal(blob, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Value.Eq(MustParse("2.5e6")) {
		t.Fatalf("JSON round-trip mismatch: got %v", out.Value)
	}
}

func TestFormatScientific(t *testing.T) {
	if got := MustParse("12500").Format(DisplayScientific, 2); got != "12,500" {
		t.Fatalf("scientific small-value format: got %q want %q", got, "12,500")
	}
	if got := MustParse("1.234e9").Format(DisplayScientific, 2); got != "1.23e9" {
		t.Fatalf("scientific large-value format: got %q want %q", got, "1.23e9")
	}
}

func TestFormatShort(t *testing.T) {
	if got := MustParse("12500").Format(DisplayShort, 2); got != "12.5K" {
		t.Fatalf("short format: got %q want %q", got, "12.5K")
	}
	if got := MustParse("1.234e39").Format(DisplayShort, 2); got != "1.23e39" {
		t.Fatalf("short fallback format: got %q want %q", got, "1.23e39")
	}
}

func TestCeil(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "0"},
		{"5", "5"},
		{"1.3", "2"},
		{"1.0001", "2"},
		{"99.999", "100"},
		{"-0.3", "0"},
		{"-1.7", "-1"},
		{"-5", "-5"},
		{"1.23e20", "1.23e20"}, // huge exponent: already integer
	}
	for _, tc := range cases {
		got := MustParse(tc.in).Ceil()
		want := MustParse(tc.want)
		if !got.Eq(want) {
			t.Errorf("Ceil(%s) = %s, want %s", tc.in, got, want)
		}
	}
}

func TestCeilIsWholeNumber(t *testing.T) {
	// For practical cost values, Ceil() should be idempotent.
	inputs := []string{"0", "1", "1.5", "17.9", "1.23e3", "9.999e5"}
	for _, s := range inputs {
		d := MustParse(s).Ceil()
		if !d.Eq(d.Ceil()) {
			t.Errorf("Ceil(%s) not idempotent", s)
		}
	}
}
