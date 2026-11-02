package incrementor

import (
	"sync"
	"bytes"
)

const (
	SECTION_SIZE = 64
	SECTION_KEY_SIZE = 52
	SECTION_MNT_SIZE = 8
	SECTION_VAL_SIZE = 4
)

type Incrementor struct {
	buffer []byte
	numKeys int
	bufferLock sync.RWMutex
	keyLocks []sync.Mutex
}

func NewIncrementor(keys int) *Incrementor {
	return &Incrementor{
		buffer: make([]byte, keys * SECTION_SIZE),
		numKeys: 0,
		bufferLock: sync.RWMutex{},
		keyLocks: make([]sync.Mutex, keys),
	}
}

func (inc Incrementor) NumKeys() int {
	return inc.numKeys
}

func (inc *Incrementor) Has(key []byte) bool {
	if len(key) > SECTION_KEY_SIZE {
		return false
	}

	// pad key for safe slicing
	padKey := make([]byte, SECTION_KEY_SIZE)
	for i := 0; i < len(key); i++ {
		padKey[i] = key[i]
	}

	inc.bufferLock.RLock() // unlock for read lock is not deferred
	_, cmp := inc.binSearch(padKey, 0, inc.numKeys)
	inc.bufferLock.RUnlock() // read lock unlocked here
	if cmp != 0 {
		return false
	}

	return true
}

func (inc Incrementor) sectionIdxFromKeyIdx(keyIdx int) int {
	return keyIdx * SECTION_SIZE
}

func (inc Incrementor) keyIdxFromSectionIdx(sectionIdx int) int {
	return sectionIdx / SECTION_SIZE
}

func (inc Incrementor) binSearch(key []byte, l, h int) (int, int) {
	if (h == l) || ((h - l) == 1) {
		lSecIdx := inc.sectionIdxFromKeyIdx(l)
		return l, bytes.Compare(inc.buffer[lSecIdx:(lSecIdx + SECTION_KEY_SIZE)], key)
	}

	mKeyIdx := l + ((h - l) / 2)
	mSecIdx := inc.sectionIdxFromKeyIdx(mKeyIdx)
	cmp := bytes.Compare(inc.buffer[mSecIdx:(mSecIdx + SECTION_KEY_SIZE)], key)
	if cmp > 0 {
		return inc.binSearch(key, l, mKeyIdx)
	}

	if cmp < 0 {
		return inc.binSearch(key, mKeyIdx, h)
	}

	return mKeyIdx, cmp
}
