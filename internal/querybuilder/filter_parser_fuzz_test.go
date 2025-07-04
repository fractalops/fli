package querybuilder

import (
	"testing"
)

func FuzzParsePortFilter(f *testing.F) {
	// Add seed corpus
	f.Add("port=80")
	f.Add("port=80..443")
	f.Add("port=-22")
	f.Add("port=80,443..8080,-22")
	f.Add("port=80..100,90..110")

	f.Fuzz(func(t *testing.T, input string) {
		expr, err := ParseFilter(input)
		if err != nil {
			// Invalid input is expected, just return
			return
		}

		// If we got a valid expression, verify it's not nil
		if expr == nil {
			t.Errorf("got nil expression for valid filter input: %s", input)
		}
	})
}

func FuzzParseIPFilter(f *testing.F) {
	// Add seed corpus
	f.Add("ip=10.0.0.1")
	f.Add("ip=10.0")
	f.Add("ip=-10.0.0.1")
	f.Add("ip=10.0.0.1,10.0,-192.168.1.1")
	f.Add("ip=10.0,10.0.0")

	f.Fuzz(func(t *testing.T, input string) {
		expr, err := ParseFilter(input)
		if err != nil {
			// Invalid input is expected, just return
			return
		}

		// If we got a valid expression, verify it's not nil
		if expr == nil {
			t.Errorf("got nil expression for valid filter input: %s", input)
		}
	})
}
