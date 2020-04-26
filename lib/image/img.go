// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package image

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/pkg/errors"
)

type ARGBImage struct {
	Width  uint32
	Height uint32
	/**
	 * Holds pixels in blocks of 4 bytes. Each block (from low address to high)
	 * the R,G,B and A component of the pixel
	 * Pixels are arranged left-to-right, top to bottom, so Pixels[0:4] is the leftmost pixel of the top row
	 * Pixel[4*(width-1):4*width] is the right-most pixel of the top row, Pixel[4*width:4*width+4] leftmost of second row etc.
	 */
	Pixels []byte
}

func (a *ARGBImage) PixelAt(row uint32, column uint32) ([]byte, error) {
	if column < 0 || column > a.Width || row < 0 || row > a.Height {
		return nil, errors.New("Out of range")
	} else {
		pos := 4 * (row*a.Width + column)
		return a.Pixels[pos : pos+4], nil
	}
}

func (a *ARGBImage) AsPng() ([]byte, error) {
	pngData := image.NewRGBA(image.Rect(0, 0, int(a.Width), int(a.Height)))
	buf := &bytes.Buffer{}
	for row := uint32(0); row < a.Height; row++ {
		for column := uint32(0); column < a.Width; column++ {
			pixelAsARGB, _ := a.PixelAt(row, column)
			pixelRGBA := color.RGBA{R: pixelAsARGB[1], G: pixelAsARGB[2], B: pixelAsARGB[3], A: pixelAsARGB[0]}
			pngData.Set(int(column), int(row), color.RGBA(pixelRGBA))
		}
	}
	if err := png.Encode(buf, pngData); err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}

type ARGBIcon struct {
	Images []ARGBImage
}

func ARGBIconHashName(ai ARGBIcon) string {
	var hasher = sha1.New()
	for _, image := range ai.Images {
		hasher.Write(image.Pixels)
	}
	return fmt.Sprintf("%X", hasher.Sum(nil))
}

type ImageData struct {
	Width        int32
	Height       int32
	Rowstride    int32
	HasAlpha     bool
	BitsPrSample int32
	Channels     int32
	Data         []uint8
}

func ImageDataHashName(id ImageData) string {
	var hasher = sha1.New()
	hasher.Write(id.Data)
	return fmt.Sprintf("%X", hasher.Sum(nil))
}

func (id ImageData) AsPng() ([]byte, error) {
	if id.Channels != 3 && id.Channels != 4 {
		return nil, fmt.Errorf("Don't know how to deal with %d", id.Channels)
	} else if id.Channels == 4 && !id.HasAlpha {
		return nil, fmt.Errorf("hasAlpha, but not 4 channels")
	}

	pngData := image.NewRGBA(image.Rect(0, 0, int(id.Width), int(id.Height)))
	pixelStride := id.Rowstride / id.Width
	var count = 0

	for y := int32(0); y < id.Height; y++ {
		for x := int32(0); x < id.Width; x++ {
			count++
			pos := int(y*id.Rowstride + x*pixelStride)
			var alpha = uint8(255)
			if id.HasAlpha {
				alpha = id.Data[pos+3]
			}
			pngData.Set(int(x), int(y), color.RGBA{R: id.Data[pos], G: id.Data[pos+1], B: id.Data[pos+2], A: alpha})
		}
	}

	buf := &bytes.Buffer{}
	err := png.Encode(buf, pngData)
	if err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}
