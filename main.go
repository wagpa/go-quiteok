package main

import (
	"go_quiteok/qoi"
	"image/png"
	"log"
	"os"
)

func main() {
	fileName := "./tmp/kodim10.qoi"

	reader, _ := os.Open(fileName)
	decoded, _ := qoi.Decode(reader)

	writer, _ := os.OpenFile(fileName+".png", os.O_CREATE|os.O_WRONLY, 0644)
	encErr := png.Encode(writer, decoded)
	if encErr != nil {
		log.Panic(encErr)
	}
}
