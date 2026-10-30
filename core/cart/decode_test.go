package cart

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	c, err := Decode([]byte(`{
		"cartItems": [
			{"sku": "A", "ean": "7891000100103", "name": "Coffee", "price": 18.90, "quantity": 2, "stock": 40, "imageUrl": "ignored"},
			{"sku": "B", "price": "0.10", "quantity": 3.0, "stock": 100, "ean": null},
			{"sku": "C", "price": 4.99e1, "quantity": 1, "stock": 0}
		],
		"offers": [{"ean": "7891000100103", "price": "18.9", "discountedPrice": 15.90}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Cart{
		Items: []Item{
			{SKU: "A", EAN: "7891000100103", Name: "Coffee", Price: 1890, Quantity: 2, Stock: 40},
			{SKU: "B", Price: 10, Quantity: 3, Stock: 100},
			{SKU: "C", Price: 4990, Quantity: 1, Stock: 0},
		},
		Offers: []Offer{{EAN: "7891000100103", Price: 1890, DiscountedPrice: 1590}},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("Decode = %+v\nwant %+v", c, want)
	}
}

func TestDecodeOffersAreOptional(t *testing.T) {
	for _, input := range []string{`{"cartItems": []}`, `{"cartItems": [], "offers": null}`} {
		c, err := Decode([]byte(input))
		if err != nil || len(c.Items) != 0 || len(c.Offers) != 0 {
			t.Errorf("Decode(%s) = %+v, %v", input, c, err)
		}
	}
}

func TestExceedsDepth(t *testing.T) {
	nest := func(n int) string { return strings.Repeat("[", n) + strings.Repeat("]", n) }
	tests := []struct {
		input string
		want  bool
	}{
		{`{"cartItems": []}`, false},
		{nest(3), false},
		{nest(4), true},
		{`{"a": {"b": {"c": 1}}}`, false},
		{`{"a": {"b": {"c": {}}}}`, true},
		{`{"a": "[[[[[[[["}`, false},          // brackets inside strings do not count
		{`{"a": "\"[[[[[[", "b": []}`, false}, // nor after an escaped quote
		{`{"a": "\\", "b": [[[[]]]]}`, true},  // an escaped backslash closes nothing
	}
	for _, tt := range tests {
		if got := exceedsDepth([]byte(tt.input), 3); got != tt.want {
			t.Errorf("exceedsDepth(%s, 3) = %v; want %v", tt.input, got, tt.want)
		}
	}

	deep := `{"cartItems": [], "meta": ` + nest(maxDepth) + `}`
	if _, err := Decode([]byte(deep)); err == nil || err.Error() != "the JSON nests more than 64 levels deep" {
		t.Errorf("Decode(%d levels) error = %v", maxDepth+1, err)
	}
	if _, err := Decode([]byte(`{"cartItems": [], "meta": ` + nest(maxDepth-1) + `}`)); err != nil {
		t.Errorf("Decode(%d levels) error = %v", maxDepth, err)
	}
}

func TestDecodeErrors(t *testing.T) {
	item := `"sku": "A", "price": 1.00, "quantity": 1, "stock": 1`
	tests := []struct {
		input, code, message string
	}{
		{``, ErrInvalidJSON, "invalid JSON: empty input"},
		{`{"cartItems": [}`, ErrInvalidJSON, "invalid JSON: invalid character '}' looking for beginning of value"},
		{`{"cartItems": []} {}`, ErrInvalidJSON, "invalid JSON: unexpected content after the top-level value"},
		{`[]`, ErrInvalidInput, "the cart must be a JSON object"},
		{`{}`, ErrInvalidInput, "cartItems: is required"},
		{`{"cartItems": null}`, ErrInvalidInput, "cartItems: must be an array"},
		{`{"cartItems": {}}`, ErrInvalidInput, "cartItems: must be an array"},
		{`{"cartItems": [1]}`, ErrInvalidInput, "cartItems[0]: must be an object"},
		{`{"cartItems": [{"price": 1.00, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].sku: is required"},
		{`{"cartItems": [{"sku": 10, "price": 1.00, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].sku: must be a string"},
		{`{"cartItems": [{"sku": "A", "ean": 7891000100103, "price": 1.00, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].ean: must be a string"},
		{`{"cartItems": [{"sku": "A", "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].price: is required"},
		{`{"cartItems": [{"sku": "A", "price": true, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, `cartItems[0].price: must be a number or a decimal string, such as 49.90 or "49.90"`},
		{`{"cartItems": [{"sku": "A", "price": "49,90", "quantity": 1, "stock": 1}]}`, ErrInvalidInput, `cartItems[0].price: invalid amount: "49,90"`},
		{`{"cartItems": [{"sku": "A", "price": 49.999, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].price: must have at most 2 decimal places (cents), got 49.999"},
		{`{"cartItems": [{"sku": "A", "price": 1e20, "quantity": 1, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].price: out of the supported range: 1e20"},
		{`{"cartItems": [{"sku": "A", "price": 1.00, "quantity": "2", "stock": 1}]}`, ErrInvalidInput, "cartItems[0].quantity: must be an integer"},
		{`{"cartItems": [{"sku": "A", "price": 1.00, "quantity": 1.5, "stock": 1}]}`, ErrInvalidInput, "cartItems[0].quantity: must be an integer, got 1.5"},
		{`{"cartItems": [{"sku": "A", "price": 1.00, "quantity": 1}]}`, ErrInvalidInput, "cartItems[0].stock: is required"},
		{`{"cartItems": [{` + item + `}], "offers": {}}`, ErrInvalidInput, "offers: must be an array"},
		{`{"cartItems": [{` + item + `}], "offers": [{"sku": "A", "price": 1.00}]}`, ErrInvalidInput, "offers[0].discountedPrice: is required"},
	}
	for _, tt := range tests {
		_, err := Decode([]byte(tt.input))
		var e *Error
		if !errors.As(err, &e) || e.Code != tt.code || e.Message != tt.message {
			t.Errorf("Decode(%s)\n error = %v\n want  %s %q", tt.input, err, tt.code, tt.message)
		}
	}
}
