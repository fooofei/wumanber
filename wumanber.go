package wumanber

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

const (
	blockSize = 3
)

// hashBlock will hash a block which size is blockSize=3
func hashBlock(s []byte) uint32 {
	var b [4]byte
	switch len(s) {
	case 3:
		b[2] = s[2]
		b[1] = s[1]
		b[0] = s[0]
	default:
		b[0] = 0 // for set breakpoint
	}
	i := binary.LittleEndian.Uint32(b[:])
	return i
}

func hashBlock2(s []byte, end int) uint32 {
	return hashBlock(s[end-blockSize : end])
}

func toUint16(a, b byte) uint16 {
	var c [2]byte
	c[0] = a
	c[1] = b
	return binary.LittleEndian.Uint16(c[:])
}

type prefixHashPair struct {
	PrefixHash uint32
	Index      int
}

type prefixHashItem []prefixHashPair

type WuManber struct {
	Patterns          [][]byte
	minSubPatternSize int // 并不是所有 pattern 里取长度最小的，因此命名为 subPattern
	tableSize         int
	shiftTable        []int
	prefixHashTable   []prefixHashItem
	byteTable         [][]int // pattern length = 1
	shortTable        [][]int // pattern length = 2

}

func getTableSize(patternCount int, patternMinSize int) int {
	primes := []int{1003, 10007, 100003, 1000003, 10000019, 100000007}
	threshold := 10 * patternMinSize
	for _, p := range primes {
		if p > patternCount && p/patternCount > threshold {
			return p
		}
	}
	return primes[len(primes)-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getMinSubPatternSize(patterns [][]byte) int {
	minSubPatternSize := math.MaxUint32
	for _, p := range patterns {
		if len(p) >= blockSize {
			minSubPatternSize = min(minSubPatternSize, len(p))
		}
	}
	return minSubPatternSize
}

func (w *WuManber) hashBlock(s []byte, end int) uint32 {
	h := hashBlock2(s, end)
	return h % uint32(w.tableSize)
}

func (w *WuManber) add1(c byte, index int) {
	if w.byteTable == nil {
		w.byteTable = make([][]int, 0x100)
	}
	w.byteTable[c] = append(w.byteTable[c], index)
}

func (w *WuManber) add2(c uint16, index int) {
	if w.shortTable == nil {
		w.shortTable = make([][]int, 0x100_00)
	}
	w.shortTable[c] = append(w.shortTable[c], index)
}

func (w *WuManber) add3(pattern []byte, index int) {
	minSize := w.minSubPatternSize
	// assert(blockSize <= len(pattern))
	for j := minSize; j >= blockSize; j-- {
		h := w.hashBlock(pattern, j)
		w.shiftTable[h] = min(minSize-j, w.shiftTable[h])
	}
	// 等价与 for 循环的首次
	h := w.hashBlock(pattern, minSize)
	ph := w.hashBlock(pattern, blockSize)
	w.prefixHashTable[h] = append(w.prefixHashTable[h], prefixHashPair{
		PrefixHash: ph,
		Index:      index,
	})
}

func (w *WuManber) add(pattern []byte, index int) {
	switch len(pattern) {
	case 1:
		w.add1(pattern[0], index)
	case 2:
		w.add2(toUint16(pattern[0], pattern[1]), index)
	default:
		w.add3(pattern, index)
	}
}

func New(patterns [][]byte) (*WuManber, error) {
	// init block
	patternCount := len(patterns)
	if patternCount == 0 {
		return nil, errors.New("failed init, cannot work without patterns")
	}

	minSubPatternSize := getMinSubPatternSize(patterns)

	tableSize := getTableSize(patternCount, minSubPatternSize)

	w := &WuManber{
		Patterns:          patterns,
		minSubPatternSize: minSubPatternSize,
		tableSize:         tableSize,
		shiftTable:        make([]int, tableSize),
		prefixHashTable:   make([]prefixHashItem, tableSize),
		byteTable:         nil,
		shortTable:        nil,
	}
	for i := range w.shiftTable {
		w.shiftTable[i] = minSubPatternSize - blockSize + 1
	}
	for i := range patterns {
		w.add(patterns[i], i)
	}
	return w, nil
}

// 短字符性能特别差 不建议使用
// return true for continue search others
// false for break search
func (w *WuManber) search1(text []byte, matchedCallback func(needle []byte, needleIndex, textIndex int) bool) bool {
	if len(w.byteTable) == 0 {
		return true
	}
	for i := range text {
		c := text[i]
		same := w.byteTable[c]
		if len(same) > 0 {
			for _, index := range same {
				if !matchedCallback(w.Patterns[index], index, i) {
					return false
				}
			}
		}
	}
	return true
}

func (w *WuManber) search2(text []byte, matchedCallback func(needle []byte, needleIndex, textIndex int) bool) bool {
	if len(w.shortTable) == 0 {
		return true
	}
	for i := 0; i < len(text)-1; i++ {
		c := toUint16(text[i], text[i+1])
		same := w.shortTable[c]
		if len(same) > 0 {
			for _, index := range same {
				if !matchedCallback(w.Patterns[index], index, i) {
					return false
				}
			}
		}
	}
	return true
}

func (w *WuManber) search3(text []byte, matchedCallback func(needle []byte, needleIndex, textIndex int) bool) {
	i := w.minSubPatternSize
loop:
	for i <= len(text) {
		h := w.hashBlock(text, i)
		shift := w.shiftTable[h]
		if shift == 0 {
			windowOff := i - w.minSubPatternSize
			prefixHash := w.hashBlock(text, windowOff+blockSize)
			samePrefixHashs := &w.prefixHashTable[h] // assign to a value for debug
			for _, item := range *samePrefixHashs {
				if prefixHash == item.PrefixHash {
					needle := w.Patterns[item.Index]
					textWindow := text[windowOff:]
					if len(textWindow) >= len(needle) && bytes.Equal(textWindow[:len(needle)], needle) {
						if !matchedCallback(needle, item.Index, windowOff) {
							break loop
						}
					}
				}
			}
			shift = 1
		}
		i += shift
	}
}

func (w *WuManber) Search(text []byte, matchedCallback func(needle []byte, needleIndex, textIndex int) bool) {
	continueSearch := w.search1(text, matchedCallback)
	if continueSearch {
		continueSearch = w.search2(text, matchedCallback)
	}
	if continueSearch {
		w.search3(text, matchedCallback)
	}
}
