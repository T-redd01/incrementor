package incrementor

import (
	"testing"
	"sync"
	"math/rand/v2"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
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
				err := inc.Add(part[j])
				require.NoError(t, err)
			}
		}()
	}

	close(blocker)
	wg.Wait()
}

func TestIncrementorPubDelRace(t *testing.T) {
	numKeys := 100
	genKeys := make([][]byte, numKeys)

	inc := NewIncrementor(numKeys)

	for i := 0; i < numKeys; i++ {
		genKeys[i] = genMathRandBytes(SECTION_KEY_SIZE)
		err := inc.Add(genKeys[i])
		require.NoError(t, err)
	}

	wg := sync.WaitGroup{}
	blocker := make(chan struct{})

	partSize := numKeys / 4
	for i := 0; i < numKeys; i += partSize {
		var part [][]byte
		for j := 0; j < partSize && j+i < numKeys; j++ {
			part = append(part, genKeys[i+j])
		}

		wg.Add(1)
		go func (pt [][]byte) {
			defer wg.Done()

			<-blocker

			for _, p := range pt {
				_ = inc.Del(p)
			}
		}(part)
	}


	close(blocker)
	wg.Wait()

	// all keys deleted
	assert.Equal(t, 0, inc.KeyCount())
}
