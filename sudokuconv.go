package sudokuconv

import "math/bits"

func ToBytes(board [9][9]int) ([]byte, error) {
	im := prepare(board)

	bytes := [25]byte{}
	bytes[0] = im.RowWith9Last << 4

	bitIdx := uint(4)
	var shiftL uint8 = 3
	var encMasks = [3]uint8{4, 2, 1}
	for _, v := range append(im.NineIndices, im.Values...) {
		for shiftR, mask := range encMasks {
			bytes[bitIdx/8] = bytes[bitIdx/8] + (v&mask)>>uint8(2-shiftR)<<shiftL
			shiftL = (shiftL - 1) % 8
			bitIdx++
		}
	}

	return bytes[:], nil
}

func FromBytes(bytes []byte) ([9][9]int, error) {
	var initialMaskIdx uint = 4
	var digitIdx uint
	values := []uint8{}
	var currentValue uint8
	var deserMasks = [8]uint8{128, 64, 32, 16, 8, 4, 2, 1}
	for _, b := range bytes {
		for maskIdx := initialMaskIdx; maskIdx < 8; maskIdx++ {
			currentValue = currentValue + (b&deserMasks[maskIdx])>>(7-maskIdx)<<(2-digitIdx)
			digitIdx = (digitIdx + 1) % 3
			if digitIdx == 0 {
				values = append(values, currentValue)
				currentValue = 0
			}
		}
		initialMaskIdx = 0
	}
	b := (&intermediate{
		RowWith9Last: bytes[0] >> 4,
		NineIndices:  values[:8],
		Values:       values[8:]}).ToBoard()
	return solve(b), nil
}

type intermediate struct {
	RowWith9Last uint8
	NineIndices  []uint8
	Values       []uint8
}

func (im *intermediate) ToBoard() [9][9]int {
	b := [9][9]int{}
	b[im.RowWith9Last][8] = 9
	for rowIdx, colIdx := range im.NineIndices {
		if rowIdx >= int(im.RowWith9Last) {
			b[rowIdx+1][colIdx] = 9
		} else {
			b[rowIdx][colIdx] = 9
		}
	}
	valIdx := 0
	for rowIdx, row := range b {
		for colIdx, val := range row {
			if rowIdx < 8 && colIdx < 8 && val != 9 {
				b[rowIdx][colIdx] = int(im.Values[valIdx]) + 1
				valIdx++
			}
		}
	}
	return b
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

func solve(board [9][9]int) [9][9]int {
	for rowIdx, row := range board {
		if rowIdx < 8 {
			board[rowIdx][8] = missing(row)
		}
	}
	for colIdx := 0; colIdx < 9; colIdx++ {
		col := [9]int{
			board[0][colIdx],
			board[1][colIdx],
			board[2][colIdx],
			board[3][colIdx],
			board[4][colIdx],
			board[5][colIdx],
			board[6][colIdx],
			board[7][colIdx],
		}
		board[8][colIdx] = missing(col)
	}
	return board
}

func missing(group [9]int) int {
	var taken uint8
	for _, val := range group {
		taken = taken + 1<<(uint(val)-1)
	}
	return bits.TrailingZeros8(taken^255) + 1
}
