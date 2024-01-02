package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeToHex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    []byte
		expected []byte
	}{
		{
			input:    []byte{},
			expected: []byte("0x"),
		},
		{
			input:    []byte{0x01},
			expected: []byte("0x01"),
		},
		{
			input:    []byte{0x01, 0x23, 0x45},
			expected: []byte("0x012345"),
		},
		{
			input:    []byte{0x00, 0xff, 0x80},
			expected: []byte("0x00ff80"),
		},
	}

	for _, tc := range testCases {
		actual := encodeToHex(tc.input)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestDecodeToHex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    []byte
		expected []byte
	}{
		{
			input:    []byte("0x"),
			expected: []byte{},
		},
		{
			input:    []byte("0x01"),
			expected: []byte{0x01},
		},
		{
			input:    []byte("0x012345"),
			expected: []byte{0x01, 0x23, 0x45},
		},
		{
			input:    []byte("0x00ff80"),
			expected: []byte{0x00, 0xff, 0x80},
		},
	}

	for _, tc := range testCases {
		actual, err := decodeToHex(tc.input)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestHexIsValid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected bool
	}{
		{
			input:    "0x",
			expected: true,
		},
		{
			input:    "0x01",
			expected: true,
		},
		{
			input:    "0x012345",
			expected: true,
		},
		{
			input:    "0x00ff80",
			expected: true,
		},
		{
			input:    "0xg",
			expected: false,
		},
		{
			input:    "0x123",
			expected: true,
		},
		{
			input:    "0xabcdefg",
			expected: false,
		},
		{
			input:    "0xABCDEF",
			expected: true,
		},
	}

	for _, tc := range testCases {
		actual := hexIsValid(tc.input)
		assert.Equal(t, tc.expected, actual)
	}
}
