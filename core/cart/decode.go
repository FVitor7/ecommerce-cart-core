package cart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Decode parses the JSON input of Calculate:
//
//	{
//	  "cartItems": [{"sku", "ean", "name", "price", "quantity", "stock"}, ...],
//	  "offers":    [{"sku", "ean", "price", "discountedPrice"}, ...]
//	}
//
// "offers" is optional. Money may be a JSON number (49.90) or a decimal string
// ("49.90"); either way it is read from its literal text, never through
// float64, and must be a whole number of cents. Unknown fields are ignored.
//
// Decode checks types and shapes only; Calculate checks the values.
func Decode(data []byte) (Cart, error) {
	if exceedsDepth(data, maxDepth) {
		return Cart{}, invalidInput("", "the JSON nests more than %d levels deep", maxDepth)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return Cart{}, &Error{Code: ErrInvalidJSON, Message: "invalid JSON: empty input"}
		}
		return Cart{}, &Error{Code: ErrInvalidJSON, Message: "invalid JSON: " + err.Error()}
	}
	if _, err := dec.Token(); err != io.EOF {
		return Cart{}, &Error{Code: ErrInvalidJSON, Message: "invalid JSON: unexpected content after the top-level value"}
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return Cart{}, invalidInput("", "the cart must be a JSON object")
	}

	var c Cart
	items, ok := obj["cartItems"]
	if !ok {
		return Cart{}, invalidInput("cartItems", "is required")
	}
	err := eachObject(items, "cartItems", func(f *fields) {
		c.Items = append(c.Items, Item{
			SKU:      f.str("sku", true),
			EAN:      f.str("ean", false),
			Name:     f.str("name", false),
			Price:    f.money("price"),
			Quantity: f.integer("quantity"),
			Stock:    f.integer("stock"),
		})
	})
	if err != nil {
		return Cart{}, err
	}
	if offers := obj["offers"]; offers != nil {
		err := eachObject(offers, "offers", func(f *fields) {
			c.Offers = append(c.Offers, Offer{
				SKU:             f.str("sku", false),
				EAN:             f.str("ean", false),
				Price:           f.money("price"),
				DiscountedPrice: f.money("discountedPrice"),
			})
		})
		if err != nil {
			return Cart{}, err
		}
	}
	return c, nil
}

// maxDepth bounds JSON nesting. A cart nests 3 or 4 levels; the limit keeps
// hostile input from exhausting the small stack of the WebAssembly module,
// since encoding/json decodes nested values recursively.
const maxDepth = 64

// exceedsDepth reports whether data nests objects and arrays deeper than max.
// It only counts brackets outside strings; the decoder validates the rest.
func exceedsDepth(data []byte, max int) bool {
	depth, inString, escaped := 0, false, false
	for _, c := range data {
		switch {
		case escaped:
			escaped = false
		case inString:
			escaped = c == '\\'
			inString = c != '"'
		case c == '"':
			inString = true
		case c == '{' || c == '[':
			if depth++; depth > max {
				return true
			}
		case c == '}' || c == ']':
			depth--
		}
	}
	return false
}

// eachObject calls fn for each object of a JSON list, stopping at the first
// field error.
func eachObject(v any, path string, fn func(*fields)) error {
	list, ok := v.([]any)
	if !ok {
		return invalidInput(path, "must be an array")
	}
	for i, elem := range list {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		obj, ok := elem.(map[string]any)
		if !ok {
			return invalidInput(elemPath, "must be an object")
		}
		f := &fields{obj: obj, path: elemPath}
		fn(f)
		if f.err != nil {
			return f.err
		}
	}
	return nil
}

// fields reads typed values from a JSON object. After the first error, reads
// return zero values and the error is kept in err.
type fields struct {
	obj  map[string]any
	path string
	err  *Error
}

func (f *fields) fail(key, format string, args ...any) {
	if f.err == nil {
		f.err = invalidInput(f.path+"."+key, format, args...)
	}
}

// get returns the value of key; absent keys and null are both reported as nil.
func (f *fields) get(key string, required bool) any {
	if f.err != nil {
		return nil
	}
	v := f.obj[key]
	if v == nil && required {
		f.fail(key, "is required")
	}
	return v
}

func (f *fields) str(key string, required bool) string {
	switch v := f.get(key, required).(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		f.fail(key, "must be a string")
		return ""
	}
}

func (f *fields) money(key string) Cents {
	var lit string
	switch v := f.get(key, true).(type) {
	case nil:
		return 0
	case json.Number:
		lit = string(v)
	case string:
		lit = v
	default:
		f.fail(key, `must be a number or a decimal string, such as 49.90 or "49.90"`)
		return 0
	}
	cents, err := parseDecimal(lit, 2)
	switch {
	case errors.Is(err, errSyntax):
		f.fail(key, "invalid amount: %q", lit)
	case errors.Is(err, errInexact):
		f.fail(key, "must have at most 2 decimal places (cents), got %s", lit)
	case errors.Is(err, errRange):
		f.fail(key, "out of the supported range: %s", lit)
	}
	return cents
}

func (f *fields) integer(key string) int64 {
	v, ok := f.get(key, true).(json.Number)
	if !ok {
		f.fail(key, "must be an integer") // no-op if get already failed
		return 0
	}
	n, err := parseDecimal(string(v), 0)
	switch {
	case errors.Is(err, errRange):
		f.fail(key, "out of the supported range: %s", v)
	case err != nil:
		f.fail(key, "must be an integer, got %s", v)
	}
	return n
}
