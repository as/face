package face

import (
	"image"
	"image/color"
	"image/draw"
)

// SkinMask sets mask to covering non-facial colors in the RGB
// colorspace according to pixels in src and returns mask. If mask
// is nil, the function allocates a new mask. Drawing the resulting
// mask over src results in an image where only the facial pixels
// have a non-zero alpha.
//
// The mask dimensions correspond to the pixels processed by this
// function in src. If src is an *image.RGBA and mask is nil or an
// *image.Alpha, this function takes a fast-path if the bounds are
// the same for src and mask (or mask is nil).
//
// Note: This function currently assumes the input image is chromatic
// using a grayscale image will yield poor results.
func SkinMask(src image.Image, mask draw.Image) (mask0 draw.Image, cover float64) {
	return skinMaskColor(src, mask)
}

// Content rates the level of posterization in the provided image in
// r in the range [0, 256). The range [0, 64] generally indicates that
// src is highly posterized.
//
// If src.Bounds == r, and src is an *image.RGBA, a fast-
// path is taken.
func Content(src image.Image, r image.Rectangle) uint8 {
	const (
		threshold = 64
	)
	if src.Bounds() == r {
		src, ok := src.(*image.RGBA)
		if ok {
			return contentRGBA(src)
		}
	}
	Y := 0
	C := [256 * 3]byte{}
	for y := r.Min.Y; y <= r.Max.Y; y++ {
		for x := r.Min.X; x <= r.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			r >>= 8
			g >>= 8
			b >>= 8
			Y = int(r+g+b) / 3
			C[byte(Y)]++
		}
	}
	n := 0
	for _, v := range C {
		if v > threshold {
			n++
		}
	}
	if n > 255 {
		n = 255
	}
	return byte(n)
}

func skinMaskColor(src image.Image, mask draw.Image) (mask0 draw.Image, cover float64) {
	var amask bool
	if mask == nil {
		mask = image.NewAlpha(src.Bounds())
		amask = true
	} else {
		_, amask = mask.(*image.Alpha)
	}
	if src.Bounds() == mask.Bounds() {
		if src, ok := src.(*image.RGBA); ok && amask {
			return skinMaskColorRGBA(src, mask.(*image.Alpha))
		}
	}

	const (
		minR       = 75 << 8 || 75
		minRGdelta = 20 << 8 || 20
		maxRGdelta = 90 << 8 || 90
		maxRGrat   = 2.5
	)
	r := mask.Bounds()
	m := 0
	for y := r.Min.Y; y <= r.Max.Y; y++ {
		for x := r.Min.X; x <= r.Max.X; x++ {
			r, g, _, _ := src.At(x, y).RGBA()
			if r < minR {
				continue
			}
			if r-g < minRGdelta || r-g > maxRGdelta {
				continue
			}
			if float32(r)/float32(g) >= maxRGrat {
				continue
			}
			mask.Set(x, y, color.Opaque)
			m++
		}
	}
	return mask, float64(m) / float64(r.Dy()*r.Dx())
}

func skinMaskColorRGBA(src *image.RGBA, mask *image.Alpha) (mask0 *image.Alpha, cover float64) {
	const (
		minR       = 75
		minRGdelta = 20
		maxRGdelta = 90
		maxRGrat   = 2.5
	)

	r := mask.Bounds()
	if src.Bounds() != r {
		panic("skinMaskColorRGBA: doesn't support subimage masks")
	}

	sp := (r.Min.Y-src.Rect.Min.Y)*src.Stride + (r.Min.X-src.Rect.Min.X)*4
	ep := r.Dx() * r.Dy() * 4
	mp := -1
	n := 0

	for pix := src.Pix; sp != ep; sp += 4 {
		mp++
		g := pix[sp+1]
		r := pix[sp]
		if r < minR {
			continue
		}
		if r-g < minRGdelta || r-g > maxRGdelta {
			continue
		}
		if float32(r)/float32(g) >= maxRGrat {
			continue
		}
		mask.Pix[mp] = 255
		n++
	}
	return mask, float64(n) / float64(r.Dy()*r.Dx())
}

func contentRGBA(src *image.RGBA) uint8 {
	const (
		threshold = 64
	)
	r := src.Bounds()
	C := [256]byte{}
	sp := (r.Min.Y-src.Rect.Min.Y)*src.Stride + (r.Min.X-src.Rect.Min.X)*4
	ep := src.Bounds().Dx() * src.Bounds().Dy() * 4
	pix := src.Pix
	for sp != ep {
		C[(pix[sp]+pix[sp+1]+pix[sp+2])/3]++
		sp += 4
	}
	c := 0
	for _, v := range C {
		if v > threshold {
			c++
		}
	}
	if c > 255 {
		c = 255
	}
	return byte(c)
}
