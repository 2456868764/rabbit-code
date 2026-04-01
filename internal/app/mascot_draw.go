package app

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

func encodeMascotPNG(w io.Writer, size int) error {
	if size < 64 {
		size = 64
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	cy := float64(size) / 2
	outerR := float64(size) * 0.48

	dark := color.NRGBA{R: 0x4a, G: 0x52, B: 0x5c, A: 0xff}
	light := color.NRGBA{R: 0x8b, G: 0x92, B: 0x9e, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	black := color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}
	accent := color.NRGBA{R: 0x3d, G: 0x45, B: 0x52, A: 0xff}
	ear := color.NRGBA{R: 0x42, G: 0x4a, B: 0x55, A: 0xff}

	s := float64(size)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := dist(px, py, cx, cy)
			if d > outerR+1 {
				continue
			}
			// circular soft edge
			if d > outerR {
				a := uint8((1 - (d-outerR)) * 255)
				if a > 255 {
					a = 255
				}
				img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: a})
				continue
			}

			// ears (drawn first in stack order below we overwrite with body — actually ears on top)
			inEarL := inEllipse(px, py, cx-s*0.18, cy-s*0.20, s*0.10, s*0.15)
			inEarR := inEllipse(px, py, cx+s*0.18, cy-s*0.20, s*0.10, s*0.15)
			// main body + head (single soft loaf)
			inFur := inEllipse(px, py, cx, cy+s*0.04, s*0.40, s*0.36)

			var out color.NRGBA
			switch {
			case !inFur && !inEarL && !inEarR:
				// inside circle mask but not rabbit silhouette → transparent
				out = color.NRGBA{A: 0}
			case inEarL || inEarR:
				out = ear
			default:
				out = dark
				if inEllipse(px, py, cx, cy+s*0.10, s*0.18, s*0.14) {
					out = light
				}
				if inBrace(px, py, cx, cy+s*0.20, s) {
					out = accent
				}
			}

			// face (on top of fur)
			if inFur || inEarL || inEarR {
				if dist(px, py, cx-s*0.12, cy-s*0.02) < s*0.065 {
					if dist(px, py, cx-s*0.12, cy-s*0.02) < s*0.032 {
						out = black
					} else {
						out = white
					}
				} else if dist(px, py, cx+s*0.12, cy-s*0.02) < s*0.065 {
					if dist(px, py, cx+s*0.12, cy-s*0.02) < s*0.032 {
						out = black
					} else {
						out = white
					}
				} else if dist(px, py, cx, cy+s*0.05) < s*0.022 {
					out = black
				}
			}

			img.SetNRGBA(x, y, out)
		}
	}
	return png.Encode(w, img)
}

func inBrace(px, py, cx, cy, sz float64) bool {
	w := sz * 0.035
	left := px >= cx-sz*0.07-w && px <= cx-sz*0.07+w && py >= cy-w*2.2 && py <= cy+w*2.2
	right := px >= cx+sz*0.07-w && px <= cx+sz*0.07+w && py >= cy-w*2.2 && py <= cy+w*2.2
	return left || right
}

func inEllipse(px, py, cx, cy, rx, ry float64) bool {
	if rx <= 0 || ry <= 0 {
		return false
	}
	dx, dy := (px-cx)/rx, (py-cy)/ry
	return dx*dx+dy*dy <= 1
}

func dist(x1, y1, x2, y2 float64) float64 {
	return math.Hypot(x1-x2, y1-y2)
}

type limitedWriter struct {
	b []byte
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func (w *limitedWriter) Bytes() []byte { return w.b }
