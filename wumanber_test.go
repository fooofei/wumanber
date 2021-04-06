package wumanber

import (
	"fmt"
	"sort"
	"testing"

	"gotest.tools/v3/assert"
)

func stringToBytes(ss []string) [][]byte {
	bb := make([][]byte, 0)
	for _, s := range ss {
		bb = append(bb, []byte(s))
	}
	return bb
}

func TestWuManber_One(t *testing.T) {
	w, err := New(stringToBytes([]string{
		"a",
		"b",
		"c",
		"ef",
	}))
	assert.NilError(t, err)

	hits := make([]int, 0)
	w.Search([]byte("abcdefghcijefg"), func(needle []byte, needleIndex, textIndex int) bool {
		hits = append(hits, textIndex)
		return true
	})

	sort.Ints(hits)
	assert.DeepEqual(t, hits, []int{0, 1, 2, 4, 8, 11})
}

func TestWuManber_Search(t *testing.T) {
	w, err := New(stringToBytes([]string{
		"abc",
		"efg",
	}))
	assert.NilError(t, err)

	hits := make([]int, 0)
	w.Search([]byte("abcdefghijefg"), func(needle []byte, needleIndex, textIndex int) bool {
		hits = append(hits, textIndex)
		return true
	})

	assert.DeepEqual(t, hits, []int{0, 4, 10})
}

func TestNoData(t *testing.T) {
	w, err := New(stringToBytes([]string{
		"foo", "baz", "bar",
	}))
	assert.NilError(t, err)

	hits := 0
	w.Search([]byte(""), func(needle []byte, needleIndex, textIndex int) bool {
		fmt.Printf("searched '%s' at %v\n", needle, textIndex)
		hits += 1
		return true
	})
	assert.Equal(t, hits, 0)
}

func TestSuffixes(t *testing.T) {
	w, err := New(stringToBytes([]string{"Superman", "uperman", "perman", "erman"}))
	assert.NilError(t, err)
	needleIndexs := make([]int, 0)
	textIndexs := make([]int, 0)
	w.Search([]byte("The Man Of Steel: Superman"), func(needle []byte, needleIndex, textIndex int) bool {
		needleIndexs = append(needleIndexs, needleIndex)
		textIndexs = append(textIndexs, textIndex)
		return true
	})
	sort.Ints(needleIndexs)
	assert.DeepEqual(t, needleIndexs, []int{0, 1, 2, 3})
	assert.DeepEqual(t, textIndexs, []int{18, 19, 20, 21})
}

func TestPrefixes(t *testing.T) {
	w, err := New(stringToBytes([]string{"Superman", "Superma", "Superm", "Super"}))
	assert.NilError(t, err)
	needleIndexs := make([]int, 0)
	textIndexs := make([]int, 0)
	w.Search([]byte("The Man Of Steel: Superman"), func(needle []byte, needleIndex, textIndex int) bool {
		needleIndexs = append(needleIndexs, needleIndex)
		textIndexs = append(textIndexs, textIndex)
		return true
	})
	sort.Ints(needleIndexs)
	assert.DeepEqual(t, needleIndexs, []int{0, 1, 2, 3})
	assert.DeepEqual(t, textIndexs, []int{18, 18, 18, 18})
}

func TestOverlappingPatterns(t *testing.T) {
	w, err := New(stringToBytes([]string{"Man ", "n Of", "Of S"}))
	assert.NilError(t, err)
	needleIndexs := make([]int, 0)
	textIndexs := make([]int, 0)
	w.Search([]byte("The Man Of Steel"), func(needle []byte, needleIndex, textIndex int) bool {
		needleIndexs = append(needleIndexs, needleIndex)
		textIndexs = append(textIndexs, textIndex)
		return true
	})
	sort.Ints(needleIndexs)
	assert.DeepEqual(t, needleIndexs, []int{0, 1, 2})
	assert.DeepEqual(t, textIndexs, []int{4, 6, 8})
}

func TestNothingMatches(t *testing.T) {
	w, err := New(stringToBytes([]string{"baz", "bar", "foo"}))
	assert.NilError(t, err)
	needleIndexs := make([]int, 0)
	textIndexs := make([]int, 0)
	w.Search([]byte("A Man A Plan A Canal: Panama, which Man Planned The Canal"), func(needle []byte, needleIndex, textIndex int) bool {
		needleIndexs = append(needleIndexs, needleIndex)
		textIndexs = append(textIndexs, textIndex)
		return true
	})
	sort.Ints(needleIndexs)
	assert.DeepEqual(t, needleIndexs, []int{})
	assert.DeepEqual(t, textIndexs, []int{})
}

var haystack = []byte("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36")
var dictionary = []string{"Mozilla", "Mac", "Macintosh", "Safari", "Sausage"}

func BenchmarkMatchWorks(b *testing.B) {
	w, err := New(stringToBytes(dictionary))
	assert.NilError(b, err)
	for i := 0; i < b.N; i++ {
		w.Search(haystack, func(needle []byte, needleIndex, textIndex int) bool {
			return true
		})
	}
}
