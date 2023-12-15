package etherman

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateProof(t *testing.T) string {
	t.Helper()

	var buf bytes.Buffer
	buf.WriteString("0x")

	for i := 0; i < 2*ProofLength; i++ {
		hash := generateRandomHexString(t, HashLength)
		_, err := buf.WriteString(hash)
		require.NoError(t, err)
	}

	return buf.String()
}

func generateRandomHexString(t *testing.T, length int) string {
	t.Helper()

	numBytes := length / 2
	randomBytes := make([]byte, numBytes)

	_, err := rand.Read(randomBytes)
	require.NoError(t, err)

	return hex.EncodeToString(randomBytes)
}
