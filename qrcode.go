package main

import (
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/nfnt/resize"
)

func DecodeQRCode(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return "", err
	}

	// Resize the image if needed
	img = resize.Resize(300, 0, img, resize.Bilinear)

	qrCode, err := qr.Decode(img)
	if err != nil {
		return "", err
	}

	if qrCode == nil {
		return "", barcode.NotFound
	}

	return qrCode.Content, nil
}
