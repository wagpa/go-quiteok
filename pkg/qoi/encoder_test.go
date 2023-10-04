package qoi

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"testing"
)

func TestEncode(t *testing.T) {
	for _, fname := range testFiles {
		t.Run(fname, func(t *testing.T) {

			// given
			pngFile, err := os.Open(fmt.Sprintf("./data/%s.png", fname))
			if err != nil {
				t.Fatal(err)
			}
			pngImg, err := png.Decode(pngFile)
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer

			// when
			err = Encode(&buf, pngImg)
			if err != nil {
				t.Fatal(err)
			}

			// then
			qoiImg, err := Decode(&buf)
			if err != nil {
				t.Fatal(err)
			}
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

func BenchmarkEncode(b *testing.B) {
	for _, fname := range testFiles {
		b.Run(fname, func(b *testing.B) {

			// read and decode png file
			b.StopTimer()
			qoiFile, err := os.Open(fmt.Sprintf("./data/%s.qoi", fname))
			if err != nil {
				b.Fatal(err)
			}
			qoiStat, err := qoiFile.Stat()
			if err != nil {
				b.Fatal(err)
			}
			pngFile, err := os.Open(fmt.Sprintf("./data/%s.png", fname))
			if err != nil {
				b.Fatal(err)
			}
			pngImg, err := png.Decode(pngFile)
			if err != nil {
				b.Fatal(err)
			}

			bs := make([]byte, 0, qoiStat.Size()*2)
			buf := bytes.NewBuffer(bs)
			b.StartTimer()

			// encode file
			for i := 0; i < b.N; i++ {
				buf.Reset()
				err = Encode(buf, pngImg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
