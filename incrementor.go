package incrementor

import (
	"sync"
)

const (
	SECTION_SIZE = 64
	SECTION_KEY_SIZE = 56
)

type Incrementor struct {
	buffer []byte
	bufferLock sync.RWMutex
	keyLocks []sync.Mutex
}

func NewIncrementor(key int) *Incrementor {
	return &Incrementor{
		buffer: make([]byte, keys * SECTION_SIZE),
		bufferLock: sync.RWMutex{},
		keyLocks: make([]sync.Mutex, keys),
	}
}

func (inc Incrementor) SectionIdxFromKeyIdx(keyIdx int) int {
	return keyIdx * SECTION_SIZE
}

func (inc Incrementor) KeyIdxFromSectionIdx(sectionIdx int) int {
	return sectionIdx / SECTION_SIZE
}

func (inc Incrementor) NumKeys() {
	return len(inc.keyLocks)
}

func (inc Incrementor) bin_search(key []byte, l, h int) (int, int) {
	if (h == l) || ((h - l) == 1) {
		lSecIdx := inc.SectionIdxFromKeyIdx(l)
		return lSecIdx, bytes.Compare(inc.buffer[lSecIdx:(lSecIdx + SECTION_SIZE)], key)
	}

	mSecIdx := inc.SectionIdxFromKeyIdx(l + ((h - l) / 2))
	cmp := bytes.Compare(inc.buffer[mSecIdx:(mSecIdx + SECTION_SIZE)], key)
	if cmp > 0 {
		return inc.bin_search(key, l, inc.KeyIdxFromSectionIdx(mSecIdx))
	}

	if cmp < 0 {
		return inc.bin_search(key, inc.KeyIdxFromSectionIdx(mSecIdx), h)
	}

	return mSecIdx, cmp
}

