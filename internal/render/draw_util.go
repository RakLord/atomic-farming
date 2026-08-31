package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func fillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h int, thickness float32, c color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), thickness, c, false)
}

// drawText renders s with its top-left corner at (x, y).
func drawText(dst *ebiten.Image, s string, face text.Face, x, y int, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

// drawTextRight renders s so its right edge sits at x.
func drawTextRight(dst *ebiten.Image, s string, face text.Face, x, y int, c color.Color) {
	w, _ := text.Measure(s, face, 0)
	drawText(dst, s, face, x-int(w), y, c)
}

// drawTextCentered renders s horizontally centred on cx.
func drawTextCentered(dst *ebiten.Image, s string, face text.Face, cx, y int, c color.Color) {
	w, _ := text.Measure(s, face, 0)
	drawText(dst, s, face, cx-int(w)/2, y, c)
}

// textWidth measures a string in logical pixels.
func textWidth(s string, face text.Face) (w, h int) {
	fw, fh := text.Measure(s, face, 0)
	return int(fw), int(fh)
}
