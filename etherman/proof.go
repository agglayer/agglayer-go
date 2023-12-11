package etherman

import (
	"fmt"
	"strings"

	"github.com/0xPolygon/cdk-validium-node/encoding"
)

type Proof [24][HashLength]byte

func ConvertProof(p string) (Proof, error) {
	const expectedLength = 24*HashLength*2 + 2
	if len(p) != expectedLength {
		return Proof{}, fmt.Errorf("invalid proof length. Expected length: %d, Actual length %d", expectedLength, len(p))
	}
	p = strings.TrimPrefix(p, "0x")
	proof := Proof{}
	for i := 0; i < 24; i++ {
		data := p[i*64 : (i+1)*64]
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
