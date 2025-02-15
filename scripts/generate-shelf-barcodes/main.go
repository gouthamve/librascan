package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/ean"
	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gouthamve/librascan/migrations"
)

func main() {
	for idx, shelf := range migrations.Shelfs {
		idx++
		// Create 6 digit shelf ID
		shelfID := fmt.Sprintf("%06d", idx)

		for rowIdx := 1; rowIdx <= shelf.Rows; rowIdx++ {
			code := fmt.Sprintf("%s%d", shelfID, rowIdx)
			eanCode, err := ean.Encode(code)
			if err != nil {
				panic(err)
			}

			eanCodeScaled, err := barcode.Scale(eanCode, 250, 75)
			if err != nil {
				panic(err)
			}

			// create a new image
			font, err := truetype.Parse(goregular.TTF)
			if err != nil {
				panic(err)
			}
			face := truetype.NewFace(font, &truetype.Options{Size: 20})

			imgWidth := 250.0
			imgHeight := 150.0
			imgCtx := gg.NewContext(int(imgWidth), int(imgHeight))
			imgCtx.SetFontFace(face)

			imgCtx.DrawRectangle(0, 0, imgWidth, imgHeight)
			imgCtx.SetRGB(1, 1, 1)
			imgCtx.Fill()

			imgCtx.SetRGB(0, 0, 0)

			imgCtx.DrawStringAnchored(shelf.Name, 1, 90, 0, 1)
			imgCtx.DrawImage(eanCodeScaled, 0, 5)

			imgCtx.DrawStringAnchored(fmt.Sprintf("Row: %d", rowIdx), 1, 110, 0, 1)

			// rotate image 90 degrees
			img := imaging.Rotate270(imgCtx.Image())

			fmt.Println(img.Bounds())

			file, err := os.Create(fmt.Sprintf("./barcodes/barcode-shelf-%s-row-%d.png", shelfID, rowIdx))
			if err != nil {
				panic(err)
			}

			// encode the barcode as png
			png.Encode(file, img)
			file.Close()
		}
	}
}
