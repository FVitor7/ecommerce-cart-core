package cart

import "testing"

func TestNormalizeGTIN(t *testing.T) {
	valid := map[string]string{
		"96385074":       "00000096385074", // EAN-8
		"036000291452":   "00036000291452", // UPC-A
		"0036000291452":  "00036000291452", // the same product as EAN-13
		"4006381333931":  "04006381333931", // EAN-13
		"7891000100103":  "07891000100103", // EAN-13, GS1 Brazil prefix
		"10012345678902": "10012345678902", // GTIN-14
	}
	for code, want := range valid {
		got, err := normalizeGTIN(code)
		if err != nil || got != want {
			t.Errorf("normalizeGTIN(%q) = %q, %v; want %q", code, got, err, want)
		}
	}

	invalid := map[string]string{
		"4006381333932":   "EAN/GTIN has an invalid check digit",
		"96385075":        "EAN/GTIN has an invalid check digit",
		"400638133393":    "EAN/GTIN has an invalid check digit",
		"40063813339":     "EAN/GTIN must have 8, 12, 13 or 14 digits",
		"123456789012345": "EAN/GTIN must have 8, 12, 13 or 14 digits",
		"400638133393X":   "EAN/GTIN must contain only digits",
		" 4006381333931":  "EAN/GTIN must contain only digits",
		"4006-381333931":  "EAN/GTIN must contain only digits",
	}
	for code, want := range invalid {
		if _, err := normalizeGTIN(code); err == nil || err.Error() != want {
			t.Errorf("normalizeGTIN(%q) error = %v; want %q", code, err, want)
		}
	}
}
