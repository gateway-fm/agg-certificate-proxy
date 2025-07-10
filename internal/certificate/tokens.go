package certificate

import (
	"fmt"
	"strconv"
	"strings"
)

type TokenValue struct {
	OriginNetwork uint32
	Address       string
	DollarValue   uint64
	Multiplier    uint64
}

// Hash returns a keccack hash of the token value based on origin network combined with the address
func (t TokenValue) ID() string {
	return fmt.Sprintf("%d:%s", t.OriginNetwork, t.Address)
}

func ParseTokenValues(tokenValues string) (map[string]TokenValue, error) {
	result := make(map[string]TokenValue)
	split := strings.Split(tokenValues, ",")
	for _, tokenValue := range split {
		parts := strings.Split(tokenValue, ":")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid token value: %s", tokenValue)
		}
		originNetwork, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid origin network: %s", parts[0])
		}
		address := parts[1]
		if len(address) != 40 {
			return nil, fmt.Errorf("invalid address: %s", address)
		}
		address = strings.ToLower(address)
		value, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", parts[1])
		}
		multiplier, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid multiplier: %s", parts[2])
		}
		tv := TokenValue{
			OriginNetwork: uint32(originNetwork),
			Address:       address,
			DollarValue:   value,
			Multiplier:    multiplier,
		}
		result[tv.ID()] = tv
	}
	return result, nil
}
