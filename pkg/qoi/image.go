package qoi

import (
	"image"
	"image/color"
)

// QuiteOkImage is the image type for the QuiteOk image format. It implements the image.Image interface.
type QuiteOkImage struct {
	header QuiteOkHeader
	pixels []color.NRGBA
}

// QuiteOkHeader is the header data of a QuiteOk image format. See qoi.QuiteOkImage for usage.
type QuiteOkHeader struct {
	width      uint32
	height     uint32
	channels   uint8
	colorspace uint8
}

func (img QuiteOkImage) ColorModel() color.Model {
	return color.NRGBAModel
}

func (img QuiteOkImage) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{},
		Max: image.Point{
			X: int(img.header.width),
			Y: int(img.header.height),
		},
	}
}

func (img QuiteOkImage) At(x, y int) color.Color {
	return img.pixels[x+y*int(img.header.width)]
}
