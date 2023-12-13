package common

import (
	"encoding/hex"
	"strings"
)

// DecodeHex decodes hex encoded string to raw byte slice
func DecodeHex(input string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(input, "0x"))
}
