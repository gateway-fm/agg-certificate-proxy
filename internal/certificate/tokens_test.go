package certificate

import (
	"reflect"
	"testing"
)

func Test_TokenValues(t *testing.T) {
	var tests = map[string]struct {
		input       string
		expected    map[string]TokenValue
		shouldError bool
	}{
		"single entry": {
			input: "1234567890123456789012345678901234567890:100:1000000000000000000",
			expected: map[string]TokenValue{
				"1234567890123456789012345678901234567890": {Address: "1234567890123456789012345678901234567890", DollarValue: 100, Multiplier: 1000000000000000000},
			},
		},
		"double entrry": {
			input: "1234567890123456789012345678901234567890:100:1000000000000000000,1234567890123456789012345678901234567891:200:1000000000000000000",
			expected: map[string]TokenValue{
				"1234567890123456789012345678901234567890": {Address: "1234567890123456789012345678901234567890", DollarValue: 100, Multiplier: 1000000000000000000},
				"1234567890123456789012345678901234567891": {Address: "1234567890123456789012345678901234567891", DollarValue: 200, Multiplier: 1000000000000000000},
			},
		},
		"invalid address": {
			input:       "a:100:1000000000000000000",
			expected:    map[string]TokenValue{},
			shouldError: true,
		},
		"invalid value": {
			input:       "1234567890123456789012345678901234567890:a:1000000000000000000",
			expected:    map[string]TokenValue{},
			shouldError: true,
		},
		"good with bad still errors": {
			input: "1234567890123456789012345678901234567890:100:1000000000000000000,a:200:1000000000000000000",
			expected: map[string]TokenValue{
				"1234567890123456789012345678901234567890": {Address: "1234567890123456789012345678901234567890", DollarValue: 100, Multiplier: 1000000000000000000},
			},
			shouldError: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parsed, err := ParseTokenValues(test.input)
			if test.shouldError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(parsed, test.expected) {
					t.Errorf("expected %v, got %v", test.expected, parsed)
				}
			}
		})
	}
}
