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
	numKeys := 100
	genKeys := make([][]byte, numKeys)

	for i := 0; i < numKeys; i++ {
		genKeys[i] = genMathRandBytes(SECTION_KEY_SIZE)
		t.Logf("before shuffle: [%v]%x", i, genKeys[i])
	}

	for i := 0; i < numKeys; i++ {
		rand.Shuffle(numKeys, func (i, j int) {
			tmp := genKeys[i]
			genKeys[i] = genKeys[j]
			genKeys[j] = tmp
		})
	}

	for i := 0; i < numKeys; i++ {
		t.Logf("after shuffle: [%v]%x", i, genKeys[i])
	}
}
