package certificate

import (
	"fmt"
	"strconv"
	"strings"
)

type TokenValue struct {
	Address     string
	DollarValue uint64
}

func ParseTokenValues(tokenValues string) (map[string]TokenValue, error) {
	result := make(map[string]TokenValue)
	split := strings.Split(tokenValues, ",")
	for _, tokenValue := range split {
		parts := strings.Split(tokenValue, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid token value: %s", tokenValue)
		}
		address := parts[0]
		if len(address) != 40 {
			return nil, fmt.Errorf("invalid address: %s", address)
		}
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", parts[1])
		}
		tv := TokenValue{
			Address:     address,
			DollarValue: value,
		}
		result[address] = tv
	}
	return result, nil
}
