package main

import (
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/ean"
)

func main() {
	// Create the barcode
	eanCode, err := ean.Encode("9781250319180")
	if err != nil {
		panic(err)
	}

	// fmt.Println(eanCode.Bounds())

	eanCodeScaled, err := barcode.Scale(eanCode, 200, 100)
	if err != nil {
		panic(err)
	}

	file, _ := os.Create("barcode.png")
	defer file.Close()

	// encode the barcode as png
	png.Encode(file, eanCodeScaled)
}
