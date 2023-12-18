package types

import (
	"encoding/hex"
	"strings"
)

const (
	HexBase      = 16
	HexBitSize64 = 64
)

func encodeToHex(b []byte) []byte {
	str := hex.EncodeToString(b)
	if len(str)%2 != 0 {
		str = "0" + str
	}
	return []byte("0x" + str)
}

func decodeToHex(b []byte) ([]byte, error) {
	str := string(b)
	str = strings.TrimPrefix(str, "0x")
	if len(str)%2 != 0 {
		str = "0" + str
	}
	return hex.DecodeString(str)
}

// hexIsValid checks if the provided string is a valid hexadecimal value
func hexIsValid(s string) bool {
	str := strings.TrimPrefix(s, "0x")
	for _, b := range []byte(str) {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F') {
			return false
		}
	}
	return true
}
