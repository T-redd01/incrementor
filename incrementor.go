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

	inc.bufferLock.RLock()
	defer inc.bufferLock.RUnlock()

	keyIdx, cmp := inc.binSearch(padKey, 0, inc.KeyCount())
	if cmp != 0 {
		return 0, errors.New("Key not found")
	}

	inc.keyLocks[keyIdx].Lock() // lock section in buffer before using

	sectionIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	sectionVal := inc.buffer[(sectionIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE):(sectionIdx + SECTION_SIZE)]

	endian := binary.NativeEndian

	n := endian.Uint32(sectionVal)
	n++
	endian.PutUint32(sectionVal, n)

	inc.keyLocks[keyIdx].Unlock() // unlock section in buffer after finish

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

	insertSecIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	lastSecIdx := inc.sectionIdxFromKeyIdx(inc.KeyCount() - 1)

	// copy from key insert idx in buffer until one section after last key in buffer
	// makes space / moves everything one section down so new key can be inserted
	copy(
		inc.buffer[(insertSecIdx + SECTION_SIZE):(lastSecIdx + (SECTION_SIZE * 2))],
		inc.buffer[insertSecIdx:(lastSecIdx + SECTION_SIZE)],
	)

	endian := binary.NativeEndian
	copy(inc.buffer[insertSecIdx:(insertSecIdx + SECTION_KEY_SIZE)], padKey)
	endian.PutUint64(
		inc.buffer[(insertSecIdx + SECTION_KEY_SIZE):(insertSecIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE)],
		uint64(time.Now().UnixMilli()),
	)

	inc.keyCount = inc.keyCount + 1

	return nil
}

/*
 * Del: Delete a key from the buffer
 * @key: key to delete
 *
 * Return: error if key does not exist, nil otherwise
 */
func (inc *Incrementor) Del(key []byte) error {
	// NOTE: padKey takes key and checks whether it is
	//       larger than the allowed key size, if so
	//       it will return an error, if not it will
	//       create a new with total allowed key size
	//       and copy the requested key into effectively
	//       padding it to key size if it is shorter
	//       and it always returns a new slice from make
	//       but for some reason if we decide to lock
	//       the buffer for writing after padding the key
	//       we get a data race, even more sus the
	//       keyCount field gets race aswell
	//       WHY, HOW?
	//       HOW, WHY?
	//       it doesnt make sense
	//       explore this further in the future
	inc.bufferLock.Lock()
	defer inc.bufferLock.Unlock()

	padKey, err := inc.padKey(key)
	if err != nil {
		return err
	}


	keyIdx, cmp := inc.binSearch(padKey, 0, inc.KeyCount())
	if cmp != 0 {
		return errors.New("Key not found")
	}

	// index of first byte of section to delete and section after section to delete
	delSecIdx := inc.sectionIdxFromKeyIdx(keyIdx)
	nextSecIdx := delSecIdx + SECTION_SIZE

	// index of last byte of last section and second last section
	endLstSecIdx := inc.sectionIdxFromKeyIdx(inc.KeyCount() - 1) + SECTION_SIZE
	endSecLstSecIdx := endLstSecIdx - SECTION_SIZE

	// move everything from section after delete section up
	// removing / overwriting delete section
	copy(
		inc.buffer[delSecIdx:endSecLstSecIdx],
		inc.buffer[nextSecIdx:endLstSecIdx],
	)

	// 0 last used key section
	for i := endSecLstSecIdx; i < endLstSecIdx; i++ {
		inc.buffer[i] = 0
	}

	inc.keyCount = inc.keyCount - 1

	return nil
}

/*
 * Resize: Expand or Shrink buffer
 * @keys: number of keys for buffer to hold, not less than 1
 *
 * Return: error if more keys in buffer than requested keys
 *         or requested less than 1 key
 */
func (inc *Incrementor) Resize(keys int) error {
	inc.bufferLock.Lock() // block all reading and writing
	defer inc.bufferLock.Unlock() // allow all reading and writing

	if keys < inc.KeyCount() {
		return errors.New("Requested key count lower than current key count")
	}

	if keys <= 0 {
		return errors.New("Requested key count must ne atleast 1")
	}

	// allocate new buffer and copy old values
	newBuf := make([]byte, keys * SECTION_SIZE)
	copy(newBuf, inc.buffer[0:(inc.sectionIdxFromKeyIdx(inc.KeyCount() - 1) + SECTION_SIZE)]) // only copy portion of buffer in use
	inc.buffer = newBuf

	// allocate new locks for keys
	newKeyLocks := make([]sync.Mutex, keys)
	inc.keyLocks = newKeyLocks

	return nil
}

/*
 * padKey: pad a key to SECTION_KEY_SIZE
 * @key: key to be padded
 *
 * Return: new byte slice of size SECTION_KEY_SIZE with key added
 *         error if key is larger than SECTION_SIZE
 */
func (inc Incrementor) padKey(key []byte) ([]byte, error) {
	if len(key) > SECTION_KEY_SIZE {
		return []byte{}, errors.New("Key size limit exceeded")
	}

	pk := make([]byte, SECTION_KEY_SIZE)
	copy(pk, key)

	return pk, nil
}

/*
 * sectionIdxFromKeyIdx: get section byte index in buffer from section key index
 * @keyIdx: section key index
 *
 * Return: index in buffer that section starts at
 */
func (inc Incrementor) sectionIdxFromKeyIdx(keyIdx int) int {
	return keyIdx * SECTION_SIZE // byte index of key / section in buffer
}

/*
 * keyIdxFromSectionIdx: get key index of section in buffer from section index
 * @sectionIdx: index of start of section in buffer
 *
 * Return: key index for section in buffer
 */
func (inc Incrementor) keyIdxFromSectionIdx(sectionIdx int) int {
	return sectionIdx / SECTION_SIZE // key number from byte index in buffer
}

/*
 * binSearch: perform binary search for @key in buffer
 * @key: key to search for in buffer, must be padded to SECTION_KEY_SIZE
 * @l: low bound key index of partition to search in
 * @h: high bound key index of partition to search in
 *
 * Return: last checked section key index and compare result of
 *         last checked section with key
 */
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

/*
 * printSections: print each section in buffer
 */
func (inc Incrementor) printSections() {
	for i := 0; i < inc.KeyCount(); i++ {
		secIdx := inc.sectionIdxFromKeyIdx(i)
		sectionKey := inc.buffer[secIdx:(secIdx + SECTION_KEY_SIZE)]
		sectionMnt := inc.buffer[(secIdx + SECTION_KEY_SIZE):(secIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE)]
		sectionVal := inc.buffer[(secIdx + SECTION_KEY_SIZE + SECTION_MNT_SIZE):(secIdx + SECTION_SIZE)]

		endian := binary.NativeEndian
		mnt := time.UnixMilli(int64(endian.Uint64(sectionMnt)))
		val := endian.Uint32(sectionVal)
		fmt.Println(fmt.Sprintf("%v:%s:%d", sectionKey, mnt, val))
	}
}
