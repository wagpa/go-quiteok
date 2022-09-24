package main

import (
	"go_quiteok/qoi"
	"image/png"
	"os"
)

func main() {
	//fileName := "./tmp/kodim10"
	fileName := "./tmp/dice"

	// png -> qoi
	if true {
		reader, oErr := os.Open(fileName + ".png")
		if oErr != nil {
			panic(oErr)
		}
		decoded, dErr := png.Decode(reader)
		if dErr != nil {
			panic(dErr)
		}

		writer, ofErr := os.OpenFile(fileName+".png.qoi", os.O_CREATE|os.O_WRONLY, 0644)
		if ofErr != nil {
			panic(ofErr)
		}
		eErr := qoi.Encode(writer, decoded)
		if eErr != nil {
			panic(eErr)
		}
	}

	// qoi -> png
	if true {
		reader, oErr := os.Open(fileName + ".png.qoi")
		if oErr != nil {
			panic(oErr)
		}
		decoded, dErr := qoi.Decode(reader)
		if dErr != nil {
			panic(dErr)
		}

		writer, ofErr := os.OpenFile(fileName+".png.qoi.png", os.O_CREATE|os.O_WRONLY, 0644)
		if ofErr != nil {
			panic(ofErr)
		}
		eErr := png.Encode(writer, decoded)
		if eErr != nil {
			panic(eErr)
		}
	}
}
