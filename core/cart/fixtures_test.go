package cart

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The fixtures are shared with the Python and JavaScript test suites, which
// run them through calc.wasm: the same input must give the same output in
// every language. After an intentional change to the rules, regenerate the
// expected outputs and review the diff:
//
//	go test ./cart -run 'Fixture|Scenarios' -update
var update = flag.Bool("update", false, "rewrite the expected outputs of the fixtures")

const fixturesDir = "../../fixtures"

// response is the envelope returned by CalculateJSON.
type response struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// scenario is a file in fixtures/scenarios. The input is either a JSON value
// (input) or raw text (rawInput, for malformed JSON). The expected output is
// either a result (expected) or an error (expectedError).
type scenario struct {
	Description   string          `json:"description"`
	Input         json.RawMessage `json:"input,omitempty"`
	RawInput      *string         `json:"rawInput,omitempty"`
	Expected      json.RawMessage `json:"expected,omitempty"`
	ExpectedError json.RawMessage `json:"expectedError,omitempty"`
}

func TestCheckoutFixture(t *testing.T) {
	var cart, offers map[string]json.RawMessage
	readJSON(t, filepath.Join(fixturesDir, "cart.json"), &cart)
	readJSON(t, filepath.Join(fixturesDir, "offers.json"), &offers)
	cart["offers"] = offers["offers"]
	input, err := json.Marshal(cart)
	if err != nil {
		t.Fatal(err)
	}

	var got response
	if err := json.Unmarshal(CalculateJSON(input), &got); err != nil || got.Error != nil {
		t.Fatalf("CalculateJSON failed: %s %v", got.Error, err)
	}
	path := filepath.Join(fixturesDir, "expected.json")
	if *update {
		writeFile(t, path, append(indent(t, got.Result, ""), '\n'))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertSameJSON(t, got.Result, want)
}

func TestScenarios(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(fixturesDir, "scenarios", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no scenarios found: %v", err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			var sc scenario
			readJSON(t, file, &sc)
			input := []byte(sc.Input)
			if sc.RawInput != nil {
				input = []byte(*sc.RawInput)
			}

			var got response
			if err := json.Unmarshal(CalculateJSON(input), &got); err != nil {
				t.Fatal(err)
			}
			if *update {
				sc.Expected, sc.ExpectedError = got.Result, got.Error
				writeScenario(t, file, sc)
				return
			}
			switch {
			case sc.Expected != nil && got.Result != nil:
				assertSameJSON(t, got.Result, sc.Expected)
			case sc.ExpectedError != nil && got.Error != nil:
				assertSameJSON(t, got.Error, sc.ExpectedError)
			default:
				t.Errorf("got result %s, error %s\nwant result %s, error %s", got.Result, got.Error, sc.Expected, sc.ExpectedError)
			}
		})
	}
}

func assertSameJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("got:\n%s\nwant:\n%s", indent(t, got, ""), indent(t, want, ""))
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// writeScenario rewrites a scenario keeping the input exactly as written by
// hand, so its formatting (one cart item per line, number literals) survives.
func writeScenario(t *testing.T, path string, sc scenario) {
	t.Helper()
	var b bytes.Buffer
	description, _ := json.Marshal(sc.Description)
	b.WriteString("{\n  \"description\": " + string(description) + ",\n")
	if sc.RawInput != nil {
		raw, _ := json.Marshal(*sc.RawInput)
		b.WriteString("  \"rawInput\": " + string(raw) + ",\n")
	} else {
		b.WriteString("  \"input\": " + string(sc.Input) + ",\n")
	}
	if sc.ExpectedError != nil {
		b.WriteString("  \"expectedError\": " + string(indent(t, sc.ExpectedError, "  ")))
	} else {
		b.WriteString("  \"expected\": " + string(indent(t, sc.Expected, "  ")))
	}
	b.WriteString("\n}\n")
	writeFile(t, path, b.Bytes())
}

func indent(t *testing.T, data []byte, prefix string) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := json.Indent(&b, data, prefix, "  "); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
