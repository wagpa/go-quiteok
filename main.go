package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

func main() {
	file, err := os.Open("./tmp/dice.qoi")
	if err != nil {
		log.Fatal("open error:", err)
	}

	header := decodeHeader(file)
	fmt.Printf("read file %+v\n", header)
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
	Data   []byte
}

func decodeHeader(file *os.File) QuiteOkHeader {
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

type Pixel struct {
	R byte
	G byte
	B byte
	A byte
}

const QOI_OP_RGB = byte(0b11111110)
const QOI_OP_RGBA = byte(0b11111111)
const QOI_OP_INDEX = byte(0b00000000)
const QOI_OP_DIFF = byte(0b01000000)
const QOI_OP_LUMA = byte(0b10000000)
const QOI_OP_RUN = byte(0b11000000)

func decodeBody(header QuiteOkHeader, file *os.File) QuiteOkFile {
	var data []byte
	// read raw bytes
	{
		dat, err := os.ReadFile("./tmp/dice.qoi")
		if err != nil {
			log.Fatal("read file error:", err)
		}
		data = dat[14:]
	}
	var pixels []Pixel

	lastPixel := Pixel{0, 0, 0, 255}
	seen := [64]Pixel{}
	index := 0

	for index < len(data) {
		tag8 := data[index] & 0b11111111 // 8bit tag
		tag2 := data[index] & 0b11000000 // 2bit tag

		// gen pixel
		var pixel Pixel

		if tag8 == QOI_OP_RGB {
			pixel = Pixel{
				R: data[index+1],
				G: data[index+2],
				B: data[index+3],
				A: 255, // TODO is this right?
			}
			index += 4
		} else if tag8 == QOI_OP_RGBA {
			pixel = Pixel{
				R: data[index+1],
				G: data[index+2],
				B: data[index+3],
				A: data[index+4],
			}
			index += 5
		} else if tag2 == QOI_OP_INDEX {
			pixelIndex := data[index+1] & 0b00111111
			pixel = seen[pixelIndex]
			index += 1
		} else if tag2 == QOI_OP_DIFF {
			dr := uint((data[index] & 0b00110000) >> 4)
			dg := uint((data[index] & 0b00001100) >> 2)
			db := uint((data[index] & 0b00000011) >> 0)
			// TODO
			index += 1
		} else if tag2 == QOI_OP_LUMA {
			// TODO
			index += 2
		} else if tag2 == QOI_OP_RUN {
			// TODO
			index += 1
		}

		// finish loop
		pixels = append(pixels, pixel)
		seen[genIndex(pixel)] = pixel
		lastPixel = pixel
	}

	// TODO
	return QuiteOkFile{}
}

func genIndex(pixel Pixel) uint8 {
	return (pixel.R*3 + pixel.G*5 + pixel.B*7 + pixel.A*11) % 64
}
