package qoi

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
)

// image definition

type QuiteOkImage struct {
	header QuiteOkHeader
	pixels []color.NRGBA
}

type QuiteOkHeader struct {
	Width      uint32
	Height     uint32
	Channels   uint8
	Colorspace uint8
}

func (img QuiteOkImage) ColorModel() color.Model {
	// TODO implement me!
	return color.NRGBAModel
}

func (img QuiteOkImage) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{},
		Max: image.Point{
			X: int(img.header.Width),
			Y: int(img.header.Height),
		},
	}
}

func (img QuiteOkImage) At(x, y int) color.Color {
	return img.pixels[x+y*int(img.header.Width)]
}

// decoding

// the op-codes used by the qoi file format
const (
	OpRgb   = byte(0b11111110)
	OpRgba  = byte(0b11111111)
	OpIndex = byte(0b00000000)
	OpDiff  = byte(0b01000000)
	OpLuma  = byte(0b10000000)
	OpRun   = byte(0b11000000)
)

const Magic = "qoif"

var eof = [...]byte{0, 0, 0, 0, 0, 0, 0, 1}

func genIndex(pixel color.NRGBA) int {
	return (int(pixel.R)*3 + int(pixel.G)*5 + int(pixel.B)*7 + int(pixel.A)*11) % 64
}

func Decode(reader io.Reader) (QuiteOkImage, error) {
	data, readErr := io.ReadAll(reader)
	if readErr != nil {
		return QuiteOkImage{}, readErr
	}

	var header QuiteOkHeader
	if err := decodeHeader(data[:14], &header); err != nil {
		return QuiteOkImage{}, err
	}

	size := uint64(header.Width) * uint64(header.Height)
	pixels := make([]color.NRGBA, size)
	if err := decodePixels(data[14:], &pixels); err != nil {
		return QuiteOkImage{}, err
	}

	return QuiteOkImage{
		header: header,
		pixels: pixels,
	}, nil
}

func decodeHeader(data []byte, header *QuiteOkHeader) error {
	if len(data) != 14 {
		return errors.New("invalid header size")
	}

	magic := string(data[0:4])
	if magic != Magic {
		return errors.New("invalid header magic")
	}

	*header = QuiteOkHeader{
		Width:      binary.BigEndian.Uint32(data[4:8]),
		Height:     binary.BigEndian.Uint32(data[8:12]),
		Channels:   data[12],
		Colorspace: data[13],
	}
	return nil
}

func decodePixels(data []byte, pixels *[]color.NRGBA) error {
	// prerequisite
	cursor := 0
	index := 0
	prev := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}

	// read pixels
	for cursor < len(data)-8 {
		op8 := data[cursor] & 0b11111111 // 8bit tag
		op2 := data[cursor] & 0b11000000 // 2bit tag

		if op8 == OpRgb {
			prev = color.NRGBA{
				R: data[1],
				G: data[2],
				B: data[3],
				A: prev.A,
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 4
		} else if op8 == OpRgba {
			prev = color.NRGBA{
				R: data[1],
				G: data[2],
				B: data[3],
				A: data[4],
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 5
		} else if op2 == OpIndex {
			pixelIndex := data[0] & 0b00111111
			prev = seen[pixelIndex]
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 1
		} else if op2 == OpDiff {
			dr := int((data[0]&0b00110000)>>4) - 2
			dg := int((data[0]&0b00001100)>>2) - 2
			db := int((data[0]&0b00000011)>>0) - 2
			prev = color.NRGBA{
				R: uint8((int(prev.R) + dr) % 256),
				G: uint8((int(prev.G) + dg) % 256),
				B: uint8((int(prev.B) + db) % 256),
				A: prev.A,
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 1
		} else if op2 == OpLuma {
			dg := int((data[0]&0b00111111)>>0) - 32
			dr := int((data[1]&0b11110000)>>4) - 8 + dg
			db := int((data[1]&0b00001111)>>0) - 8 + dg
			prev = color.NRGBA{
				R: uint8((int(prev.R) + dr) % 256),
				G: uint8((int(prev.G) + dg) % 256),
				B: uint8((int(prev.B) + db) % 256),
				A: prev.A,
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 2
		} else if op2 == OpRun {
			rep := int(data[0]&0b00111111) + 1
			for i := 0; i < rep; i++ {
				(*pixels)[index+i] = prev
			}
			index += rep
			cursor += 1
		} else {
			return errors.New("unknown op code")
		}
	}

	// validate file end sequence
	if len(data[cursor:]) != 8 || *(*[8]byte)(data[cursor:]) != eof {
		return errors.New("invalid eof")
	}

	return nil
}
