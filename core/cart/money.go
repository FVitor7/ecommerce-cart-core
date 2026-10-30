package cart

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Cents is an amount of money in centavos, the smallest unit of the real (BRL).
// Every calculation is done on integers, so R$ 0,10 × 3 is exactly 30 cents
// instead of the float64 result 0.30000000000000004.
type Cents = int64

// MaxSafeInteger is the largest integer a float64 represents exactly (2^53 − 1).
// Every number the engine outputs stays at or below it, so JavaScript, which
// parses JSON numbers as float64, reads exactly the value the engine wrote.
const MaxSafeInteger = 1<<53 - 1

var (
	errSyntax  = errors.New("not a decimal number")
	errInexact = errors.New("more precision than allowed")
	errRange   = errors.New("out of range")
)

// parseDecimal converts a JSON number literal such as "49.90", "49.9" or
// "4.99e1" into an exact integer scaled by 10^scale, without going through
// float64: parseDecimal("49.90", 2) is 4990.
//
// It fails with errInexact when the value needs more decimal places than scale
// ("49.999" with scale 2, "2.5" with scale 0) and with errRange when the
// magnitude exceeds MaxSafeInteger. Trailing zeros are fine: "49.900" is 4990.
func parseDecimal(lit string, scale int) (int64, error) {
	s := lit
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	// JSON grammar: int = "0" | [1-9][0-9]* ; frac = "." [0-9]+ ; exp = [eE] [+-]? [0-9]+
	intEnd := scanDigits(s, 0)
	if intEnd == 0 || (s[0] == '0' && intEnd > 1) {
		return 0, errSyntax
	}
	i := intEnd
	frac := ""
	if i < len(s) && s[i] == '.' {
		end := scanDigits(s, i+1)
		if end == i+1 {
			return 0, errSyntax
		}
		frac = s[i+1 : end]
		i = end
	}
	exp := 0
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		expNeg := false
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			expNeg = s[i] == '-'
			i++
		}
		end := scanDigits(s, i)
		if end == i {
			return 0, errSyntax
		}
		for _, c := range s[i:end] {
			if exp < 1_000_000 { // larger exponents are out of range or zero anyway
				exp = exp*10 + int(c-'0')
			}
		}
		if expNeg {
			exp = -exp
		}
		i = end
	}
	if i != len(s) {
		return 0, errSyntax
	}

	// value = digits × 10^(exp − len(frac)), so the scaled value is
	// digits × 10^shift with shift = exp − len(frac) + scale.
	digits := strings.TrimLeft(s[:intEnd]+frac, "0")
	if digits == "" {
		return 0, nil
	}
	shift := exp - len(frac) + scale
	if shift < 0 {
		drop := -shift
		if drop >= len(digits) || strings.Trim(digits[len(digits)-drop:], "0") != "" {
			return 0, errInexact
		}
		digits = digits[:len(digits)-drop]
		shift = 0
	}
	if len(digits)+shift > 16 { // MaxSafeInteger has 16 digits
		return 0, errRange
	}
	var v int64
	for _, c := range digits {
		v = v*10 + int64(c-'0')
	}
	for ; shift > 0; shift-- {
		v *= 10
	}
	if v > MaxSafeInteger {
		return 0, errRange
	}
	if neg {
		v = -v
	}
	return v, nil
}

func scanDigits(s string, i int) int {
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// addSafe and mulSafe operate on non-negative values and report false when the
// result would exceed MaxSafeInteger.
func addSafe(a, b int64) (int64, bool) {
	if b > MaxSafeInteger-a {
		return 0, false
	}
	return a + b, true
}

func mulSafe(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	if a > MaxSafeInteger/b {
		return 0, false
	}
	return a * b, true
}

// formatBRL formats cents as reais: 123456 becomes "R$ 1,234.56".
func formatBRL(c Cents) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	reais := strconv.FormatInt(c/100, 10)
	for i := len(reais) - 3; i > 0; i -= 3 {
		reais = reais[:i] + "," + reais[i:]
	}
	return fmt.Sprintf("%sR$ %s.%02d", sign, reais, c%100)
}
