package main

import (
	"errors"
	"image/png"
	"os"

	"github.com/boombuler/barcode"
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

	qrCode, err := barcode.Decode(img)
	if err != nil {
		if _, ok := err.(barcode.ReaderError); ok {
			return "", errors.New("QR code not found")
		}
		return "", err
	}

	return qrCode.Content, nil
}
