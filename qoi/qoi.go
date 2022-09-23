package qoi

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
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

var original image.Image

func Decode(reader io.Reader) (QuiteOkImage, error) {
	data, readErr := io.ReadAll(reader)
	if readErr != nil {
		return QuiteOkImage{}, readErr
	}

	{ // TODO remove after debugging
		originalFile, originalOpenErr := os.Open("./tmp/dice.png")
		if originalOpenErr != nil {
			log.Fatalln(originalOpenErr)
		}

		originalImg, originalDecodeErr := png.Decode(originalFile)
		if originalDecodeErr != nil {
			log.Fatalln(originalDecodeErr)
		}
		original = originalImg
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

		prevCursor := cursor
		prevIndex := index

		op := "None" // TODO remove after testing

		if op8 == OpRgb {
			prev = color.NRGBA{
				R: data[cursor+1],
				G: data[cursor+2],
				B: data[cursor+3],
				A: prev.A,
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 4
			op = "OpRgb"
		} else if op8 == OpRgba {
			prev = color.NRGBA{
				R: data[cursor+1],
				G: data[cursor+2],
				B: data[cursor+3],
				A: data[cursor+4],
			}
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 5
			op = "OpRgba"
		} else if op2 == OpIndex {
			pixelIndex := data[cursor] & 0b00111111
			prev = seen[pixelIndex]
			(*pixels)[index] = prev
			seen[genIndex(prev)] = prev
			index += 1
			cursor += 1
			op = "OpIndex"
		} else if op2 == OpDiff {
			dr := int((data[cursor]&0b00110000)>>4) - 2
			dg := int((data[cursor]&0b00001100)>>2) - 2
			db := int((data[cursor]&0b00000011)>>0) - 2
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
			op = "OpDiff"
		} else if op2 == OpLuma {
			dg := int((data[cursor]&0b00111111)>>0) - 32
			dr := int((data[cursor+1]&0b11110000)>>4) - 8 + dg
			db := int((data[cursor+1]&0b00001111)>>0) - 8 + dg
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
			op = "OpLuma"
		} else if op2 == OpRun {
			rep := int(data[cursor]&0b00111111) + 1
			for i := 0; i < rep; i++ {
				(*pixels)[index+i] = prev
			}
			index += rep
			cursor += 1
			op = "OpRun"
		} else {
			return errors.New("unknown op code")
		}

		log.Println("read data", data[prevCursor:cursor], "to", prev, "op", op, " added ", index-prevIndex)
		{
			x := prevIndex % original.Bounds().Dx()
			y := prevIndex / original.Bounds().Dy()
			p := original.At(x, y)
			if prev != p {
				log.Fatal("different pixels at ", x, y, " expected ", p, " actual ", prev, " from data ", data[prevCursor:cursor], " nrgba ", prev.R, prev.G, prev.B, prev.A)
			}
		}
	}

	// validate file end sequence
	if len(data[cursor:]) != 8 || *(*[8]byte)(data[cursor:]) != eof {
		return errors.New("invalid eof")
	}

	return nil
}

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

// utility

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
