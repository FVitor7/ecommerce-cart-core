package cart

import (
	"errors"
	"testing"
)

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		lit   string
		scale int
		want  int64
		err   error
	}{
		{"49.90", 2, 4990, nil},
		{"49.9", 2, 4990, nil},
		{"49", 2, 4900, nil},
		{"0.1", 2, 10, nil},
		{"0.10", 2, 10, nil},
		{"0", 2, 0, nil},
		{"-0", 2, 0, nil},
		{"0.00", 2, 0, nil},
		{"-12.34", 2, -1234, nil},
		{"49.900", 2, 4990, nil}, // trailing zeros are still whole cents
		{"4.99e1", 2, 4990, nil},
		{"4990E-2", 2, 4990, nil},
		{"1e+2", 2, 10000, nil},
		{"0e999999999", 2, 0, nil},
		{"90071992547409.91", 2, MaxSafeInteger, nil},
		{"3", 0, 3, nil},
		{"3.0", 0, 3, nil},
		{"9007199254740991", 0, MaxSafeInteger, nil},

		{"49.999", 2, 0, errInexact},
		{"0.30000000000000004", 2, 0, errInexact},
		{"0.001", 2, 0, errInexact},
		{"1e-3", 2, 0, errInexact},
		{"1e-999999", 2, 0, errInexact},
		{"2.5", 0, 0, errInexact},

		{"90071992547409.92", 2, 0, errRange},
		{"9007199254740992", 0, 0, errRange},
		{"1e16", 0, 0, errRange},
		{"1e999999999", 2, 0, errRange},

		{"", 2, 0, errSyntax},
		{"-", 2, 0, errSyntax},
		{"abc", 2, 0, errSyntax},
		{"49,90", 2, 0, errSyntax},
		{"R$ 49.90", 2, 0, errSyntax},
		{" 49.90", 2, 0, errSyntax},
		{"049.90", 2, 0, errSyntax},
		{"+49.90", 2, 0, errSyntax},
		{"49.", 2, 0, errSyntax},
		{".5", 2, 0, errSyntax},
		{"1e", 2, 0, errSyntax},
		{"NaN", 2, 0, errSyntax},
		{"Infinity", 2, 0, errSyntax},
	}
	for _, tt := range tests {
		got, err := parseDecimal(tt.lit, tt.scale)
		if !errors.Is(err, tt.err) || got != tt.want {
			t.Errorf("parseDecimal(%q, %d) = %d, %v; want %d, %v", tt.lit, tt.scale, got, err, tt.want, tt.err)
		}
	}
}

func TestFormatBRL(t *testing.T) {
	tests := map[Cents]string{
		0:          "R$ 0.00",
		5:          "R$ 0.05",
		30:         "R$ 0.30",
		4990:       "R$ 49.90",
		100000:     "R$ 1,000.00",
		123456789:  "R$ 1,234,567.89",
		-1590:      "-R$ 15.90",
		1000000000: "R$ 10,000,000.00",
	}
	for cents, want := range tests {
		if got := formatBRL(cents); got != want {
			t.Errorf("formatBRL(%d) = %q; want %q", cents, got, want)
		}
	}
}

func TestSafeArithmetic(t *testing.T) {
	if _, ok := addSafe(MaxSafeInteger, 1); ok {
		t.Error("addSafe(MaxSafeInteger, 1) did not report overflow")
	}
	if got, ok := addSafe(MaxSafeInteger-1, 1); !ok || got != MaxSafeInteger {
		t.Errorf("addSafe(MaxSafeInteger-1, 1) = %d, %v", got, ok)
	}
	if _, ok := mulSafe(MaxSafeInteger/2+1, 2); ok {
		t.Error("mulSafe(MaxSafeInteger/2+1, 2) did not report overflow")
	}
	if got, ok := mulSafe(1990, 3); !ok || got != 5970 {
		t.Errorf("mulSafe(1990, 3) = %d, %v", got, ok)
	}
	if got, ok := mulSafe(MaxSafeInteger, 0); !ok || got != 0 {
		t.Errorf("mulSafe(MaxSafeInteger, 0) = %d, %v", got, ok)
	}
}
