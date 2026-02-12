package timelapse

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func AddTimestampWithRoundedBox(data []byte, ts time.Time) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	label := ts.Format("15:04:05")

	face, _ := loadFont(20) // например 20px
	drawer := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.White),
		Face: face,
	}

	textWidth := drawer.MeasureString(label).Round()
	padding := 8
	radius := 10

	// Позиция текста
	x := rgba.Bounds().Max.X - textWidth - padding*2 - 20
	y := 50

	// Прямоугольник под текст
	box := image.Rect(
		x-padding,
		y-face.Metrics().Ascent.Round()-padding,
		x+textWidth+padding,
		y+face.Metrics().Descent.Round()+padding,
	)

	// Рисуем скруглённый прямоугольник
	drawRoundedRect(rgba, box, radius, color.RGBA{0, 0, 0, 150})

	// Рисуем текст
	drawer.Dot = fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}
	drawer.DrawString(label)

	var out bytes.Buffer
	err = jpeg.Encode(&out, rgba, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func loadFont(size float64) (font.Face, error) {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return face, nil
}

func drawRoundedRect(img *image.RGBA, rect image.Rectangle, r int, col color.Color) {
	// Центр
	inner := image.Rect(rect.Min.X+r, rect.Min.Y, rect.Max.X-r, rect.Max.Y)
	draw.Draw(img, inner, &image.Uniform{col}, image.Point{}, draw.Over)

	// Левый и правый прямоугольники
	left := image.Rect(rect.Min.X, rect.Min.Y+r, rect.Min.X+r, rect.Max.Y-r)
	right := image.Rect(rect.Max.X-r, rect.Min.Y+r, rect.Max.X, rect.Max.Y-r)
	draw.Draw(img, left, &image.Uniform{col}, image.Point{}, draw.Over)
	draw.Draw(img, right, &image.Uniform{col}, image.Point{}, draw.Over)

	// Углы
	drawCircle(img, rect.Min.X+r, rect.Min.Y+r, r, col)
	drawCircle(img, rect.Max.X-r-1, rect.Min.Y+r, r, col)
	drawCircle(img, rect.Min.X+r, rect.Max.Y-r-1, r, col)
	drawCircle(img, rect.Max.X-r-1, rect.Max.Y-r-1, r, col)
}

func drawCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	rr := r * r
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= rr {
				img.Set(cx+x, cy+y, col)
			}
		}
	}
}
