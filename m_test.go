package incrementor

import (
	"testing"
)

func TestMakeNew(t *testing.T) {
	/*
	buffer := []byte{
		// are
		97, 114, 101, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// hello
		104, 101, 108, 108, 111, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// strings
		115, 116, 114, 105, 110, 103, 115, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// these
		116, 104, 101, 115, 101, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// world
		119, 111, 114, 108, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	*/
	inc := NewIncrementor(5)
	/*
	inc.buffer = buffer
	inc.keyCount = 5
	*/

	err := inc.Add([]byte{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("Error while adding: %v", err)
	}

	err = inc.Add([]byte{4,3,2,1})
	if err != nil {
		t.Fatalf("Error while adding: %v", err)
	}

	err = inc.Add([]byte{5,3,2,1})
	if err != nil {
		t.Fatalf("Error while adding: %v", err)
	}

	err = inc.Add([]byte{0,3,2,1})
	if err != nil {
		t.Fatalf("Error while adding: %v", err)
	}

	err = inc.Add([]byte{6,3,2,1})
	if err != nil {
		t.Fatalf("Error while adding: %v", err)
	}

	for i := 0; i < inc.KeyLimit(); i++ {
		secIdx := inc.sectionIdxFromKeyIdx(i)
		t.Logf("key[%v]: %v", i, inc.buffer[secIdx:(secIdx + SECTION_SIZE)])
	}
}
