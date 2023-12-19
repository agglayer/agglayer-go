package etherman

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/0xPolygonHermez/zkevm-node/encoding"
)

type Proof [ProofLength][HashLength]byte

func (p Proof) Equals(other Proof) bool {
	for i := 0; i < len(p); i++ {
		if !bytes.Equal(p[i][:], other[i][:]) {
			return false
		}
	}
	return true
}

func BytesToProof(data []byte) (Proof, error) {
	// Check if the length of the input data is compatible with the Proof type
	expectedLength := ProofLength * HashLength
	if len(data) != expectedLength {
		return Proof{}, fmt.Errorf("invalid byte slice length. Expected length: %d, Actual length: %d", expectedLength, len(data))
	}

	// Create a Proof type and copy the data into it
	var proof Proof
	for i := 0; i < ProofLength; i++ {
		copy(proof[i][:], data[i*HashLength:(i+1)*HashLength])
	}

	return proof, nil
}

func ConvertProof(p string) (Proof, error) {
	const (
		expectedLength = ProofLength*HashLength*2 + 2
		rawHashLength  = 2 * HashLength // 1 byte gets substituted with 2 hex digits (1 hex digit = nibble = 4 bits)
	)

	if len(p) != expectedLength {
		return Proof{}, fmt.Errorf("invalid proof length. Expected length: %d, Actual length %d", expectedLength, len(p))
	}
	p = strings.TrimPrefix(p, "0x")
	proof := Proof{}
	for i := 0; i < ProofLength; i++ {
		data := p[i*rawHashLength : (i+1)*rawHashLength]
		p, err := encoding.DecodeBytes(&data)
		if err != nil {
			return Proof{}, fmt.Errorf("failed to decode proof, err: %w", err)
		}
		var aux [HashLength]byte
		copy(aux[:], p)
		proof[i] = aux
	}
	return proof, nil
}
