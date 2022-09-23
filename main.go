package main

import (
	"go_quiteok/qoi"
	"image/png"
	"log"
	"os"
)

func main() {
	name := "./tmp/dice"

	original, originalOpenErr := os.Open(name + ".png")
	if originalOpenErr != nil {
		log.Fatalln(originalOpenErr)
	}

	originalImg, originalDecodeErr := png.Decode(original)
	if originalDecodeErr != nil {
		log.Fatalln(originalDecodeErr)
	}

	file, openErr := os.Open(name + ".qoi")
	if openErr != nil {
		log.Fatalln(openErr)
	}

	img, decodeErr := qoi.Decode(file)
	if decodeErr != nil {
		log.Fatalln(decodeErr)
	}

	// validate pixels
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.At(x, y) != originalImg.At(x, y) {
				log.Fatal("different pixels at ", x, y, " expected ", originalImg.At(x, y), " actual ", img.At(x, y))
			}
		}
	}

	if f, err := os.Create(name + "_2.png"); err != nil {
		encodeErr := png.Encode(f, img)
		if encodeErr != nil {
			log.Fatal(encodeErr)
		}
	}

	if f, err := os.Create(name + "_2.qoi"); err != nil {
		encodeErr := qoi.Encode(f, img)
		if encodeErr != nil {
			log.Fatal(encodeErr)
		}
	}
}
