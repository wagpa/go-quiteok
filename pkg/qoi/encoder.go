package qoi

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// Encode encodes an image.Image to an io.Writer with the QuiteOk image format (qoi).
// It first uses EncodeConfig to encode the image config and then encodes the pixel data.
func Encode(w io.Writer, img image.Image) error {
	if err := EncodeConfig(w, img); err != nil {
		return err
	}
	return encodePixels(w, img)
}

// EncodeConfig encodes an image.Config to an io.Writer with the QuiteOk image format (qoi).
// The advanced io.Writer cannot be used to then Encode the image.Image.
func EncodeConfig(w io.Writer, img image.Image) error {
	buf := make([]byte, 0, 14)

	buf = append(buf, Magic...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(img.Bounds().Dx()))
	buf = binary.BigEndian.AppendUint32(buf, uint32(img.Bounds().Dy()))
	buf = append(buf, byte(4))
	buf = append(buf, byte(0))

	_, err := w.Write(buf)
	return err
}

func encodePixels(w io.Writer, img image.Image) error {
	last := color.NRGBA{A: 255}
	seen := make([]color.NRGBA, 64)
	run := 0

	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			curr := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			hash := hashColor(curr)

			// handle run
			if run >= 62 || (run >= 1 && last != curr) {
				_, err := w.Write([]byte{
					opRun | byte(run-1),
				})
				if err != nil {
					return err
				}
				run = 0
			}

			// opRun
			if last == curr {
				run += 1
				continue
			}

			// opIndex
			other := seen[hash]
			if other == curr {
				_, err := w.Write([]byte{
					opIndex | hash,
				})
				if err != nil {
					return err
				}
				seen[hash] = curr
				last = curr
				continue
			}

			if curr.A == last.A {

				// opDiff
				dr := curr.R - last.R
				dg := curr.G - last.G
				db := curr.B - last.B
				if (254 <= dr || dr <= 1) && (254 <= dg || dg <= 1) && (254 <= db || db <= 1) {
					_, err := w.Write([]byte{
						opDiff | ((dr + 2) << 4) | ((dg + 2) << 2) | ((db + 2) << 0),
					})
					if err != nil {
						return err
					}
					seen[hash] = curr
					last = curr
					continue
				}

				// opLuma
				drDg := dr - dg
				dbDg := db - dg
				if (248 <= drDg || drDg <= 7) && (224 <= dg || dg <= 31) && (248 <= dbDg || dbDg <= 7) {
					_, err := w.Write([]byte{
						opLuma | (dg + 32),
						((drDg + 8) << 4) | (dbDg + 8),
					})
					if err != nil {
						return err
					}
					seen[hash] = curr
					last = curr
					continue
				}

				// opRgb
				_, err := w.Write([]byte{
					opRgb,
					curr.R,
					curr.G,
					curr.B,
				})
				if err != nil {
					return err
				}
				seen[hash] = curr
				last = curr
				continue
			}

			// opRgba
			_, err := w.Write([]byte{
				opRgba,
				curr.R,
				curr.G,
				curr.B,
				curr.A,
			})
			if err != nil {
				return err
			}
			seen[hash] = curr
			last = curr
		}
	}

	// handle dangling run
	if run >= 1 {
		_, err := w.Write([]byte{
			opRun | byte(run-1),
		})
		if err != nil {
			return err
		}
	}

	// write EOF
	_, err := w.Write(eof[:])
	if err != nil {
		return err
	}

	return nil
}
