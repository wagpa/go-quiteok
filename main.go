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
