package test

import (
	"embed"
	"fmt"
	"go_quiteok/pkg/qoi"
	"image/png"
	"testing"
)

//go:embed all:data/*
var data embed.FS

var testFiles = []string{
	"testcard",
	"dice",
	"kodim10",
	"kodim23",
	"qoi_logo",
	"testcard_rgba",
	"wikipedia_008",
}

func TestDecode(t *testing.T) {
	for _, fname := range testFiles {
		t.Run(fname, func(t *testing.T) {

			// given
			qoiFile, err := data.Open(fmt.Sprintf("data/%s.qoi", fname))
			if err != nil {
				t.Fatal(err)
			}
			pngFile, err := data.Open(fmt.Sprintf("data/%s.png", fname))
			if err != nil {
				t.Fatal(err)
			}
			pngImg, err := png.Decode(pngFile)
			if err != nil {
				t.Fatal(err)
			}

			// when
			qoiImg, err := qoi.Decode(qoiFile)
			if err != nil {
				t.Fatal(err)
			}

			// then
			for y := 0; y < pngImg.Bounds().Dy(); y++ {
				for x := 0; x < pngImg.Bounds().Dx(); x++ {
					pngR, pngG, pngB, pngA := pngImg.At(x, y).RGBA()
					qoiR, qoiG, qoiB, qoiA := qoiImg.At(x, y).RGBA()
					if pngR != qoiR || pngG != qoiG || pngB != qoiB || pngA != qoiA {
						t.Fatal(fmt.Errorf("invalid pixel at (%d, %d): expected %+v, actual %+v", x, y, pngImg.At(x, y), qoiImg.At(x, y)))
					}
				}
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	for _, fname := range testFiles {
		b.Run(fname, func(b *testing.B) {

			// read file
			b.StopTimer()
			f, err := data.Open(fmt.Sprintf("data/%s.qoi", fname))
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()

			// decode file
			_, err = qoi.Decode(f)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}
