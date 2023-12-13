package etherman

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
)

func generateRandomHexString(length int) (string, error) {
	numBytes := length / 2
	randomBytes := make([]byte, numBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

func generateProof() (string, error) {
	var buf bytes.Buffer
	buf.WriteString("0x")

	for i := 0; i < 2*ProofLength; i++ {
		hash, err := generateRandomHexString(HashLength)
		if err != nil {
			return "", err
		}

		_, err = buf.WriteString(hash)
		if err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}
