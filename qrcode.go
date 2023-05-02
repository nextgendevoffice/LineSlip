package main

import (
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/kaxap/gozxing"
	"github.com/kaxap/gozxing/qrcode"
)

func DecodeQRCode(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)

	if err != nil {
		if _, ok := err.(gozxing.NotFound); ok {
			return "", errors.New("QR code not found")
		}
		return "", err
	}

	return result.GetText(), nil
}
