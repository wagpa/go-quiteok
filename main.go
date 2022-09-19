package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func main() {
	dat, err := os.ReadFile("./tmp/dice.qoi")
	if err != nil {
		log.Fatal("read error:", err)
	}

	file := decode(dat)
	fmt.Printf("read file %+v\n", file.Header)
}

type QuiteOkHeader struct {
	Magic      [4]byte // "qoif" -> 113 111 105 102
	Width      uint32
	Height     uint32
	Channels   uint8
	Colorspace uint8
}

type QuiteOkFile struct {
	Header QuiteOkHeader
	Data   []byte
}

func decode(data []byte) QuiteOkFile {
	if len(data) < 14 {
		log.Fatal("cannot decode: ", "data to short")
	}

	var header QuiteOkHeader
	headerData := data[:14]
	log.Print("header data len: ", len(headerData), headerData)
	reader := bytes.NewReader(headerData)
	dec := gob.NewDecoder(reader)
	err := dec.Decode(&header)
	if err != nil {
		log.Fatal("header decode error: ", err)
	}
	return QuiteOkFile{
		Header: header,
		Data:   data[14:],
	}
}
