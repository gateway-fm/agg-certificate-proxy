package certificate

import (
	"fmt"
	"strconv"
	"strings"
)

type TokenValue struct {
	Address     string
	DollarValue uint64
	Multiplier  uint64
}

func ParseTokenValues(tokenValues string) (map[string]TokenValue, error) {
	result := make(map[string]TokenValue)
	split := strings.Split(tokenValues, ",")
	for _, tokenValue := range split {
		parts := strings.Split(tokenValue, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid token value: %s", tokenValue)
		}
		address := parts[0]
		if len(address) != 40 {
			return nil, fmt.Errorf("invalid address: %s", address)
		}
		address = strings.ToLower(address)
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", parts[1])
		}
		multiplier, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid multiplier: %s", parts[2])
		}
		tv := TokenValue{
			Address:     address,
			DollarValue: value,
			Multiplier:  multiplier,
		}
		result[address] = tv
	}
	return result, nil
}
