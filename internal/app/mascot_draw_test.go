package app

import (
	"bytes"
	"image/png"
	"testing"
)

func TestMascotPNG_validPNG(t *testing.T) {
	b, err := MascotPNG()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 {
		t.Fatalf("png too small: %d", len(b))
	}
	_, err = png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
}

func TestEncodeMascotPNG_size(t *testing.T) {
	var buf limitedWriter
	if err := encodeMascotPNG(&buf, 128); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 128 || b.Dy() != 128 {
		t.Fatal(b)
	}
}
