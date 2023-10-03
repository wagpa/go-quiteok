package qoi

import "errors"

const (
	OpRgb   = byte(0b11111110)
	OpRgba  = byte(0b11111111)
	OpIndex = byte(0b00000000)
	OpDiff  = byte(0b01000000)
	OpLuma  = byte(0b10000000)
	OpRun   = byte(0b11000000)
	// OpMask is the mask for 2-bit op codes
	OpMask = 0b11000000
	// Magic is the magic code used for files of the QuiteOk image format.
	Magic = "qoif"
)

var (
	// eof is the end of file code used by files of the QuiteOk image format
	eof        = [...]byte{0, 0, 0, 0, 0, 0, 0, 1}
	startPixel = [4]byte{0, 0, 0, 255}
	zeroPixel  = [4]byte{0, 0, 0, 0}
)

var (
	ErrInvalidMagic     = errors.New("invalid magic")
	ErrInvalidEOF       = errors.New("invalid EOF")
	ErrInvalidRunLength = errors.New("invalid run length: must be between 1 and 62")
)

func hashColor(pix *[4]uint8) byte {
	return (pix[0]*3 + pix[1]*5 + pix[2]*7 + pix[3]*11) % 64
}
