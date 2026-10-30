package cart

import (
	"errors"
	"strings"
)

// normalizeGTIN validates an EAN-8, UPC-A (12 digits), EAN-13 or GTIN-14 code
// and returns it left-padded with zeros to 14 digits, the canonical GS1 form.
// Padding makes the same product compare equal however it was written:
// UPC-A "036000291452" and EAN-13 "0036000291452" are both "00036000291452".
func normalizeGTIN(code string) (string, error) {
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return "", errors.New("EAN/GTIN must contain only digits")
		}
	}
	switch len(code) {
	case 8, 12, 13, 14:
	default:
		return "", errors.New("EAN/GTIN must have 8, 12, 13 or 14 digits")
	}

	// GS1 check digit: weights 3, 1, 3, 1... from the rightmost data digit.
	sum := 0
	for i, weight := len(code)-2, 3; i >= 0; i, weight = i-1, 4-weight {
		sum += int(code[i]-'0') * weight
	}
	if want := (10 - sum%10) % 10; int(code[len(code)-1]-'0') != want {
		return "", errors.New("EAN/GTIN has an invalid check digit")
	}
	return strings.Repeat("0", 14-len(code)) + code, nil
}
