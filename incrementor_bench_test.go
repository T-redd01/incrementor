package incrementor

import (
	"testing"
)

func BenchmarkIncrementorPubAdd(b *testing.B) {
	tests := []struct{
		name string
		numKeys int
	}{
		{
			"LowKeySize",
			64,
		},
		{
			"MidKeySize",
			5000,
		},
		{
			"HighKeySize",
			100000,
		},
	}

	for _, test := range tests {
		b.Run(test.name, func (b *testing.B) {
			inc := NewIncrementor(test.numKeys)
			genKeys := make([][]byte, test.numKeys)

			for i := 0; i < test.numKeys; i++ {
				genKeys[i] = genMathRandBytes(SECTION_KEY_SIZE)
			}

			for b.Loop() {
				inc.keyCount = 0 // reset keys
				for _, gk := range genKeys {
					_ = inc.Add(gk)
				}
			}

			nsPerItem := float64(b.Elapsed().Nanoseconds()) / float64(b.N * test.numKeys)
			b.ReportMetric(nsPerItem, "ns/item")
		})
	}
}

func BenchmarkIncrementorPubGet(b *testing.B) {
	tests := []struct{
		name string
		numKeys int
	}{
		{
			"LowKeySize",
			64,
		},
		{
			"MidKeySize",
			5000,
		},
		{
			"HighKeySize",
			100000,
		},
	}

	for _, test := range tests {
		b.Run(test.name, func (b *testing.B) {
			inc := NewIncrementor(test.numKeys)
			genKeys := make([][]byte, test.numKeys)

			for i := 0; i < test.numKeys; i++ {
				genKeys[i] = genMathRandBytes(SECTION_KEY_SIZE)
				err := inc.Add(genKeys[i])
				if err != nil {
					b.Fatalf("Error Add: %v", err)
				}
			}

			i := 0
			for b.Loop() {
				_, _ = inc.Get(genKeys[i%test.numKeys])
				i++
			}
		})
	}
}
