package qoi

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// Encode encodes a given image to the QuiteOk image format and writes the encoded bytes to the writer.
func Encode(writer io.Writer, image image.Image) error {
	// prerequisite
	index := 0 // pixel index in image
	prev := color.NRGBA{A: 255}
	seen := [64]color.NRGBA{}
	var data []byte

	// TODO offsets? image.Bounds().Min.Y
	var curr color.NRGBA
	xy := func(i int) (int, int) {
		x := i % image.Bounds().Max.X
		y := i / image.Bounds().Max.X
		return x, y
	}

	// read pixels & generate data
	size := image.Bounds().Dx() * image.Bounds().Dy()
	for index < size {
		x, y := xy(index)
		curr = color.NRGBAModel.Convert(image.At(x, y)).(color.NRGBA)

		// OpRun
		if prev == curr {
			run := 1
			for run < 62 && run+index < size {
				nx, ny := xy(index + run)
				next := color.NRGBAModel.Convert(image.At(nx, ny)).(color.NRGBA)
				if next != prev {
					break
				}
				run++
			}

			data = append(
				data,
				OpRun|byte(run-1),
			)
			seen[hashColor(&curr)] = curr
			index += run

			continue
		}

		// OpIndex
		if hash := hashColor(&curr); seen[hash] == curr {
			data = append(
				data,
				OpIndex|byte(hash),
			)
			prev = curr
			index += 1

			continue
		}

		// OpRgba
		if prev.A != curr.A {
			data = append(
				data,
				OpRgba,
				curr.R,
				curr.G,
				curr.B,
				curr.A,
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		// alpha channel is the same

		dg := modDist(int(prev.G), int(curr.G), 256)
		dr := modDist(int(prev.R), int(curr.R), 256)
		db := modDist(int(prev.B), int(curr.B), 256)

		// OpDiff
		if (-2 <= dr && dr <= 1) && (-2 <= dg && dg <= 1) && (-2 <= db && db <= 1) {
			data = append(
				data,
				OpDiff|
					byte((dr+2)<<4)|
					byte((dg+2)<<2)|
					byte((db+2)<<0),
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		drDg := dr - dg
		dbDg := db - dg

		// OpLuma
		if (-32 <= dg && dg <= 31) && (-8 <= drDg && drDg <= 7) && (-8 <= dbDg && dbDg <= 7) {
			data = append(
				data,
				OpLuma|byte(dg+32),
				byte((drDg+8)<<4)|byte(dbDg+8),
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}

		// OpRgb
		if true {
			data = append(
				data,
				OpRgb,
				curr.R,
				curr.G,
				curr.B,
			)
			seen[hashColor(&curr)] = curr
			prev = curr
			index += 1

			continue
		}
	}

	// create file bytes
	var file []byte

	// add magic
	file = append(
		file,
		Magic...,
	)
	// add size
	file = binary.BigEndian.AppendUint32(file, uint32(image.Bounds().Dx()))
	file = binary.BigEndian.AppendUint32(file, uint32(image.Bounds().Dy()))
	// add format
	file = append(
		file,
		uint8(4),
		uint8(0),
	)
	// add pixels
	file = append(
		file,
		data...,
	)
	// add eof indicator
	file = append(
		file,
		eof[:]...,
	)

	// write to file
	if _, err := writer.Write(file); err != nil {
		return err
	}
	return nil
}

// Calculates the (directed) distance between two numbers in a wrapped room (modulo room).
//
//	x := modDist(a, b, m)
//	(a + x + m) % m == b
func modDist(a int, b int, m int) int {
	// make sure, that a <= b
	v := 1
	if a > b {
		b, a = a, b
		v = -1
	}

	ab := b - a
	ba := (a + m) - b

	if ab < ba {
		return v * ab
	} else {
		return v * -1 * ba
	}
}
