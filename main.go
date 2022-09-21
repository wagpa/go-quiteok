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

var original *image.Image

func main() {
	// get original file for debugging
	oFile, _ := os.Open("./tmp/dice.png")
	oImg, _ := png.Decode(oFile)
	original = &oImg

	// actual decoding
	name := "./tmp/dice.qoi"
	file := DecodeFile(name)
	fmt.Printf("read file %+v\n", file.Header)
	img := image.NewRGBA(image.Rect(0, 0, int(file.Header.Width), int(file.Header.Height)))
	for index, pixel := range file.Pixels {
		x := index % int(file.Header.Width)
		y := index / int(file.Header.Width)
		img.Set(x, y, pixel)
	}

	// write decoded to png
	f, _ := os.Create(name + ".png")
	png.Encode(f, img)
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
	Pixels []color.NRGBA
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

func DecodePixels(data []byte) []color.NRGBA {
	var cursor int
	var pixels []color.NRGBA

	prev := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}

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

func decodeBlock(op byte, data []byte, prev *color.NRGBA, seen *[64]color.NRGBA) (int, []color.NRGBA) {
	op8 := op & 0b11111111 // 8bit tag
	op2 := op & 0b11000000 // 2bit tag

	if op8 == QoiOpRgb {
		return 4, []color.NRGBA{{
			R: data[1],
			G: data[2],
			B: data[3],
			A: 255,
		}}
	} else if op8 == QoiOpRgba {
		return 5, []color.NRGBA{{
			R: data[1],
			G: data[2],
			B: data[3],
			A: data[4],
		}}
	} else if op2 == QoiOpIndex {
		pixelIndex := data[0] & 0b00111111
		return 1, []color.NRGBA{seen[pixelIndex]}
	} else if op2 == QoiOpDiff {
		dr := int((data[0]&0b00110000)>>4) - 2
		dg := int((data[0]&0b00001100)>>2) - 2
		db := int((data[0]&0b00000011)>>0) - 2
		return 1, []color.NRGBA{{
			R: uint8((int(prev.R) + dr) % 255),
			G: uint8((int(prev.G) + dg) % 255),
			B: uint8((int(prev.B) + db) % 255),
			A: prev.A,
		}}
	} else if op2 == QoiOpLuma {
		dg := int((data[0]&0b00111111)>>0) - 32
		dr := int((data[1]&0b11110000)>>4) - 8 + dg
		db := int((data[1]&0b00001111)>>0) - 8 + dg
		return 2, []color.NRGBA{{
			R: uint8((int(prev.R) + dr) % 255),
			G: uint8((int(prev.G) + dg) % 255),
			B: uint8((int(prev.B) + db) % 255),
			A: prev.A,
		}}
	} else if op2 == QoiOpRun { // is ok
		rep := int(data[0]&0b00111111) + 1
		pixels := make([]color.NRGBA, rep)
		for i := range pixels {
			pixels[i] = *prev
		}
		return 1, pixels
	}

	log.Fatal("unknown op code")
	return 1, []color.NRGBA{}
}

func genIndex(pixel color.NRGBA) int {
	return (int(pixel.R)*3 + int(pixel.G)*5 + int(pixel.B)*7 + int(pixel.A)*11) % 64
}
