package qoi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

func Decode(r io.Reader) (image.Image, error) {
	conf, err := decodeConfig(r)
	if err != nil {
		return nil, err
	}
	return decodePixels(r, conf)
}

func decodeConfig(r io.Reader) (image.Config, error) {
	// read the header bytes
	buf := make([]byte, 14)
	if _, err := io.ReadAtLeast(r, buf, len(buf)); err != nil {
		return image.Config{}, err
	}
	// validate the magic bytes
	if string(buf[:4]) != Magic {
		return image.Config{}, fmt.Errorf("%w: expected %q, actual %q", ErrInvalidMagic, Magic, string(buf[:4]))
	}
	// read the width and height (ignores `channels` and `colorspace`)
	return image.Config{
		Width:      int(binary.BigEndian.Uint32(buf[4:8])),
		Height:     int(binary.BigEndian.Uint32(buf[8:12])),
		ColorModel: color.NRGBAModel,
	}, nil
}

func decodePixels(r io.Reader, conf image.Config) (image.Image, error) {
	img := image.NewNRGBA(image.Rect(0, 0, conf.Width, conf.Height))

	last := &startPixel
	buf := make([]byte, 8)
	run := byte(0)
	seen := make([]*[4]uint8, 64)
	for i := range seen {
		seen[i] = &zeroPixel
	}

	// decode
	for y := 0; y < conf.Height; y++ {
		for x := 0; x < conf.Width; x++ {
			off := img.PixOffset(x, y)
			pix := (*[4]uint8)(img.Pix[off : off+4 : off+4])
			// handle other run iterations
			if run > 0 {
				run -= 1
				pix[0] = last[0]
				pix[1] = last[1]
				pix[2] = last[2]
				pix[3] = last[3]
				continue
			}
			// decode new pixel
			if _, err := r.Read(buf[:1]); err != nil {
				return nil, err
			}
			switch {
			case buf[0] == OpRgb:
				if _, err := r.Read(buf[1:4]); err != nil {
					return nil, err
				}
				pix[0] = buf[1]
				pix[1] = buf[2]
				pix[2] = buf[3]
				pix[3] = last[3]
				seen[hashColor(pix)] = pix
				last = pix
			case buf[0] == OpRgba:
				if _, err := r.Read(buf[1:5]); err != nil {
					return nil, err
				}
				pix[0] = buf[1]
				pix[1] = buf[2]
				pix[2] = buf[3]
				pix[3] = buf[4]
				seen[hashColor(pix)] = pix
				last = pix
			case buf[0]&OpMask == OpIndex:
				s := seen[buf[0]]
				pix[0] = s[0]
				pix[1] = s[1]
				pix[2] = s[2]
				pix[3] = s[3]
				last = pix
			case buf[0]&OpMask == OpDiff:
				pix[0] = last[0] + (buf[0]>>4)&0x3 - 2
				pix[1] = last[1] + (buf[0]>>2)&0x3 - 2
				pix[2] = last[2] + (buf[0]>>0)&0x3 - 2
				pix[3] = last[3]
				seen[hashColor(pix)] = pix
				last = pix
			case buf[0]&OpMask == OpLuma:
				if _, err := r.Read(buf[1:2]); err != nil {
					return nil, err
				}
				dg := (buf[0] & 0b00111111) - 32
				dr := (buf[1]&0b11110000)>>4 - 8 + dg
				db := (buf[1]&0b00001111)>>0 - 8 + dg
				pix[0] = last[0] + dr
				pix[1] = last[1] + dg
				pix[2] = last[2] + db
				pix[3] = last[3]
				seen[hashColor(pix)] = pix
				last = pix
			case buf[0]&OpMask == OpRun:
				run = buf[0]&0b00111111 + 1
				if run > 62 || run < 1 {
					return nil, fmt.Errorf("%w: actual %d", ErrInvalidRunLength, run)
				}
				// first run iteration
				run -= 1
				pix[0] = last[0]
				pix[1] = last[1]
				pix[2] = last[2]
				pix[3] = last[3]
			}
		}
	}

	// check EOF sequence
	if _, err := r.Read(buf); err != nil {
		return nil, err
	}
	if !bytes.Equal(buf, eof[:]) {
		return nil, ErrInvalidEOF
	}

	return img, nil
}
