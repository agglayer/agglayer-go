package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestArgUint64_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		arg      ArgUint64
		expected []byte
	}{
		{
			name:     "Positive number",
			arg:      ArgUint64(123),
			expected: []byte("0x7b"),
		},
		{
			name:     "Zero",
			arg:      ArgUint64(0),
			expected: []byte("0x0"),
		},
		{
			name:     "Large number",
			arg:      ArgUint64(18446744073709551615),
			expected: []byte("0xffffffffffffffff"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.arg.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestArgUint64_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected ArgUint64
		wantErr  bool
	}{
		{
			name:     "Valid input",
			input:    []byte("0x7b"),
			expected: ArgUint64(123),
			wantErr:  false,
		},
		{
			name:     "Invalid input",
			input:    []byte("0xabcj"),
			expected: ArgUint64(0),
			wantErr:  true,
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: ArgUint64(0),
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var arg ArgUint64
			err := arg.UnmarshalText(test.input)
			if (err != nil) != test.wantErr {
				t.Errorf("Unexpected error status, got error: %v, want error: %v", err, test.wantErr)
			}
			if arg != test.expected {
				t.Errorf("Unexpected result, got: %v, want: %v", arg, test.expected)
			}
		})
	}
}

func TestArgBytes_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		arg      ArgBytes
		expected []byte
	}{
		{
			name:     "Non-empty bytes",
			arg:      ArgBytes{0x01, 0x02, 0x03},
			expected: []byte("0x010203"),
		},
		{
			name:     "Empty bytes",
			arg:      ArgBytes{},
			expected: []byte("0x"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.arg.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestArgBytes_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected ArgBytes
		wantErr  bool
	}{
		{
			name:     "Valid input",
			input:    []byte("0x010203"),
			expected: ArgBytes{0x01, 0x02, 0x03},
			wantErr:  false,
		},
		{
			name:     "Invalid input",
			input:    []byte("0xabcj"),
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: ArgBytes{},
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var arg ArgBytes
			err := arg.UnmarshalText(test.input)
			if (err != nil) != test.wantErr {
				t.Errorf("Unexpected error status, got error: %v, want error: %v", err, test.wantErr)
			}
			assert.Equal(t, test.expected, arg)
		})
	}
}

func TestArgHash_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected ArgHash
		wantErr  bool
	}{
		{
			name:     "Valid input",
			input:    []byte("0x1234567890abcdef"),
			expected: ArgHash(common.HexToHash("1234567890abcdef")),
			wantErr:  false,
		},
		{
			name:     "Invalid input",
			input:    []byte("0xabcj"),
			expected: ArgHash{},
			wantErr:  true,
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: ArgHash{},
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var arg ArgHash
			err := arg.UnmarshalText(test.input)
			if (err != nil) != test.wantErr {
				t.Errorf("Unexpected error status, got error: %v, want error: %v", err, test.wantErr)
			}
			assert.Equal(t, test.expected, arg)
		})
	}
}
