package qoi

import (
	"image/color"
)

// A List of opcodes used in the file. They specify how the bytes are encoded.
const (
	OpRgb   = byte(0b11111110)
	OpRgba  = byte(0b11111111)
	OpIndex = byte(0b00000000)
	OpDiff  = byte(0b01000000)
	OpLuma  = byte(0b10000000)
	OpRun   = byte(0b11000000)
)

// Magic is the magic code used for files of the QuiteOk image format.
const Magic = "qoif"

// The end of file code used by files of the QuiteOk image format.
var eof = [...]byte{0, 0, 0, 0, 0, 0, 0, 1}

// Generates a hash from the provided color. It is a number between 0 and 63.
func hashColor(pixel *color.NRGBA) int {
	return (int(pixel.R)*3 + int(pixel.G)*5 + int(pixel.B)*7 + int(pixel.A)*11) % 64
}
