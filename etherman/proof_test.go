package etherman

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestConvertProof(t *testing.T) {
	validInputProof, err := generateProof()
	require.NoError(t, err)

	validRawProof, err := hexutil.Decode(validInputProof)
	require.NoError(t, err)

	validProof, err := BytesToProof(validRawProof)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		expected    Proof
		expectedErr string
	}{
		{
			name:        "valid proof",
			input:       validInputProof,
			expected:    validProof,
			expectedErr: "",
		},
		{
			name:        "invalid proof length",
			input:       "0x1234",
			expectedErr: fmt.Sprintf("invalid proof length. Expected length: %d, Actual length %d", ProofLength*HashLength*2+2, 6),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertProof(tt.input)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.True(t, result.Equals(tt.expected), "expected: %v, actual: %v", tt.expected, result)
			}
		})
	}
}
