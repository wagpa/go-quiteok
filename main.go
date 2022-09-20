package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	name := "./tmp/dice.qoi"
	file := DecodeFile(name)
	fmt.Printf("read file %+v\n", file.Header)

	img := image.NewRGBA(image.Rect(0, 0, int(file.Header.Width), int(file.Header.Height)))
	for index, pixel := range file.Pixels {
		img.Set(index%int(file.Header.Width), index/int(file.Header.Width), pixel)
	}
	f, _ := os.Create(name + ".png")
	png.Encode(f, img)

	// tests
	if len(file.Pixels) != int(file.Header.Width)*int(file.Header.Height) {
		log.Print("invalid pixels ", len(file.Pixels))
	}
}

type QuiteOkHeader struct {
	Magic      [4]byte // "qoif" -> 113, 111, 105, 102
	Width      uint32
	Height     uint32
	Channels   uint8
	Colorspace uint8
}

type QuiteOkFile struct {
	Header QuiteOkHeader
	Pixels []color.RGBA
}

const QoiOpRgb = byte(0b11111110)
const QoiOpRgba = byte(0b11111111)
const QoiOpIndex = byte(0b00000000)
const QoiOpDiff = byte(0b01000000)
const QoiOpLuma = byte(0b10000000)
const QoiOpRun = byte(0b11000000)

func DecodeFile(name string) QuiteOkFile {
	data, readErr := os.ReadFile(name)
	if readErr != nil {
		log.Fatal("read file error: ", readErr)
	}

	return QuiteOkFile{
		Header: DecodeHeader(data[:14]),
		Pixels: DecodePixels(data[14:]),
	}
}

func DecodeHeaderFile(file *os.File) QuiteOkHeader {
	var header QuiteOkHeader
	err := binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		log.Fatal("read file error: ", err)
	}
	if header.Magic != [4]byte{113, 111, 105, 102} {
		log.Fatal("invalid file format: ", "magic invalid")
	}
	return header
}

func DecodeHeader(data []byte) QuiteOkHeader {
	return QuiteOkHeader{
		Magic:      *(*[4]byte)(data[0:4]),
		Width:      binary.BigEndian.Uint32(data[4:8]),
		Height:     binary.BigEndian.Uint32(data[8:12]),
		Channels:   data[12],
		Colorspace: data[13],
	}
}

func DecodePixels(data []byte) []color.RGBA {
	var cursor int
	var pixels []color.RGBA

	prev := color.RGBA{A: 255}
	seen := [64]color.RGBA{}

	for cursor < len(data)-8 {
		delta, decoded := decodeBlock(data[cursor], data[cursor:cursor+5], &prev, &seen)

		lastDecoded := decoded[len(decoded)-1]

		pixels = append(pixels, decoded...)
		seen[genIndex(lastDecoded)] = lastDecoded
		prev = lastDecoded
		cursor += delta
	}

	log.Print("last bits ", data[cursor:])

	return pixels
}

func decodeBlock(op byte, data []byte, prev *color.RGBA, seen *[64]color.RGBA) (int, []color.RGBA) {
	op8 := op & 0b11111111 // 8bit tag
	op2 := op & 0b11000000 // 2bit tag

	if op8 == QoiOpRgb {
		return 4, []color.RGBA{{
			R: data[1],
			G: data[2],
			B: data[3],
			A: 255,
		}}
	} else if op8 == QoiOpRgba {
		return 5, []color.RGBA{{
			R: data[1],
			G: data[2],
			B: data[3],
			A: data[4],
		}}
	} else if op2 == QoiOpIndex {
		pixelIndex := data[0] & 0b00111111
		return 1, []color.RGBA{seen[pixelIndex]}
	} else if op2 == QoiOpDiff {
		dr := uint((data[0]&0b00110000)>>4) - 2
		dg := uint((data[0]&0b00001100)>>2) - 2
		db := uint((data[0]&0b00000011)>>0) - 2
		return 1, []color.RGBA{{
			R: byte((uint(prev.R) + dr + 255) % 255),
			G: byte((uint(prev.G) + dg + 255) % 255),
			B: byte((uint(prev.B) + db + 255) % 255),
			A: prev.A,
		}}
	} else if op2 == QoiOpLuma {
		dg := uint((data[0]&0b00111111)>>0) - 32
		dr := uint((data[1]&0b11110000)>>4) - 8 + dg
		db := uint((data[1]&0b00001111)>>0) - 8 + dg
		return 2, []color.RGBA{{
			R: byte((uint(prev.R) + dr + 255) % 255),
			G: byte((uint(prev.G) + dg + 255) % 255),
			B: byte((uint(prev.B) + db + 255) % 255),
			A: prev.A,
		}}
	} else if op2 == QoiOpRun {
		rep := uint(data[0]&0b00111111) + 1
		pixels := make([]color.RGBA, rep)
		for i := range pixels {
			pixels[i] = *prev
		}
		return 1, pixels
	}

	log.Fatal("unknown op code")
	return 1, []color.RGBA{}
}

func genIndex(pixel color.RGBA) uint8 {
	return (pixel.R*3 + pixel.G*5 + pixel.B*7 + pixel.A*11) % 64
}
