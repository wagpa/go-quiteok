package qoi

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"log"
)

// Decode Reads all bytes from the reader and tries to decode an image with the QuiteOk image format from it.
func Decode(reader io.Reader) (*QuiteOkImage, error) {
	// TODO read continuously until eof
	data, readErr := io.ReadAll(reader)
	if readErr != nil {
		return nil, readErr
	}

	img := &QuiteOkImage{}

	if err := decodeHeader(data[:14], &img.header); err != nil {
		return nil, err
	}

	size := uint64(img.header.width) * uint64(img.header.height)
	pixels := make([]color.NRGBA, size)
	if err := decodePixels(data[14:], &pixels); err != nil {
		return nil, err
	}
	if len(pixels) != cap(pixels) {
		return nil, fmt.Errorf("invalid number of pixels decoded, expected %d, actual %d", cap(pixels), len(pixels))
	}

	return img, nil
}

// Reads the given bytes, decodes them and writes the decoded to the qoi.QuiteOkHeader. It expects exactly 14 bytes.
func decodeHeader(data []byte, header *QuiteOkHeader) error {
	if len(data) != 14 {
		return fmt.Errorf("invalid header size, expected length 14, actual is %d", len(data))
	}

	magic := string(data[0:4])
	if magic != Magic {
		log.Println("got magic", magic)
		return fmt.Errorf("invalid magic bytes in header, expected '%s', actual is '%s'", Magic, magic)
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
	for dataIndex < len(data)-8 { // TODO read until eof instead of length based

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
				return fmt.Errorf("invalid run length decoded, expected in bounds 1 and 62, actual %d", run)
			}
			for i := 0; i < run; i++ {
				(*pixels)[pixelIndex+i] = pixel
			}

			pixelIndex += run
			dataIndex += 1
			continue
		}

		return fmt.Errorf("unknwon opcode read, read op8: %b op2: %b", op8, op2)
	}

	// validate file end sequence
	if len(data[dataIndex:]) != 8 || *(*[8]byte)(data[dataIndex:]) != eof {
		return fmt.Errorf("invalid eof bytes received")
	}
	return nil
}
