package incrementor

import (
	"testing"
	"sync"
	"math/rand/v2"

	"github.com/stretchr/testify/require"
)

func TestIncrementorPubGetRace(t *testing.T) {
	numKeys := 10
	genKeys := make([][]byte, numKeys)

	inc := NewIncrementor(numKeys)
	for i := 0; i < numKeys; i++ {
		key := genMathRandBytes(SECTION_KEY_SIZE)
		err := inc.Add(key)
		require.NoError(t, err)
		genKeys[i] = key
	}

	wg := sync.WaitGroup{}
	blocker := make(chan struct{})
	tasks := 1000
	for i := 0; i < tasks; i++ {
		wg.Add(1)
		go func () {
			defer wg.Done()

			n := rand.IntN(numKeys)
			<-blocker

			_, _ = inc.Get(genKeys[n])
		}()
	}

	close(blocker)
	wg.Wait()
}

func TestIncrementorPubAddRace(t *testing.T) {
	numKeys := 100
	genKeys := make([][]byte, numKeys)

	inc := NewIncrementor(numKeys)

	for i := 0; i < numKeys; i++ {
		genKeys[i] = genMathRandBytes(SECTION_KEY_SIZE)
	}

	wg := sync.WaitGroup{}
	blocker := make(chan struct{})

	partN := numKeys / 4
	for i := 0; i < numKeys; i += partN {
		part := genKeys[i:i+partN]

		wg.Add(1)
		go func () {
			defer wg.Done()

			<-blocker

			for j := 0; j < partN && j < numKeys; j++ {
				_ = inc.Add(part[j])
			}
		}()
	}

	close(blocker)
	wg.Wait()
}
