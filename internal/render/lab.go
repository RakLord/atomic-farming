package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"atomicfarming/internal/plant"
)

// The genetics lab is a development tool first and the seed of the in-game
// lab screen second. The procedural generator cannot be tuned by reading test
// output, so this exists to look at it: scrub a plant through its life, step
// one allele and watch the shape move a little, and breed two specimens to
// see the offspring.

const (
	labPreviewX = 20
	labPreviewY = 78
	labPreviewW = 340
	labPreviewH = 380

	labSliderY = labPreviewY + labPreviewH + 22
	labSliderH = 10

	labGridX    = 380
	labGridY    = labPreviewY
	labGridCell = 112

	labGenesX     = 736
	labGenesY     = labPreviewY
	labGenesW     = screenW - labGenesX - 20
	labRowH       = 25
	labRowsPerCol = 23
)

var (
	colorLabBG     = color.RGBA{0x0a, 0x0f, 0x0c, 0xff}
	colorLabPanel  = color.RGBA{0x14, 0x1c, 0x16, 0xff}
	colorLabSel    = color.RGBA{0x2e, 0x44, 0x30, 0xff}
	colorLabSlider = color.RGBA{0x8e, 0xd9, 0x6a, 0xff}
	colorSky       = color.RGBA{0x18, 0x22, 0x2c, 0xff}

	groupColors = [plant.GroupCount]color.RGBA{
		{0x6f, 0xa8, 0x54, 0xff}, // Stem
		{0x4f, 0xbf, 0x82, 0xff}, // Foliage
		{0xd2, 0x7a, 0xd8, 0xff}, // Flower
		{0xe0, 0x7a, 0x4a, 0xff}, // Fruit
		{0x77, 0x88, 0x99, 0xff}, // Noise
		{0x5a, 0x9e, 0xd8, 0xff}, // Vigour
		{0xd8, 0xc0, 0x4a, 0xff}, // Yield
		{0xb0, 0x8a, 0xe0, 0xff}, // Meta
	}
)

// handleLabInput drives the lab. It runs instead of farm input while the lab
// is open, so a stray click cannot disturb the farm behind it.
func (g *Game) handleLabInput() {
	l := g.lab

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyL) {
		l.Open = false
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		l.SelectGene(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		l.SelectGene(-1)
	}

	step := 1
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		step = 10
	}
	secondAllele := ebiten.IsKeyPressed(ebiten.KeyShift)
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		l.AdjustGene(step, secondAllele)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		l.AdjustGene(-step, secondAllele)
	}

	// Held, not just-pressed: scrubbing maturity wants to be continuous.
	if ebiten.IsKeyPressed(ebiten.KeyBracketRight) {
		l.ScrubMaturity(0.01)
	}
	if ebiten.IsKeyPressed(ebiten.KeyBracketLeft) {
		l.ScrubMaturity(-0.01)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		l.AutoGrow = !l.AutoGrow
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		l.Randomise()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		l.Mutate()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit0) {
		l.Reset()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		l.StoreParentA()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		l.StoreParentB()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		l.Breed()
	}
}

func (g *Game) drawLab(dst *ebiten.Image) {
	l := g.lab
	pheno := plant.ExpressFull(l.Genome)

	fillRect(dst, 0, 0, screenW, screenH, colorLabBG)
	drawText(dst, "GENETICS LAB", fontTitle, 20, 24, colorText)
	drawTextRight(dst, "L or Esc to close", fontSmall, screenW-20, 30, colorTextMuted)

	g.drawLabSpecimen(dst, pheno)
	g.drawLabMutations(dst)
	g.drawLabGenes(dst, pheno)
	g.drawLabKeys(dst)
}

