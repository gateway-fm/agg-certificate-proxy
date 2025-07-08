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
			input: "0x1234567890123456789012345678901234567890:100",
			expected: map[string]TokenValue{
				"0x1234567890123456789012345678901234567890": {Address: "0x1234567890123456789012345678901234567890", DollarValue: 100},
			},
		},
		"double entrry": {
			input: "0x1234567890123456789012345678901234567890:100,0x1234567890123456789012345678901234567891:200",
			expected: map[string]TokenValue{
				"0x1234567890123456789012345678901234567890": {Address: "0x1234567890123456789012345678901234567890", DollarValue: 100},
				"0x1234567890123456789012345678901234567891": {Address: "0x1234567890123456789012345678901234567891", DollarValue: 200},
			},
		},
		"invalid address": {
			input:       "0xa:100",
			expected:    map[string]TokenValue{},
			shouldError: true,
		},
		"invalid value": {
			input:       "0x1234567890123456789012345678901234567890:a",
			expected:    map[string]TokenValue{},
			shouldError: true,
		},
		"good with bad still errors": {
			input: "0x1234567890123456789012345678901234567890:100,0xa:200",
			expected: map[string]TokenValue{
				"0x1234567890123456789012345678901234567890": {Address: "0x1234567890123456789012345678901234567890", DollarValue: 100},
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
