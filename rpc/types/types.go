package types

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ArgUint64 helps to marshal uint64 values provided in the RPC requests
type ArgUint64 uint64

// MarshalText marshals into text
func (b ArgUint64) MarshalText() ([]byte, error) {
	buf := make([]byte, 2) //nolint:gomnd
	copy(buf, `0x`)
	buf = strconv.AppendUint(buf, uint64(b), Base)
	return buf, nil
}

// UnmarshalText unmarshals from text
func (b *ArgUint64) UnmarshalText(input []byte) error {
	str := strings.TrimPrefix(string(input), "0x")
	num, err := strconv.ParseUint(str, Base, BitSize64)
	if err != nil {
		return err
	}
	*b = ArgUint64(num)
	return nil
}

// Hex returns a hexadecimal representation
func (b ArgUint64) Hex() string {
	bb, _ := b.MarshalText()
	return string(bb)
}

// ArgUint64Ptr returns the pointer of the provided ArgUint64
func ArgUint64Ptr(a ArgUint64) *ArgUint64 {
	return &a
}

// ArgBytes helps to marshal byte array values provided in the RPC requests
type ArgBytes []byte

// MarshalText marshals into text
func (b ArgBytes) MarshalText() ([]byte, error) {
	return encodeToHex(b), nil
}

// UnmarshalText unmarshals from text
func (b *ArgBytes) UnmarshalText(input []byte) error {
	hh, err := decodeToHex(input)
	if err != nil {
		return nil
	}
	aux := make([]byte, len(hh))
	copy(aux[:], hh[:])
	*b = aux
	return nil
}

// Hex returns a hexadecimal representation
func (b ArgBytes) Hex() string {
	bb, _ := b.MarshalText()
	return string(bb)
}

// ArgBytesPtr helps to marshal byte array values provided in the RPC requests
func ArgBytesPtr(b []byte) *ArgBytes {
	bb := ArgBytes(b)

	return &bb
}

// ArgBig helps to marshal big number values provided in the RPC requests
type ArgBig big.Int

// UnmarshalText unmarshals an instance of ArgBig into an array of bytes
func (a *ArgBig) UnmarshalText(input []byte) error {
	buf, err := decodeToHex(input)
	if err != nil {
		return err
	}

	b := new(big.Int)
	b.SetBytes(buf)
	*a = ArgBig(*b)

	return nil
}

// MarshalText marshals an array of bytes into an instance of ArgBig
func (a ArgBig) MarshalText() ([]byte, error) {
	b := (*big.Int)(&a)

	return []byte("0x" + b.Text(Base)), nil
}

// Hex returns a hexadecimal representation
func (b ArgBig) Hex() string {
	bb, _ := b.MarshalText()
	return string(bb)
}

func decodeToHex(b []byte) ([]byte, error) {
	str := string(b)
	str = strings.TrimPrefix(str, "0x")
	if len(str)%2 != 0 {
		str = "0" + str
	}
	return DecodeString(str)
}

func encodeToHex(b []byte) []byte {
	str := EncodeToString(b)
	if len(str)%2 != 0 {
		str = "0" + str
	}
	return []byte("0x" + str)
}

// ArgHash represents a common.Hash that accepts strings
// shorter than 64 bytes, like 0x00
type ArgHash common.Hash

// UnmarshalText unmarshals from text
func (arg *ArgHash) UnmarshalText(input []byte) error {
	if !IsValid(string(input)) {
		return fmt.Errorf("invalid hash, it needs to be a hexadecimal value")
	}

	str := strings.TrimPrefix(string(input), "0x")
	*arg = ArgHash(common.HexToHash(str))
	return nil
}

// Hash returns an instance of common.Hash
func (arg *ArgHash) Hash() common.Hash {
	result := common.Hash{}
	if arg != nil {
		result = common.Hash(*arg)
	}
	return result
}

// ArgAddress represents a common.Address that accepts strings
// shorter than 32 bytes, like 0x00
type ArgAddress common.Address

// UnmarshalText unmarshals from text
func (b *ArgAddress) UnmarshalText(input []byte) error {
	if !IsValid(string(input)) {
		return fmt.Errorf("invalid address, it needs to be a hexadecimal value")
	}

	str := strings.TrimPrefix(string(input), "0x")
	*b = ArgAddress(common.HexToAddress(str))
	return nil
}

// Address returns an instance of common.Address
func (arg *ArgAddress) Address() common.Address {
	result := common.Address{}
	if arg != nil {
		result = common.Address(*arg)
	}
	return result
}

const (
	// Base represents the hexadecimal base, which is 16
	Base = 16

	// BitSize64 64 bits
	BitSize64 = 64
)

// DecError represents an error when decoding a hex value
type DecError struct{ msg string }

func (err DecError) Error() string { return err.msg }

// EncodeToHex generates a hex string based on the byte representation, with the '0x' prefix
func EncodeToHex(str []byte) string {
	return "0x" + EncodeToString(str)
}

// EncodeToString is a wrapper method for EncodeToString
func EncodeToString(str []byte) string {
	return EncodeToString(str)
}

// DecodeString returns the byte representation of the hexadecimal string
func DecodeString(str string) ([]byte, error) {
	return DecodeString(str)
}

// DecodeHex converts a hex string to a byte array
func DecodeHex(str string) ([]byte, error) {
	str = strings.TrimPrefix(str, "0x")

	return DecodeString(str)
}

// MustDecodeHex type-checks and converts a hex string to a byte array
func MustDecodeHex(str string) []byte {
	buf, err := DecodeHex(str)
	if err != nil {
		panic(fmt.Errorf("could not decode hex: %v", err))
	}

	return buf
}

// DecodeUint64 type-checks and converts a hex string to a uint64
func DecodeUint64(str string) uint64 {
	i := DecodeBig(str)
	return i.Uint64()
}

// EncodeUint64 encodes a number as a hex string with 0x prefix.
func EncodeUint64(i uint64) string {
	enc := make([]byte, 2, 10) //nolint:gomnd
	copy(enc, "0x")
	return string(strconv.AppendUint(enc, i, Base))
}

// BadNibble is a nibble that is bad
const BadNibble = ^uint64(0)

// DecodeNibble decodes a byte into a uint64
func DecodeNibble(in byte) uint64 {
	switch {
	case in >= '0' && in <= '9':
		return uint64(in - '0')
	case in >= 'A' && in <= 'F':
		return uint64(in - 'A' + 10) //nolint:gomnd
	case in >= 'a' && in <= 'f':
		return uint64(in - 'a' + 10) //nolint:gomnd
	default:
		return BadNibble
	}
}

// EncodeBig encodes bigint as a hex string with 0x prefix.
// The sign of the integer is ignored.
func EncodeBig(bigint *big.Int) string {
	numBits := bigint.BitLen()
	if numBits == 0 {
		return "0x0"
	}

	return fmt.Sprintf("%#x", bigint)
}

// DecodeBig converts a hex number to a big.Int value
func DecodeBig(hexNum string) *big.Int {
	str := strings.TrimPrefix(hexNum, "0x")
	createdNum := new(big.Int)
	createdNum.SetString(str, Base)

	return createdNum
}

// IsValid checks if the provided string is a valid hexadecimal value
func IsValid(s string) bool {
	str := strings.TrimPrefix(s, "0x")
	for _, b := range []byte(str) {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F') {
			return false
		}
	}
	return true
}