func (g *Game) drawLabSpecimen(dst *ebiten.Image, pheno plant.Phenotype) {
	l := g.lab

	fillRect(dst, labPreviewX, labPreviewY, labPreviewW, labPreviewH, colorSky)
	// A soil band, so the plant is visibly standing on something.
	fillRect(dst, labPreviewX, labPreviewY+labPreviewH-14, labPreviewW, 14, colorSoil)
	g.sprites.DrawPlantFitted(dst, pheno, l.Maturity, labPreviewX, labPreviewY, labPreviewW, labPreviewH-14)
	strokeRect(dst, labPreviewX, labPreviewY, labPreviewW, labPreviewH, 1, colorDivider)

	// Maturity slider.
	fillRect(dst, labPreviewX, labSliderY, labPreviewW, labSliderH, colorLabPanel)
	fillRect(dst, labPreviewX, labSliderY, int(float64(labPreviewW)*l.Maturity), labSliderH, colorLabSlider)
	label := fmt.Sprintf("maturity %.2f", l.Maturity)
	if l.AutoGrow {
		label += "  (growing)"
	}
	drawText(dst, label, fontSmall, labPreviewX, labSliderY+16, colorTextMuted)

	y := labSliderY + 38
	drawText(dst, "Strain "+l.Genome.Label(), fontBody, labPreviewX, y, colorText)
	y += 22
	for _, line := range []string{
		fmt.Sprintf("stem    %v", pheno.StemArchetype()),
		fmt.Sprintf("leaf    %v", pheno.LeafArchetype()),
		fmt.Sprintf("flower  %v", pheno.FlowerArchetype()),
		fmt.Sprintf("fruit   %v", pheno.FruitArchetype()),
	} {
		drawText(dst, line, fontSmall, labPreviewX, y, colorTextMuted)
		y += 16
	}

	y += 6
	parents := "parents: "
	if l.HasParentA {
		parents += "A " + l.ParentA.Label()
	} else {
		parents += "A —"
	}
	if l.HasParentB {
		parents += "   B " + l.ParentB.Label()
	} else {
		parents += "   B —"
	}
	drawText(dst, parents, fontSmall, labPreviewX, y, colorTextMuted)
}

// drawLabMutations renders one-step neighbours of the specimen. Seeing them
// together is how the locality property is judged by eye: every tile should
// read as the same plant, slightly varied.
func (g *Game) drawLabMutations(dst *ebiten.Image) {
	drawText(dst, "ONE MUTATION AWAY", fontSmall, labGridX, labGridY-14, colorTextMuted)

	previews := g.lab.MutationPreviews()
	for i, genome := range previews {
		col, row := i%3, i/3
		x := labGridX + col*labGridCell
		y := labGridY + row*labGridCell

		fillRect(dst, x, y, labGridCell-6, labGridCell-6, colorSky)
		fillRect(dst, x, y+labGridCell-6-6, labGridCell-6, 6, colorSoil)
		g.sprites.DrawPlantFitted(dst, plant.ExpressFull(genome), g.lab.Maturity, x, y, labGridCell-6, labGridCell-12)
		strokeRect(dst, x, y, labGridCell-6, labGridCell-6, 1, colorDivider)
	}

	hits, misses := g.sprites.Stats()
	drawText(dst, fmt.Sprintf("sprite cache: %d held, %d hits, %d rasterised", g.sprites.Len(), hits, misses),
		fontSmall, labGridX, labGridY+3*labGridCell+8, colorTextMuted)
}

func (g *Game) drawLabGenes(dst *ebiten.Image, pheno plant.Phenotype) {
	l := g.lab
	colW := labGenesW / 2

	for i := 0; i < plant.GeneCount; i++ {
		col, row := i/labRowsPerCol, i%labRowsPerCol
		x := labGenesX + col*colW
		y := labGenesY + row*labRowH

		spec := plant.GeneCatalog[i]
		pair := l.Genome[i]

		if plant.GeneID(i) == l.Selected {
			fillRect(dst, x-4, y-3, colW-8, labRowH-2, colorLabSel)
		}
		fillRect(dst, x, y+3, 3, 12, groupColors[spec.Group])

		drawText(dst, spec.Name, fontSmall, x+10, y, colorText)
		drawTextRight(dst, fmt.Sprintf("%3d/%-3d %3d", pair.A, pair.B, pheno[i]),
			fontSmall, x+colW-16, y, colorTextMuted)
	}

	if l.Selected.Valid() {
		spec := plant.GeneCatalog[l.Selected]
		drawText(dst, fmt.Sprintf("%s  ·  %s group  ·  %s expression", spec.Name, spec.Group, spec.Expr),
			fontSmall, labGenesX, labGenesY+labRowsPerCol*labRowH+10, colorText)
	}
}

func (g *Game) drawLabKeys(dst *ebiten.Image) {
	keys := "up/down select gene   left/right adjust (shift: allele B, ctrl: x10)   " +
		"[ ] maturity   space grow   R random   M mutate   1/2 store parent   B breed   0 reset"
	drawText(dst, keys, fontSmall, 20, screenH-24, colorTextMuted)
}
