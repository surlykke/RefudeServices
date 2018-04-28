// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package icons

import (
	"image"
	"bytes"
	"image/color"
	"image/png"
	"fmt"
	"net/http"
	"github.com/surlykke/RefudeServices/lib/service"
	"hash/fnv"
	"github.com/pkg/errors"
	"github.com/surlykke/RefudeServices/lib/xdg"
	"os"
	"log"
	"github.com/surlykke/RefudeServices/lib/resource"
	"strconv"
)

type PNGImg struct {
	Size    int
	pngData []byte
}

type PNGIcon struct {
	resource.DefaultResource
	images   []PNGImg
}

func (icon* PNGIcon) GET(w http.ResponseWriter, r *http.Request) {
		var data []byte = nil

		size := 32
		if sizes, ok := r.URL.Query()["size"]; ok && len(sizes) == 1 {
			if tmp, err := strconv.Atoi(sizes[0]); err == nil {
				size = tmp
			}
		}

		for _, sizedPng := range icon.images {
			data = sizedPng.pngData
			if sizedPng.Size > size {
				break
			}
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
}


func (icon* PNGIcon) MediaType() resource.MediaType{
	return "image/png"
}

type Img struct {
	Width  int32
	Height int32
	/*
	 * Holds pixels in blocks of 4 bytes. Each block (from low address to high)
	 * the A,R,G and B component of the pixel
	 */
	Pixels []byte
}

func (img *Img) PixelAt(row int32, column int32) ([]byte, error) {
	if column < 0 || column > img.Width || row < 0 || row > img.Height {
		return nil, errors.New("Out of range")
	} else {
		pos := 4 * (row*img.Width + column)
		return img.Pixels[pos : pos+4], nil
	}
}

type Icon []Img

/**
 * Icons retrieved from the X-server (EWMH) will come as arrays of uint. There will be first two ints giving
 * width and height, then width*height uints each holding a pixel in ARGB format (on 64bit system the 4 most
 * significant bytes are not used). After that it may repeat: again a width and height uint and then pixels and
 * so on...
 */
func ExtractARGBIcon(uints []uint) []Img {
	res := make([]Img, 0)
	for len(uints) >= 2 {
		width := int32(uints[0])
		height := int32(uints[1])
		uints = uints[2:]
		if len(uints) < int(width*height) {
			break
		}
		pixels := make([]byte, 4*width*height)
		for pos := int32(0); pos < width*height; pos++ {
			pixels[4*pos] = uint8((uints[pos] & 0xFF000000) >> 24)
			pixels[4*pos+1] = uint8((uints[pos] & 0xFF0000) >> 16)
			pixels[4*pos+2] = uint8((uints[pos] & 0xFF00) >> 8)
			pixels[4*pos+3] = uint8(uints[pos] & 0xFF)
		}
		res = append(res, Img{Width: width, Height: height, Pixels: pixels})
		uints = uints[width*height:]
	}

	return res
}

var hicolorMapSizes = map[int32]bool{
	16:  true,
	22:  true,
	24:  true,
	32:  true,
	36:  true,
	48:  true,
	64:  true,
	72:  true,
	96:  true,
	128: true,
	192: true,
	256: true,
	512: true,
}

func SaveAsPngToSessionIconDir(argbIcon Icon) string {
	var sessionIconDir = xdg.RuntimeDir + "/org.refude.icon-service-session-icons/"
	var wroteSomething = false
	hash := fnv.New64a()
	for _, img := range argbIcon {
		hash.Write(img.Pixels)
	}
	var iconName = fmt.Sprintf("%X", hash.Sum64())

	for _, img := range argbIcon {
		fmt.Println("Consider img, width:", img.Width, ", height:", img.Height)
		if img.Height == img.Width && hicolorMapSizes[img.Height] {
			var destDir = fmt.Sprintf("%shicolor/%dx%d/apps", sessionIconDir, img.Width, img.Height)
			var destPath = destDir + "/" + iconName + ".png"
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
					continue
				}
				wroteSomething = wroteSomething || makeAndWritePng(img, destPath)
			} else {
				continue
			}
		} else {
			fmt.Println("Skip")
		}
	}

	if wroteSomething {
		if _, err := os.Create(sessionIconDir + "/marker"); err != nil {
			log.Println("Error updating marker:", err)
		}
	}

	return iconName
}

func makeAndWritePng(img Img, path string) bool {
	pngData := image.NewRGBA(image.Rect(0, 0, int(img.Width), int(img.Height)))
	buf := bytes.Buffer{}
	for row := int32(0); row < img.Height; row++ {
		for column := int32(0); column < img.Width; column++ {
			pixelAsARGB, _ := img.PixelAt(row, column)
			pixelRGBA := color.RGBA{R: pixelAsARGB[1], G: pixelAsARGB[2], B: pixelAsARGB[3], A: pixelAsARGB[0]}
			pngData.Set(int(column), int(row), color.RGBA(pixelRGBA))
		}
	}
	png.Encode(&buf, pngData)
	w, err := os.Create(path);
	if err != nil {
		log.Println("Unable to write", path, err)
		return false
	}
	defer w.Close()
	w.Write(buf.Bytes())
	return true
}

func ServeAsPng(argbIcon Icon) (string, error) {
	hash := fnv.New64a()
	for _, img := range argbIcon {
		hash.Write(img.Pixels)
	}
	path := fmt.Sprintf("/icons/%X", hash.Sum64())

	if !service.Has(path) {
		pngIcon := PNGIcon{}
		for _, img := range argbIcon {
			pngData := image.NewRGBA(image.Rect(0, 0, int(img.Width), int(img.Height)))
			buf := bytes.Buffer{}
			for row := int32(0); row < img.Height; row++ {
				for column := int32(0); column < img.Width; column++ {
					pixelAsARGB, _ := img.PixelAt(row, column)
					pixelRGBA := color.RGBA{R: pixelAsARGB[1], G: pixelAsARGB[2], B: pixelAsARGB[3], A: pixelAsARGB[0]}
					pngData.Set(int(column), int(row), color.RGBA(pixelRGBA))
				}
			}
			png.Encode(&buf, pngData)
			pngIcon.images = append(pngIcon.images, PNGImg{int(img.Width), buf.Bytes()})
		}
		if len(pngIcon.images) < 1 {
			return "", fmt.Errorf("No icons in argument")
		}

		service.Map(path, &pngIcon)
	}

	return path, nil
}
