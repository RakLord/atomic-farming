// Command plantsheet renders a contact sheet of procedurally generated plants
// to a PNG, then exits.
//
// The procedural generator cannot be judged from test output — whether plants
// look good, vary enough, and grow smoothly is something you have to look at.
// This is the quick way to look at a lot of them at once; the in-game lab
// (press L) is the way to tune one.
//
//	go run ./cmd/plantsheet                      # a random population
//	go run ./cmd/plantsheet -mode growth         # one plant through its life
//	go run ./cmd/plantsheet -mode mutations      # a strain and its neighbours
//	go run ./cmd/plantsheet -mode archetypes     # every stem x leaf pairing
//
// It opens a window briefly because Ebitengine needs a graphics context to
// rasterise; it closes itself as soon as the sheet is written.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/render"
)

var (
	out    = flag.String("out", "plantsheet.png", "PNG file to write")
	mode   = flag.String("mode", "population", "population, growth, mutations, or archetypes")
	seed   = flag.Uint64("seed", 1, "base genome seed")
	cols   = flag.Int("cols", 8, "tiles per row")
	rows   = flag.Int("rows", 5, "rows of tiles")
	tilePx = flag.Int("tile", 150, "tile size in pixels")
)

// cell is one tile of the sheet: a plant at a maturity, with a caption.
type cell struct {
	genome   plant.Genome
	maturity float64
	caption  string
}

func main() {
	flag.Parse()

	cells := buildCells()
	if len(cells) == 0 {
		log.Fatalf("unknown mode %q", *mode)
	}

	s := &sheet{cells: cells, w: *cols * *tilePx, h: *rows * *tilePx}
	ebiten.SetWindowSize(s.w/2, s.h/2)
	ebiten.SetWindowTitle("plantsheet")
	if err := ebiten.RunGame(s); err != nil {
		log.Fatal(err)
	}
	if s.err != nil {
		log.Fatal(s.err)
	}
	fmt.Printf("wrote %s (%dx%d, %d plants)\n", *out, s.w, s.h, len(cells))
}

func buildCells() []cell {
	total := *cols * *rows
	var cells []cell

	switch *mode {
	case "population":
		for i := 0; i < total; i++ {
			g := plant.RandomGenome(*seed + uint64(i))
			cells = append(cells, cell{genome: g, maturity: 1, caption: g.Label()})
		}
	case "growth":
		g := plant.RandomGenome(*seed)
		for i := 0; i < total; i++ {
			t := float64(i) / float64(total-1)
			cells = append(cells, cell{genome: g, maturity: t, caption: fmt.Sprintf("t=%.2f", t)})
		}
	case "mutations":
		base := plant.RandomGenome(*seed)
		cells = append(cells, cell{genome: base, maturity: 1, caption: "parent " + base.Label()})
		for i := 1; i < total; i++ {
			m := plant.Mutate(base, plant.Hash64(uint64(i)), plant.BasisPoints/12)
			cells = append(cells, cell{genome: m, maturity: 1, caption: m.Label()})
		}
	case "archetypes":
		for stem := 0; stem < int(plant.StemArchetypeCount); stem++ {
			for leaf := 0; leaf < int(plant.LeafArchetypeCount); leaf++ {
				g := plant.RandomGenome(*seed)
				g[plant.GeneStemArchetype] = pairAt(stem, int(plant.StemArchetypeCount))
				g[plant.GeneLeafArchetype] = pairAt(leaf, int(plant.LeafArchetypeCount))
				cells = append(cells, cell{
					genome:   g,
					maturity: 1,
					caption:  fmt.Sprintf("%v/%v", plant.StemArchetype(stem), plant.LeafArchetype(leaf)),
				})
			}
		}
	}
	return cells
}

// pairAt returns a homozygous pair landing in the middle of archetype bucket i.
func pairAt(i, n int) plant.GenePair {
	v := plant.Allele((i*256 + 128) / n)
	return plant.GenePair{A: v, B: v}
}

type sheet struct {
	cells   []cell
	w, h    int
	sprites *render.SpriteCache
	written bool
	err     error
}

func (s *sheet) Update() error {
	if s.written {
		return ebiten.Termination
	}
	return nil
}

func (s *sheet) Draw(screen *ebiten.Image) {
	if s.written {
		return
	}
	if s.sprites == nil {
		// Big enough to hold the whole sheet, so nothing is rasterised twice.
		s.sprites = render.NewSpriteCache(len(s.cells) + 8)
	}

	screen.Fill(color.RGBA{0x14, 0x1c, 0x22, 0xff})
	for i, c := range s.cells {
		x := (i % *cols) * *tilePx
		y := (i / *cols) * *tilePx
		if y >= s.h {
			break
		}
		ground := y + *tilePx - 10
		vecFill(screen, x+1, y+1, *tilePx-2, *tilePx-2, color.RGBA{0x1b, 0x25, 0x2c, 0xff})
		vecFill(screen, x+1, ground, *tilePx-2, 9, color.RGBA{0x4a, 0x33, 0x22, 0xff})
		s.sprites.DrawPlantFitted(screen, plant.ExpressFull(c.genome), c.maturity,
			x+4, y+4, *tilePx-8, *tilePx-14)
	}

	s.err = s.write(screen)
	s.written = true
}

func (s *sheet) write(screen *ebiten.Image) error {
	buf := make([]byte, s.w*s.h*4)
	screen.ReadPixels(buf)

	img := image.NewRGBA(image.Rect(0, 0, s.w, s.h))
	copy(img.Pix, buf)

	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func (s *sheet) Layout(int, int) (int, int) { return s.w, s.h }

func vecFill(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	sub := ebiten.NewImage(w, h)
	sub.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(sub, op)
	sub.Deallocate()
}
