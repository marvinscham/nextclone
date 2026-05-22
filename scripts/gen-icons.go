//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

var appSizes = []int{16, 24, 32, 48, 64, 128, 256, 512}
var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}
var faviconSizes = []int{16, 32, 48}

func main() {
	src := loadPNG("assets/sync_512.png")

	for _, size := range appSizes {
		writePNG(fmt.Sprintf("assets/sync_%d.png", size), resize(src, size))
	}
	writeICO("assets/nextclone.ico", src, icoSizes)
	writeICO("assets/favicon.ico", src, faviconSizes)
}

func loadPNG(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return img
}

func resize(src image.Image, size int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func writePNG(path string, img image.Image) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func writeICO(path string, src image.Image, sizes []int) {
	images := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, resize(src, size)); err != nil {
			panic(err)
		}
		images = append(images, buf.Bytes())
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	must(binary.Write(f, binary.LittleEndian, uint16(0)))
	must(binary.Write(f, binary.LittleEndian, uint16(1)))
	must(binary.Write(f, binary.LittleEndian, uint16(len(images))))

	offset := uint32(6 + len(images)*16)
	for i, data := range images {
		size := sizes[i]
		width := byte(size)
		height := byte(size)
		if size == 256 {
			width = 0
			height = 0
		}
		_, err := f.Write([]byte{width, height, 0, 0})
		must(err)
		must(binary.Write(f, binary.LittleEndian, uint16(1)))
		must(binary.Write(f, binary.LittleEndian, uint16(32)))
		must(binary.Write(f, binary.LittleEndian, uint32(len(data))))
		must(binary.Write(f, binary.LittleEndian, offset))
		offset += uint32(len(data))
	}

	for _, data := range images {
		_, err := f.Write(data)
		must(err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
