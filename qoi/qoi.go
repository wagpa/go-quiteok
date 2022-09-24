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
	width      uint32
	height     uint32
	channels   uint8
	colorspace uint8
}

func (img QuiteOkImage) ColorModel() color.Model {
	return color.NRGBAModel
}

func (img QuiteOkImage) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{},
		Max: image.Point{
			X: int(img.header.width),
			Y: int(img.header.height),
		},
	}
}

func (img QuiteOkImage) At(x, y int) color.Color {
	return img.pixels[x+y*int(img.header.width)]
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

	size := uint64(header.width) * uint64(header.height)
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
		width:      binary.BigEndian.Uint32(data[4:8]),
		height:     binary.BigEndian.Uint32(data[8:12]),
		channels:   data[12],
		colorspace: data[13],
	}
	return nil
}

func decodePixels(data []byte, pixels *[]color.NRGBA) error {
	// prerequisite
	dataIndex := 0  // pixelIndex in data slice (input)
	pixelIndex := 0 // pixelIndex in pixel slice (output)
	pixel := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}

	// read pixels
	for dataIndex < len(data)-8 {

		op8 := data[dataIndex] & 0b11111111 // 8bit tag

		if op8 == OpRgb {
			pixel = color.NRGBA{
				R: data[dataIndex+1],
				G: data[dataIndex+2],
				B: data[dataIndex+3],
				A: pixel.A,
			}

			(*pixels)[pixelIndex] = pixel
			seen[genIndex(pixel)] = pixel
			pixelIndex += 1
			dataIndex += 4
			continue
		}

		if op8 == OpRgba {
			pixel = color.NRGBA{
				R: data[dataIndex+1],
				G: data[dataIndex+2],
				B: data[dataIndex+3],
				A: data[dataIndex+4],
			}

			(*pixels)[pixelIndex] = pixel
			seen[genIndex(pixel)] = pixel
			pixelIndex += 1
			dataIndex += 5
			continue
		}

		op2 := data[dataIndex] & 0b11000000 // 2bit tag

		if op2 == OpIndex {
			index := data[dataIndex] & 0b00111111
			pixel = seen[index]

			(*pixels)[pixelIndex] = pixel
			pixelIndex += 1
			dataIndex += 1
			continue
		}

		if op2 == OpDiff {
			dr := int((data[dataIndex]&0b00110000)>>4) - 2
			dg := int((data[dataIndex]&0b00001100)>>2) - 2
			db := int((data[dataIndex]&0b00000011)>>0) - 2
			pixel = color.NRGBA{
				R: uint8((int(pixel.R) + dr) % 256),
				G: uint8((int(pixel.G) + dg) % 256),
				B: uint8((int(pixel.B) + db) % 256),
				A: pixel.A,
			}

			(*pixels)[pixelIndex] = pixel
			seen[genIndex(pixel)] = pixel
			pixelIndex += 1
			dataIndex += 1
			continue
		}

		if op2 == OpLuma {
			dg := int((data[dataIndex+0]&0b00111111)>>0) - 32
			dr := int((data[dataIndex+1]&0b11110000)>>4) - 8 + dg
			db := int((data[dataIndex+1]&0b00001111)>>0) - 8 + dg
			pixel = color.NRGBA{
				R: uint8((int(pixel.R) + dr) % 256),
				G: uint8((int(pixel.G) + dg) % 256),
				B: uint8((int(pixel.B) + db) % 256),
				A: pixel.A,
			}

			(*pixels)[pixelIndex] = pixel
			seen[genIndex(pixel)] = pixel
			pixelIndex += 1
			dataIndex += 2
			continue
		}

		if op2 == OpRun {
			rep := int(data[dataIndex]&0b00111111) + 1
			if rep > 62 || rep < 1 {
				return errors.New("run length out of bounds")
			}
			for i := 0; i < rep; i++ {
				(*pixels)[pixelIndex+i] = pixel
			}

			pixelIndex += rep
			dataIndex += 1
			continue
		}

		return errors.New("unknown op code")
	}

	// validate file end sequence
	if len(data[dataIndex:]) != 8 || *(*[8]byte)(data[dataIndex:]) != eof {
		return errors.New("invalid eof")
	}

	if pixelIndex != cap(*pixels) {
		return errors.New("invalid number of pixels decoded")
	}

	return nil
}

// Encode --------------------------------

func Encode(writer io.Writer, image image.Image) error {
	// prerequisite
	cursor := 0 // cursor in data slice
	index := 0  // pixel index in image
	prev := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}
	var data []byte

	// TODO offsets? image.Bounds().Min.Y
	var curr color.NRGBA
	xy := func(i int) (int, int) {
		x := index % image.Bounds().Dx()
		y := index / image.Bounds().Dy()
		return x, y
	}

	// read pixels & generate data
	size := image.Bounds().Dx() * image.Bounds().Dy()
	for index < size {
		x, y := xy(index)
		curr = color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)

		// OpRun
		if prev == curr {
			run := index
			for ; prev == curr && (index-run) < 62; index++ {
				x, y = xy(index)
				curr = color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)
			}
			data[cursor] = OpRun | byte(run)
			cursor += 1
			continue
		}

		dg := int(curr.G) - int(prev.G)
		dr := int(curr.R) - int(prev.R)
		db := int(curr.B) - int(prev.B)

		// OpLuma
		if (dg < 31 && dg > -32) && (dr < 7 && dr > -8) && (db < 7 && db > -8) {
			data[cursor] = OpLuma | byte(dg+32)
			data[cursor] = byte((dr-dg+8)<<4) | byte(db-dg+8)
			seen[genIndex(curr)] = curr
			prev = curr
			cursor += 2
			index += 1
			continue
		}

		// OpDiff
		if (dr < 1 && dr > -2) && (dg < 1 && dg > -2) && (db < 1 && db > -2) {
			data[cursor] = OpDiff | byte(dr+2<<4) | byte(dg+2<<2) | byte(db+2<<0)
			seen[genIndex(curr)] = curr
			prev = curr
			cursor += 1
			index += 1
			continue
		}

		// OpDiff
		if key := genIndex(curr); seen[key] == curr {
			data[cursor] = OpIndex | byte(key)
			prev = curr
			cursor += 1
			index += 1
			continue
		}

		// OpRgb
		if prev.A == curr.A {
			data[cursor] = OpRgb
			data[cursor+1] = curr.R
			data[cursor+2] = curr.G
			data[cursor+3] = curr.B
			prev = curr
			cursor += 4
			index += 1
			continue
		}

		// OpRgba
		if true {
			data[cursor] = OpRgba
			data[cursor+1] = curr.R
			data[cursor+2] = curr.G
			data[cursor+3] = curr.B
			data[cursor+4] = curr.A
			prev = curr
			cursor += 5
			index += 1
			continue
		}
	}

	// generate header
	header := make([]byte, 14)
	header = append(header, Magic...)
	binary.BigEndian.AppendUint32(header, uint32(image.Bounds().Dx()))
	binary.BigEndian.AppendUint32(header, uint32(image.Bounds().Dy()))
	header = append(
		header,
		uint8(4), // TODO get actual #channels
		uint8(0), // TODO get actual colorspace
	)

	// write file
	file := append(header, data...)
	if _, err := writer.Write(file); err != nil {
		return err
	}
	return nil
}
