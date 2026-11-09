package incrementor

import (
	"testing"
	"math/rand/v2"
)

func genMathRandBytes(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < n; i += 8 {
		rnum := rand.Uint64()
		for j := 0; j < 8 && j+i < n; j++ {
			b[j+i] = byte(rnum >> (8 * uint(j)))
		}
	}

	return b
}

func TestSome(t *testing.T) {
	for i := 0; i < 15; i++ {
		n := rand.IntN(11)
		t.Logf("got rand n: %v", n)
	}
}
