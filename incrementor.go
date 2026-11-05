package incrementor

import (
	"sync"
	"bytes"
	"errors"
	"encoding/binary"
)

const (
	SECTION_SIZE = 64
	SECTION_KEY_SIZE = 52
	SECTION_MNT_SIZE = 8
	SECTION_VAL_SIZE = 4
)

type Incrementor struct {
	buffer []byte
	useKeys int
	bufferLock sync.RWMutex
	keyLocks []sync.Mutex
}

func NewIncrementor(keys int) *Incrementor {
	return &Incrementor{
		buffer: make([]byte, keys * SECTION_SIZE),
		useKeys: 0,
		bufferLock: sync.RWMutex{},
		keyLocks: make([]sync.Mutex, keys),
	}
}

func (inc Incrementor) UseKeys() int {
	return inc.useKeys
}

func (inc Incrementor) NumKeys() int {
	return len(inc.buffer) / SECTION_SIZE
}

func (inc *Incrementor) Has(key []byte) bool {
	padKey, err := inc.padKey(key)
	if err != nil {
		return false
	}

	inc.bufferLock.RLock() // unlock for read lock is not deferred
	_, cmp := inc.binSearch(padKey, 0, inc.useKeys)
	inc.bufferLock.RUnlock() // read lock unlocked here
	if cmp != 0 {
		return false
	}

	return true
}

func (inc *Incrementor) Get(key []byte) (uint32, error) {
	padKey, err := inc.padKey(key)
	if err != nil {
		return 0, err
	}

	inc.bufferLock.RLock() // unlock for read lock is not deferred

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.UseKeys())
	if cmp != 0 {
		inc.bufferLock.RUnlock() // unlock before return
		return 0, errors.New("Key not found")
	}

	inc.keyLocks[keyIdx].Lock() // lock section in buffer before using

	sectionIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	// last 4 bytes / value part of section
	sectionVal := inc.buffer[(sectionIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE):(sectionIdx + SECTION_SIZE)]
	// IMPORTANT: check endianess before following through here
	n := binary.BigEndian.Uint32(sectionVal)
	n++
	binary.BigEndian.PutUint32(sectionVal, n)

	inc.keyLocks[keyIdx].Unlock() // unlock section in buffer after finish

	inc.bufferLock.RUnlock()
	return n, nil
}

func (inc *Incrementor) Add(key []byte) error {
	if (len(inc.buffer) / SECTION_SIZE) == inc.UseKeys() {
		return errors.New("Buffer is full")
	}

	padKey, err := inc.padKey(key)
	if err != nil {
		return err
	}

	inc.bufferLock.Lock() // lock buffer for writing

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.UseKeys())
	if cmp == 0 {
		inc.bufferLock.Unlock()
		return errors.New("Key already exists in buffer")
	}

	prev := make([]byte, SECTION_SIZE)
	curr := make([]byte, SECTION_SIZE)
	for i := keyIdx; i < (inc.UseKeys() + 1); i++ {
		// save current section
		currSecIdx := inc.sectionIdxFromKeyIdx(i)
		copy(curr, inc.buffer[currSecIdx:(currSecIdx + SECTION_SIZE)])
		// write previous into current
		copy(inc.buffer[currSecIdx:(currSecIdx + SECTION_SIZE)], prev)
		// set previous to current
		prev := curr
	}

	inc.bufferLock.Unlock()
	return nil
}

func (inc Incrementor) padKey(key []byte) ([]byte, error) {
	if len(key) > SECTION_KEY_SIZE {
		return []byte{}, errors.New("Key size limit exceeded")
	}

	pk := make([]byte, SECTION_KEY_SIZE)
	copy(pk, key)

	return pk, nil
}

func (inc Incrementor) sectionIdxFromKeyIdx(keyIdx int) int {
	return keyIdx * SECTION_SIZE // byte index of key / section in buffer
}

func (inc Incrementor) keyIdxFromSectionIdx(sectionIdx int) int {
	return sectionIdx / SECTION_SIZE // key number from byte index in buffer
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
