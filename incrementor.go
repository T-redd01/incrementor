package incrementor

import (
	"fmt"
	"sync"
	"bytes"
	"errors"
	"time"
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
	keyCount int
	bufferLock sync.RWMutex
	keyLocks []sync.Mutex
}

func NewIncrementor(keys int) *Incrementor {
	if keys <= 0 {
		panic("Incrementor must have atleast 1 key slot")
	}

	return &Incrementor{
		buffer: make([]byte, keys * SECTION_SIZE),
		keyCount: 0,
		bufferLock: sync.RWMutex{},
		keyLocks: make([]sync.Mutex, keys),
	}
}

/*
 * KeyCount: how many keys were added to buffer
 *
 * Return: number of keys in buffer
 */
func (inc Incrementor) KeyCount() int {
	return inc.keyCount
}

/*
 * KeyLimit: total number of keys buffer is able to hold
 *
 * Return: total number of slots for keys in buffer
 */
func (inc Incrementor) KeyLimit() int {
	return len(inc.buffer) / SECTION_SIZE
}

/*
 * Has: Searches for key in buffer
 * @key: key to search for
 *
 * Return: true if found key false otherwise
 */
func (inc *Incrementor) Has(key []byte) bool {
	padKey, err := inc.padKey(key)
	if err != nil {
		return false
	}

	inc.bufferLock.RLock() // unlock for read lock is not deferred
	_, cmp := inc.binSearch(padKey, 0, inc.keyCount)
	inc.bufferLock.RUnlock() // read lock unlocked here
	if cmp != 0 {
		return false
	}

	return true
}

/*
 * Get: find key and increment value
 * @key: key to search for
 *
 * Return: incremented value for key in buffer
 */
func (inc *Incrementor) Get(key []byte) (uint32, error) {
	padKey, err := inc.padKey(key)
	if err != nil {
		return 0, err
	}

	inc.bufferLock.RLock() // unlock for read lock is not deferred

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.KeyCount())
	if cmp != 0 {
		inc.bufferLock.RUnlock() // unlock before return
		return 0, errors.New("Key not found")
	}

	inc.keyLocks[keyIdx].Lock() // lock section in buffer before using

	// capture section value part slice for key
	// cast to 'uint32' and increment
	// overwrite captured section slice 
	// in buffer with new incremented value
	sectionIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	sectionVal := inc.buffer[(sectionIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE):(sectionIdx + SECTION_SIZE)]
	// IMPORTANT: check endianess before following through here
	endian := binary.NativeEndian
	n := endian.Uint32(sectionVal)
	n++
	endian.PutUint32(sectionVal, n)

	inc.keyLocks[keyIdx].Unlock() // unlock section in buffer after finish

	inc.bufferLock.RUnlock() // finish read operation on buffer
	return n, nil
}

/*
 * Add: Insert a key into the buffer
 * @key: key to insert
 *
 * Return: error if fail to add key nil otherwise
 */
func (inc *Incrementor) Add(key []byte) error {
	// lock buffer for writing, keycount is also mutated so check must be synchronized aswell
	inc.bufferLock.Lock()
	defer inc.bufferLock.Unlock()

	if inc.KeyLimit() == inc.KeyCount() {
		return errors.New("Buffer is full")
	}

	padKey, err := inc.padKey(key)
	if err != nil {
		return err
	}

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.KeyCount())
	if cmp == 0 {
		return errors.New("Key already exists in buffer")
	}

	// if last checked buffer section is less than key
	// and there are keys in the buffer
	// new key must go into the next section
	if cmp < 0 && inc.KeyCount() != 0 {
		keyIdx = keyIdx + 1
	}

	prev := make([]byte, SECTION_SIZE)

	// new key is written on first run to its correct position
	// mutate 'prev' to contain the values of the new 
	// section to be added
	copy(prev[0:SECTION_KEY_SIZE], key)
	endian := binary.NativeEndian
	endian.PutUint64(
		prev[SECTION_KEY_SIZE:(SECTION_KEY_SIZE + SECTION_MNT_SIZE)],
		uint64(time.Now().UnixMilli()),
	)

	curr := make([]byte, SECTION_SIZE)
	for i := keyIdx; i < (inc.KeyCount() + 1); i++ {
		// save current section
		currSecIdx := inc.sectionIdxFromKeyIdx(i)
		copy(curr, inc.buffer[currSecIdx:(currSecIdx + SECTION_SIZE)])
		// write previous into current
		copy(inc.buffer[currSecIdx:(currSecIdx + SECTION_SIZE)], prev)
		// write curr into prev
		copy(prev, curr)
	}

	inc.keyCount = inc.keyCount + 1

	return nil
}

func (inc *Incrementor) Del(key []byte) error {
	padKey, err := inc.padKey(key)
	if err != nil {
		return err
	}

	inc.bufferLock.Lock() // lock buffer for writing

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.KeyCount())
	if cmp != 0 {
		inc.bufferLock.Unlock()
		return errors.New("Key not found")
	}

	for keyIdx < (inc.KeyCount() - 1) {
		currSecIdx := inc.sectionIdxFromKeyIdx(keyIdx)
		nextSecIdx := inc.sectionIdxFromKeyIdx(keyIdx + 1)
		copy(inc.buffer[currSecIdx:(currSecIdx + SECTION_SIZE)], inc.buffer[nextSecIdx:(nextSecIdx + SECTION_SIZE)])

		keyIdx++
	}

	// 0 last used key section
	lstSecIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	for i := lstSecIdx; i < (lstSecIdx + SECTION_SIZE); i++ {
		inc.buffer[i] = 0
	}

	inc.keyCount = inc.keyCount - 1

	inc.bufferLock.Unlock() // allow reading and writing after mutation

	return nil
}

func (inc *Incrementor) Resize(keys int) error {
	//fmt.Println(fmt.Sprintf("key locks len: %v, cap: %v", len(inc.keyLocks), cap(inc.
	if keys < inc.KeyCount() {
		return errors.New("Requested key count lower than current key count")
	}

	if keys <= 0 {
		return errors.New("Requested key count must ne atleast 1")
	}

	inc.bufferLock.Lock() // block all reading and writing

	// allocate new buffer and copy old values
	newBuf := make([]byte, keys * SECTION_SIZE)
	copy(newBuf, inc.buffer[0:(inc.sectionIdxFromKeyIdx(inc.KeyCount() - 1) + SECTION_SIZE)]) // only copy portion of buffer in use
	inc.buffer = newBuf

	// allocate new locks for keys
	newKeyLocks := make([]sync.Mutex, keys)
	inc.keyLocks = newKeyLocks

	inc.bufferLock.Unlock() // allow all reading and writing

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

func (inc Incrementor) printSections() {
	for i := 0; i < inc.KeyCount(); i++ {
		secIdx := inc.sectionIdxFromKeyIdx(i)
		sectionKey := inc.buffer[secIdx:(secIdx + SECTION_KEY_SIZE)]
		sectionMnt := inc.buffer[(secIdx + SECTION_KEY_SIZE):(secIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE)]
		sectionVal := inc.buffer[(secIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE):(secIdx + SECTION_SIZE)]

		// IMPORTANT: check endianess
		endian := binary.NativeEndian
		mnt := time.UnixMilli(int64(endian.Uint64(sectionMnt)))
		val := endian.Uint32(sectionVal)
		fmt.Println(fmt.Sprintf("%x:%s:%d", sectionKey, mnt, val))
	}
}
