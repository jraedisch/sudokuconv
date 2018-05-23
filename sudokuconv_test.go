package sudokuconv_test

import (
	"reflect"
	"testing"

	"github.com/jraedisch/sudokuconv"
)

var boards = [][9][9]int{
	[9][9]int{
		{9, 8, 7, 6, 5, 4, 3, 2, 1},
		{6, 5, 4, 3, 2, 1, 9, 8, 7},
		{3, 2, 1, 9, 8, 7, 6, 5, 4},
		{8, 9, 6, 7, 4, 5, 2, 1, 3},
		{7, 4, 5, 2, 1, 3, 8, 9, 6},
		{2, 1, 3, 8, 9, 6, 7, 4, 5},
		{5, 7, 9, 4, 6, 8, 1, 3, 2},
		{4, 6, 8, 1, 3, 2, 5, 7, 9},
		{1, 3, 2, 5, 7, 9, 4, 6, 8},
	},
}

func TestToBytes(t *testing.T) {
	for i, expected := range boards {
		bs, err := sudokuconv.ToBytes(expected)
		if err != nil {
			t.Errorf("unexpected error encoding #%d: %v\n", i, err)
		}
		actual, err := sudokuconv.FromBytes(bs)
		if err != nil {
			t.Errorf("unexpected error decoding #%d: %v\n", i, err)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("decoded board does not match original #%d:\n%v\n%v\n", i, expected, actual)
		}
	}
}
