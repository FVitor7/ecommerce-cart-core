package cart

import (
	"bytes"
	"encoding/json"
	"errors"
)

// CalculateJSON is the wire API that calc.wasm exposes to every host: it takes
// the JSON described in Decode and returns {"result": Result} on success or
// {"error": Error} when the input cannot be calculated.
func CalculateJSON(input []byte) []byte {
	res, err := calculateJSON(input)
	if err != nil {
		var e *Error
		if !errors.As(err, &e) {
			e = &Error{Code: "INTERNAL_ERROR", Message: err.Error()}
		}
		return marshal(struct {
			Error *Error `json:"error"`
		}{e})
	}
	return marshal(struct {
		Result Result `json:"result"`
	}{res})
}

func calculateJSON(input []byte) (Result, error) {
	c, err := Decode(input)
	if err != nil {
		return Result{}, err
	}
	return Calculate(c)
}

func marshal(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep "&", "<" and ">" in product names readable
	if err := enc.Encode(v); err != nil {
		// Only plain structs, strings and integers are encoded; this cannot fail.
		return []byte(`{"error":{"code":"INTERNAL_ERROR","message":"failed to encode the response"}}`)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
