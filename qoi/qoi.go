package qoi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
)

// image definition

// QuiteOkImage is the image type for the QuiteOk image format. It implements the image.Image interface.
type QuiteOkImage struct {
	header QuiteOkHeader
	pixels []color.NRGBA
}

// QuiteOkHeader is the header data of a QuiteOk image format. See qoi.QuiteOkImage for usage.
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

// A List of op-codes used in the file. They specify how the bytes are encoded.
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

// Decode Reads all bytes from the reader and tries to decode an image with the QuiteOk image format from it.
func Decode(reader io.Reader) (QuiteOkImage, error) {
	data, readErr := io.ReadAll(reader)
	if readErr != nil {
		return QuiteOkImage{}, readErr
	}

	var header QuiteOkHeader
	if err := decodeHeader(data[:14], &header); err != nil {
		return QuiteOkImage{}, err
	}
	fmt.Printf("decoded header %+v\n", header)

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

// Reads the given bytes, decodes them and writes the decoded to the qoi.QuiteOkHeader. It expects exactly 14 bytes.
func decodeHeader(data []byte, header *QuiteOkHeader) error {
	if len(data) != 14 {
		return errors.New("invalid header size")
	}

	magic := string(data[0:4])
	if magic != Magic {
		log.Println("got magic", magic)
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

// Reads the given bytes and decodes them to given pixels slice.
// It expects the pixel slice to already be initialized with at least the size of the pixels to be decoded.
// The bytes have to end with the QuiteOk image format end of file byte sequence.
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
			seen[hashColor(&pixel)] = pixel
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
			seen[hashColor(&pixel)] = pixel
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
			seen[hashColor(&pixel)] = pixel
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
			seen[hashColor(&pixel)] = pixel
			pixelIndex += 1
			dataIndex += 2
			continue
		}

		if op2 == OpRun {
			run := int(data[dataIndex]&0b00111111) + 1
			if run > 62 || run < 1 {
				return errors.New("run length out of bounds")
			}
			for i := 0; i < run; i++ {
				(*pixels)[pixelIndex+i] = pixel
			}

			pixelIndex += run
			dataIndex += 1
			continue
		}

		return errors.New("unknown op code")
	}

	// validate file end sequence
	if len(data[dataIndex:]) != 8 || *(*[8]byte)(data[dataIndex:]) != eof {
		log.Println("eof", data[dataIndex-10:])
		return errors.New("invalid eof")
	}

	if pixelIndex != cap(*pixels) {
		log.Println("pixels: expected", cap(*pixels), "actual", pixelIndex)
		return errors.New("invalid number of pixels decoded")
	}

	return nil
}

// Encode

// Calculates the (directed) distance between two numbers in a wrapped room (modulo room).
//
//	x := modDist(a, b, m)
//	(a + x + m) % m == b
func modDist(a int, b int, m int) int {
	// make sure, that a <= b
	v := 1
	if a > b {
		b, a = a, b
		v = -1
	}

	ab := b - a
	ba := (a + m) - b

	if ab < ba {
		return v * ab
	} else {
		return v * -1 * ba
	}
}

// Encode encodes a given image to the QuiteOk image format and writes the encoded bytes to the writer.
func Encode(writer io.Writer, image image.Image) error {
	// prerequisite
	index := 0 // pixel index in image
	prev := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}
	var data []byte

	// TODO offsets? image.Bounds().Min.Y
	var curr color.NRGBA
	xy := func(i int) (int, int) {
		x := i % image.Bounds().Max.X
		y := i / image.Bounds().Max.X
		return x, y
	}

	// read pixels & generate data
	size := image.Bounds().Dx() * image.Bounds().Dy()
	for index < size {
		x, y := xy(index)
		curr = color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)

		// OpRun
		if prev == curr {
			run := 1
			for run < 62 && run+index < size {
				nx, ny := xy(index + run)
				next := color.NRGBAModel.Convert(image.At(nx, ny)).(color.NRGBA)
				if next != prev {
					break
				}
				run++
			}

			data = append(
				data,
				OpRun|byte(run-1),
			)
			seen[hashColor(&curr)] = curr
			index += run

			continue
		}

		// OpIndex
		if hash := hashColor(&curr); seen[hash] == curr {
			data = append(
				data,
				OpIndex|byte(hash),
			)
			prev = curr
			index += 1

			continue
		}

		// OpRgba
		if prev.A != curr.A {
			data = append(
				data,
				OpRgba,
				curr.R,
				curr.G,
				curr.B,
				curr.A,
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		// alpha channel is the same

		dg := modDist(int(prev.G), int(curr.G), 256)
		dr := modDist(int(prev.R), int(curr.R), 256)
		db := modDist(int(prev.B), int(curr.B), 256)

		// OpDiff
		if (-2 <= dr && dr <= 1) && (-2 <= dg && dg <= 1) && (-2 <= db && db <= 1) {
			data = append(
				data,
				OpDiff|
					byte((dr+2)<<4)|
					byte((dg+2)<<2)|
					byte((db+2)<<0),
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		drDg := dr - dg
		dbDg := db - dg

		// OpLuma
		if (-32 <= dg && dg <= 31) && (-8 <= drDg && drDg <= 7) && (-8 <= dbDg && dbDg <= 7) {
			data = append(
				data,
				OpLuma|byte(dg+32),
				byte((drDg+8)<<4)|byte(dbDg+8),
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		// OpRgb
		if true {
			data = append(
				data,
				OpRgb,
				curr.R,
				curr.G,
				curr.B,
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}
	}

	// create file bytes
	var file []byte

	// add magic
	file = append(
		file,
		Magic...,
	)
	// add size
	file = binary.BigEndian.AppendUint32(file, uint32(image.Bounds().Dx()))
	file = binary.BigEndian.AppendUint32(file, uint32(image.Bounds().Dy()))
	// add format
	file = append(
		file,
		uint8(4),
		uint8(0),
	)
	// add pixels
	file = append(
		file,
		data...,
	)
	// add eof indicator
	file = append(
		file,
		eof[:]...,
	)

	// write to file
	if _, err := writer.Write(file); err != nil {
		return err
	}
	return nil
}
