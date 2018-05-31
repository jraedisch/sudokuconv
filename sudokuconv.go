// Package sudokuconv contains helpers to convert solved 9x9 sudokus to compact byte slices.
package sudokuconv

import (
	"math"
	"math/bits"
	"sort"

	"github.com/pkg/errors"
)

var (
	// symbolMasks extract 3 bits a symbol (must be zero based and <8)
	symbolMasks = [3]uint8{4, 2, 1}
	// bitMasks extract each bit of a byte/uint8
	bitMasks = [8]uint8{128, 64, 32, 16, 8, 4, 2, 1}
)

// ToBytes converts a 9x9 sudoku board into a compact bit representation.
// Size is 23 or 24 bytes depending on where the 9s are.
// The returned byte slice contains 4 bits for the row where the 9 is in the last column.
// Then follow 3 bits for each of the other eight columns containing 9s.
// Then the other symbols are converted and appended as 3 bits each.
// For this, 1-8 are converted to 0-7.
// The last row and column are left out since they can trivially be computed.
// An error is returned iff the provided board is not correctly solved.
func ToBytes(board [9][9]int) ([]byte, error) {
	if !validate(board) {
		return nil, errors.New("board not solved correctly")
	}

	im := toIntermediate(board)
	bytes := [24]byte{}
	bitIdx := uint(4)
	bytes[0] = im.RowWith9Last << bitIdx

	for _, v := range append(im.NineIndices, im.OtherSymbols...) {
		for idxInSymbol, mask := range symbolMasks {
			idxInByte := 7 - bitIdx%8
			bytes[bitIdx/8] = bytes[bitIdx/8] + (v&mask)>>uint8(2-idxInSymbol)<<idxInByte
			bitIdx++
		}
	}

	return bytes[:byteSize(bitIdx)], nil
}

func byteSize(bitSize uint) int {
	return int(math.Ceil(float64(bitSize) / 8))
}

// FromBytes converts bytes (see ToBytes) back to board.
// An error is returned iff the provided bytes are malformed.
func FromBytes(bytes []byte) ([9][9]int, error) {
	if len(bytes) < 9 {
		return [9][9]int{}, errors.New("not enough bytes")
	}

	symbols := toSymbols(bytes)
	im := &intermediate{
		RowWith9Last: bytes[0] >> 4,
		NineIndices:  symbols[:8],
		OtherSymbols: symbols[8:]}
	board, err := im.toBoard()
	if err != nil {
		return [9][9]int{}, errors.Wrap(err, "incomplete bytes")
	}

	board = solveNaively(board)
	if !validate(board) {
		return [9][9]int{}, errors.New("bytes lead to incorrect board")
	}

	return board, nil
}

func toSymbols(bytes []byte) []uint8 {
	var initialIdxInByte uint = 4
	var idxInSymbol uint
	symbols := []uint8{}
	var currentValue uint8
	for _, b := range bytes {
		for idxInByte := initialIdxInByte; idxInByte < 8; idxInByte++ {
			bit := b & bitMasks[idxInByte]
			currentValue = currentValue + bit>>(7-idxInByte)<<(2-idxInSymbol)
			idxInSymbol = (idxInSymbol + 1) % 3
			if idxInSymbol == 0 {
				symbols = append(symbols, currentValue)
				currentValue = 0
			}
		}
		initialIdxInByte = 0
	}
	return symbols
}

type intermediate struct {
	RowWith9Last uint8
	NineIndices  []uint8
	OtherSymbols []uint8
}

func (im *intermediate) toBoard() ([9][9]int, error) {
	board := [9][9]int{}
	board = im.fill9s(board)
	return im.fillOtherSymbols(board)
}

func (im *intermediate) fill9s(board [9][9]int) [9][9]int {
	board[im.RowWith9Last][8] = 9
	for rowIdx, colIdx := range im.NineIndices {
		if rowIdx >= int(im.RowWith9Last) {
			board[rowIdx+1][colIdx] = 9
		} else {
			board[rowIdx][colIdx] = 9
		}
	}
	return board
}

// Fill 9s first!
func (im *intermediate) fillOtherSymbols(board [9][9]int) ([9][9]int, error) {
	valIdx := 0
	valLen := len(im.OtherSymbols)
	for rowIdx, row := range board {
		for colIdx, val := range row {
			if includeVal(rowIdx, colIdx, val) {
				if valIdx >= valLen {
					return [9][9]int{}, errors.New("not enough values")
				}
				board[rowIdx][colIdx] = int(im.OtherSymbols[valIdx]) + 1
				valIdx++
			}
		}
	}
	return board, nil
}

func toIntermediate(board [9][9]int) *intermediate {
	im := intermediate{}

	for rowIdx, row := range board {
		for colIdx, val := range row {
			if val == 9 && colIdx == 8 {
				im.RowWith9Last = uint8(rowIdx)
			} else if val == 9 {
				im.NineIndices = append(im.NineIndices, uint8(colIdx))
			} else if includeVal(rowIdx, colIdx, val) {
				// 1 is subtracted to have values from 0-7
				im.OtherSymbols = append(im.OtherSymbols, uint8(val-1))
			}
		}
	}

	return &im
}

func includeVal(rowIdx, colIdx, val int) bool {
	return !firstInBlock(rowIdx, colIdx) && rowIdx < 8 && colIdx < 8 && val != 9
}

func firstInBlock(rowIdx, colIdx int) bool {
	block := 0 + uint(1)<<uint(8-rowIdx) + uint(1)<<uint(8-colIdx)
	return block == 512 || block == 288 || block == 64
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
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			if !validateGroup(extractGrid(board, x, y)) {
				return false
			}
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

func solveNaively(board [9][9]int) [9][9]int {
	solved := solveSubgrids(board)
	solved = solveRows(solved)
	return solveCols(solved)
}

func solveSubgrids(board [9][9]int) [9][9]int {
	grids := [2][2][]int{}
	for rowIdx := 0; rowIdx < 6; rowIdx++ {
		for colIdx := 0; colIdx < 6; colIdx++ {
			grids[rowIdx/3][colIdx/3] = append(grids[rowIdx/3][colIdx/3], board[rowIdx][colIdx])
		}
	}
	for rowIdx, row := range grids {
		for colIdx, grid := range row {
			gridA := [9]int{}
			copy(gridA[:], grid)
			board[rowIdx*3][colIdx*3] = lastMissing(gridA)
		}
	}
	return board
}

// Solve subgrids first!
func solveRows(board [9][9]int) [9][9]int {
	for rowIdx, row := range board {
		if rowIdx < 8 {
			board[rowIdx][8] = lastMissing(row)
		}
	}
	return board
}

// Solve rows first!
func solveCols(board [9][9]int) [9][9]int {
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

func extractGrid(board [9][9]int, x int, y int) [9]int {
	var grid [9]int
	var gridIdx int
	for rowIdx := x * 3; rowIdx < (x+1)*3; rowIdx++ {
		for colIdx := y * 3; colIdx < (y+1)*3; colIdx++ {
			grid[gridIdx] = board[rowIdx][colIdx]
			gridIdx++
		}
	}
	return grid
}
