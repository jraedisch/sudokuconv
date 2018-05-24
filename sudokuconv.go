// Package sudokuconv contains helpers to convert solved 9x9 sudokus to compact byte slices.
package sudokuconv

import (
	"math/bits"
	"sort"

	"github.com/pkg/errors"
)

var (
	// encMasks extract the 3 bits of ints < 8
	encMasks = [3]uint8{4, 2, 1}
	// decMasks extract each bit of a byte/uint8
	decMasks = [8]uint8{128, 64, 32, 16, 8, 4, 2, 1}
)

// ToBytes converts a 9x9 sudoku board into a compact bit representation.
// The returned byte slice contains 4 bits for the row where the 9 is in the last column.
// Then follow 3 bits for each of the other eight columns containing 9s.
// Then the other values are converted and appended as 3 bits each.
// For this, 1-8 are converted to 0-7.
// The last row and column are left out since they can trivially be computed.
// An error is returned iff the provided board is not correctly solved.
//
// TODO: Leave out one value of each of the four complete subgrids.
func ToBytes(board [9][9]int) ([]byte, error) {
	if !validate(board) {
		return nil, errors.New("board not solved correctly")
	}
	im := prepare(board)

	bytes := [25]byte{}
	bytes[0] = im.RowWith9Last << 4

	bitIdx := uint(4)
	var shiftL uint8 = 3
	for _, v := range append(im.NineIndices, im.Values...) {
		for shiftR, mask := range encMasks {
			bytes[bitIdx/8] = bytes[bitIdx/8] + (v&mask)>>uint8(2-shiftR)<<shiftL
			shiftL = (shiftL - 1) % 8
			bitIdx++
		}
	}

	return bytes[:], nil
}

// FromBytes reverts ToBytes.
// An error is returned iff the provided bytes are malformed.
func FromBytes(bytes []byte) ([9][9]int, error) {
	if len(bytes) < 9 {
		return [9][9]int{}, errors.New("not enough bytes")
	}
	var initialMaskIdx uint = 4
	var digitIdx uint
	values := []uint8{}
	var currentValue uint8
	for _, b := range bytes {
		for maskIdx := initialMaskIdx; maskIdx < 8; maskIdx++ {
			currentValue = currentValue + (b&decMasks[maskIdx])>>(7-maskIdx)<<(2-digitIdx)
			digitIdx = (digitIdx + 1) % 3
			if digitIdx == 0 {
				values = append(values, currentValue)
				currentValue = 0
			}
		}
		initialMaskIdx = 0
	}
	im := &intermediate{
		RowWith9Last: bytes[0] >> 4,
		NineIndices:  values[:8],
		Values:       values[8:]}
	board, err := im.toBoard()
	if err != nil {
		return [9][9]int{}, errors.Wrap(err, "incomplete bytes")
	}
	board = solve(board)
	if !validate(board) {
		return [9][9]int{}, errors.New("bytes lead to incorrect board")
	}
	return board, nil
}

type intermediate struct {
	RowWith9Last uint8
	NineIndices  []uint8
	Values       []uint8
}

func (im *intermediate) toBoard() ([9][9]int, error) {
	board := [9][9]int{}
	board[im.RowWith9Last][8] = 9
	for rowIdx, colIdx := range im.NineIndices {
		if rowIdx >= int(im.RowWith9Last) {
			board[rowIdx+1][colIdx] = 9
		} else {
			board[rowIdx][colIdx] = 9
		}
	}
	valIdx := 0
	valLen := len(im.Values)
	for rowIdx, row := range board {
		for colIdx, val := range row {
			if rowIdx < 8 && colIdx < 8 && val != 9 {
				if valIdx >= valLen {
					return [9][9]int{}, errors.New("not enough values")
				}
				board[rowIdx][colIdx] = int(im.Values[valIdx]) + 1
				valIdx++
			}
		}
	}
	return board, nil
}

func prepare(board [9][9]int) *intermediate {
	im := intermediate{}

	for rowIdx, row := range board {
		for colIdx, val := range row {
			if val == 9 && colIdx == 8 {
				im.RowWith9Last = uint8(rowIdx)
			} else if val == 9 {
				im.NineIndices = append(im.NineIndices, uint8(colIdx))
			} else if rowIdx < 8 && colIdx < 8 {
				// 1 is subtracted to have values from 0-7
				im.Values = append(im.Values, uint8(val-1))
			}
		}
	}

	return &im
}

func validate(board [9][9]int) bool {
	for _, row := range board {
		if !validateGroup(row) {
			return false
		}
	}
	for colIdx := 0; colIdx < 9; colIdx++ {
		if !validateGroup(extractCol(board, colIdx)) {
			return false
		}
	}
	return true
}

func validateGroup(group [9]int) bool {
	sorted := group[:]
	sort.Ints(sorted)
	for idx, val := range sorted {
		if val != idx+1 {
			return false
		}
	}
	return true
}

func solve(board [9][9]int) [9][9]int {
	for rowIdx, row := range board {
		if rowIdx < 8 {
			board[rowIdx][8] = lastMissing(row)
		}
	}
	for colIdx := 0; colIdx < 9; colIdx++ {
		board[8][colIdx] = lastMissing(extractCol(board, colIdx))
	}
	return board
}

func lastMissing(group [9]int) int {
	var taken uint8
	for _, val := range group {
		taken = taken + 1<<(uint(val)-1)
	}
	return bits.TrailingZeros8(taken^255) + 1
}

func extractCol(board [9][9]int, idx int) [9]int {
	return [9]int{
		board[0][idx],
		board[1][idx],
		board[2][idx],
		board[3][idx],
		board[4][idx],
		board[5][idx],
		board[6][idx],
		board[7][idx],
		board[8][idx],
	}
}
