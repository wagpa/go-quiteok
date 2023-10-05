// Package qoi provides Decode and Encode for the QuiteOk image format (.qoi).
// Use image.RegisterFormat to register the image format:
//
//	image.RegisterFormat(qoi.Format, qoi.Magic, qoi.Decode, qoi.DecodeConfig)
package qoi

import (
	"errors"
	"image/color"
)

const (
	// Magic is the magic code used for files of the QuiteOk image format.
	Magic = "qoif"
	// Format is the used file format.
	Format = "qoi"
	// opRgb is the op code for rgb encoding.
	opRgb = byte(0b11111110)
	// opRgba is the op code for rgba encoding.
	opRgba = byte(0b11111111)
	// opIndex is the op code for index encoding (referencing previously seen colors).
	opIndex = byte(0b00000000)
	// opDiff is the op code for diff encoding (describing the difference to the previous color).
	opDiff = byte(0b01000000)
	// opLuma is the op code for luma encoding (describing the difference to the previous color).
	opLuma = byte(0b10000000)
	// opRun is the op code for rgba encoding (describing a run of matching colors to the previous color).
	opRun = byte(0b11000000)
	// op2Mask is the mask for 2-bit op codes.
	op2Mask = byte(0b11000000)
)

var (
	// ErrInvalidMagic describes an invalid Magic read for the image.Config.
	ErrInvalidMagic = errors.New("invalid magic")
	// ErrInvalidEOF describes an invalid stream of bytes read that is used to indicate the end of the encoding.
	ErrInvalidEOF = errors.New("invalid EOF")
	// ErrInvalidRunLength describes an invalid run length read.
	ErrInvalidRunLength = errors.New("invalid run length")
	// eof is the end of file code used by files of the QuiteOk image format.
	eof = [...]byte{0, 0, 0, 0, 0, 0, 0, 1}
	// startPixel is the (by contract) color value used as "last seen" for the first iteration.
	startPixel = [...]byte{0, 0, 0, 255}
	// zeroPixel is a zero color used to initialize the "previously seen" color cache.
	zeroPixel = [...]byte{0, 0, 0, 0}
)

func hashPix(pix *[4]uint8) byte {
	return (pix[0]*3 + pix[1]*5 + pix[2]*7 + pix[3]*11) % 64
}

func hashColor(col color.NRGBA) byte {
	return (col.R*3 + col.G*5 + col.B*7 + col.A*11) % 64
}
