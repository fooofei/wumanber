package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

func (w *WuManber) hashBlock(s []byte, end int) uint32 {
	h := hashBlock2(s, end)
	return h % uint32(w.tableSize)
}

func New(patterns [][]byte) (*WuManber, error) {
	// init block
	patternCount := len(patterns)
	if patternCount == 0 {
		return nil, errors.New("failed init, cannot work without patterns")
	}

	minSubPatternSize := math.MaxUint32

	for _, p := range patterns {
		if len(p) < blockSize {
			// 简化处理，不支持小于 3 个字符的匹配
			return nil, fmt.Errorf("pattern length cannot be small than blockSize %v", blockSize)
		}
		minSubPatternSize = min(minSubPatternSize, len(p))
	}

	tableSize := getTableSize(patternCount, minSubPatternSize)
	shiftTable := make([]int, tableSize)
	for i := range shiftTable {
		shiftTable[i] = minSubPatternSize - blockSize + 1
	}
	prefixHashTable := make([]prefixHashItem, tableSize)

	for i := range patterns {
		pattern := patterns[i]
		// assert(blockSize <= len(pattern))
		for j := minSubPatternSize; j >= blockSize; j-- {
			h := hashBlock2(pattern, j) % uint32(tableSize)
			shiftTable[h] = min(minSubPatternSize-j, shiftTable[h])
		}
		// 等价与 for 循环的首次
		h := hashBlock2(pattern, minSubPatternSize) % uint32(tableSize)
		ph := hashBlock2(pattern, blockSize) % uint32(tableSize)
		prefixHashTable[h] = append(prefixHashTable[h], prefixHashPair{
			PrefixHash: ph,
			Index:      i,
		})
	}

	w := &WuManber{}
	w.Patterns = make([][]byte, 0, patternCount)
	w.Patterns = append(w.Patterns, patterns...)
	w.minSubPatternSize = minSubPatternSize
	w.tableSize = tableSize
	w.shiftTable = shiftTable
	w.prefixHashTable = prefixHashTable
	return w, nil
}

func (w *WuManber) Search(text []byte, matchedCallback func(needle []byte, needleIndex, textIndex int) bool) {
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
