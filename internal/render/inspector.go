package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// The plant inspector is a read-only look at one genome: every gene, what the
// plant expresses, what it is quietly carrying, and how far it sits from each
// strain it could become.
//
// It is not the lab. The lab expresses with plant.ExpressFull, which ignores
// species ranges — a Stem's Growth Rate lives in a [22,255] window, so the lab
// would show the wrong number for anything actually growing. The inspector
// expresses through the crop's own ranges, which is what the plant really is.

const (
	inspPad     = 24
	inspHeadH   = 58
	inspLeftX   = inspPad
	inspLeftW   = 396
	inspPrevY   = inspHeadH + 20
	inspPrevH   = 210
	inspStrainY = inspPrevY + inspPrevH + 96

	inspGenesX     = inspLeftX + inspLeftW + 26
	inspGenesW     = screenW - inspGenesX - inspPad
	inspGenesY     = inspHeadH + 24
	inspRowH       = 25
	inspRowsPerCol = 23

	// carrierGap is how far two alleles must differ before a gene counts as
	// hiding something worth breeding for. A difference of one is noise.
	carrierGap = 16
)

// inspected resolves what the inspector is currently looking at.
type inspected struct {
	kind      sim.CropKind
	genome    plant.Genome
	pheno     plant.Phenotype
	growth    sim.Growth
	stages    int
	hasGrowth bool
	ok        bool
}

func (g *Game) inspectedPlant() inspected {
	u := g.uiState
	if !u.InspectOpen {
		return inspected{}
	}
	out := inspected{kind: u.InspectKind, genome: u.InspectGenome, ok: true}
	out.pheno = sim.SeedPhenotype(sim.SeedStack{Kind: u.InspectKind, Genome: u.InspectGenome})

	if u.InspectFromPlot {
		plot, found := g.state.Grid.At(u.InspectPos)
		if found && plot.Crop != nil {
			out.growth, out.stages, out.hasGrowth = plot.Growth, plot.Crop.Stages(), true
		}
	}
	return out
}

func (g *Game) handleInspectorInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.uiState.CloseInspector()
		return
	}
	// A plot harvested while being looked at leaves nothing to look at.
	if g.uiState.InspectFromPlot {
		if plot, ok := g.state.Grid.At(g.uiState.InspectPos); !ok || plot.Crop == nil {
			g.uiState.CloseInspector()
			return
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.uiState.CloseInspector()
	}
}

func (g *Game) drawInspector(dst *ebiten.Image) {
	in := g.inspectedPlant()
	if !in.ok {
		return
	}

	fillRect(dst, 0, 0, screenW, screenH, colorLabBG)
	name := sim.StrainName(in.kind, in.genome, in.pheno, sim.CropDisplayName(in.kind))
	drawText(dst, "PLANT INSPECTOR", fontTitle, inspPad, 22, colorText)
	drawTextRight(dst, "click anywhere or press Esc to close", fontSmall, screenW-inspPad, 28, colorTextMuted)
	drawText(dst, name, fontBody, inspPad+250, 26, inspectorNameColor(in))

	g.drawInspectorSpecimen(dst, in)
	g.drawInspectorGenes(dst, in)
	g.drawInspectorStrains(dst, in)
}

func inspectorNameColor(in inspected) (c colorRGBA) {
	if strain, ok := sim.IdentifyStrain(in.kind, in.genome, in.pheno); ok {
		return rarityColor(strain.Rarity)
	}
	return colorText
}

func (g *Game) drawInspectorSpecimen(dst *ebiten.Image, in inspected) {
	maturity := 1.0
	if in.hasGrowth {
		maturity = in.growth.Maturity(in.stages)
	}

	fillRect(dst, inspLeftX, inspPrevY, inspLeftW, inspPrevH, colorSky)
	fillRect(dst, inspLeftX, inspPrevY+inspPrevH-12, inspLeftW, 12, colorSoil)
	g.sprites.DrawPlantFitted(dst, in.pheno, maturity, inspLeftX, inspPrevY, inspLeftW, inspPrevH-12)
	strokeRect(dst, inspLeftX, inspPrevY, inspLeftW, inspPrevH, 1, colorDivider)

	y := inspPrevY + inspPrevH + 16
	drawText(dst, sim.CropDisplayName(in.kind)+"  ·  strain "+in.genome.Label(), fontBody, inspLeftX, y, colorText)
	y += 22

	switch {
	case in.hasGrowth && in.growth.Ready:
		drawText(dst, "ready to harvest", fontSmall, inspLeftX, y, colorReady)
	case in.hasGrowth:
		drawText(dst, fmt.Sprintf("growing — %d%% (stage %d of %d)",
			int(in.growth.Maturity(in.stages)*100), in.growth.Stage+1, in.stages),
			fontSmall, inspLeftX, y, colorTextMuted)
	default:
		drawText(dst, "an unsown seed, shown at full growth", fontSmall, inspLeftX, y, colorTextMuted)
	}

	y += 20
	drawText(dst, fmt.Sprintf("%d genes carry a hidden allele", carrierCount(in.genome)),
		fontSmall, inspLeftX, y, colorCarrier)
}

// drawConditionMark is a filled box when a requirement is met and a hollow one
// when it is not.
func drawConditionMark(dst *ebiten.Image, x, y int, met bool, c colorRGBA) {
	const size = 8
	if met {
		fillRect(dst, x, y, size, size, c)
		return
	}
	strokeRect(dst, x, y, size, size, 1, c)
}

// carrierCount is how many genes hold two meaningfully different alleles —
// the ones with something to give that the plant is not showing.
func carrierCount(g plant.Genome) int {
	n := 0
	for i := 0; i < plant.GeneCount; i++ {
		if alleleGap(g[i]) >= carrierGap {
			n++
		}
	}
	return n
}

func alleleGap(pair plant.GenePair) int {
	d := int(pair.A) - int(pair.B)
	if d < 0 {
		return -d
	}
	return d
}

func (g *Game) drawInspectorGenes(dst *ebiten.Image, in inspected) {
	colW := inspGenesW / 2
	drawText(dst, "GENOME", fontBody, inspGenesX, inspHeadH-4, colorTextMuted)
	drawTextRight(dst, "alleles → expressed", fontSmall, inspGenesX+inspGenesW, inspHeadH, colorTextMuted)

	for i := 0; i < plant.GeneCount; i++ {
		col, row := i/inspRowsPerCol, i%inspRowsPerCol
		x := inspGenesX + col*colW
		y := inspGenesY + row*inspRowH

		spec := plant.GeneCatalog[i]
		pair := in.genome[i]

		fillRect(dst, x, y+3, 3, 12, groupColors[spec.Group])
		drawText(dst, spec.Name, fontSmall, x+10, y, colorText)

		// A gene showing 140 while carrying a 240 is the most useful thing the
		// diploid model can tell a breeder, and it is invisible everywhere else
		// in the game.
		pairColor := colorTextMuted
		if alleleGap(pair) >= carrierGap {
			pairColor = colorCarrier
		}
		drawTextRight(dst, fmt.Sprintf("%3d/%-3d", pair.A, pair.B), fontSmall, x+colW-58, y, pairColor)
		drawTextRight(dst, fmt.Sprintf("%3d", in.pheno[i]), fontSmall, x+colW-16, y, colorText)
	}
}

func (g *Game) drawInspectorStrains(dst *ebiten.Image, in inspected) {
	progress := sim.StrainProgressFor(in.kind, in.genome, in.pheno)
	y := inspStrainY
	drawText(dst, "STRAINS", fontBody, inspLeftX, y, colorTextMuted)
	y += 24

	if len(progress) == 0 {
		drawText(dst, "no known strains for this crop", fontSmall, inspLeftX, y, colorTextMuted)
		return
	}

	for _, sp := range progress {
		label := sp.Strain.Name
		if sp.Met {
			label += "   — this plant"
		}
		drawText(dst, label, fontSmall, inspLeftX, y, rarityColor(sp.Strain.Rarity))
		drawTextRight(dst, sp.Strain.Rarity.String(), fontSmall, inspLeftX+inspLeftW, y, colorTextMuted)
		y += 17

		if !sp.Breedable {
			drawText(dst, "   cannot be bred — sold under licence", fontSmall, inspLeftX, y, colorTextMuted)
			y += 19
			continue
		}
		for _, c := range sp.Conditions {
			// A drawn marker rather than a tick glyph: the UI font has no
			// U+2713 and renders it as tofu.
			tint := colorUnmet
			if c.Met {
				tint = colorMet
			}
			drawConditionMark(dst, inspLeftX+6, y+4, c.Met, tint)
			drawText(dst, c.Condition.GeneName(), fontSmall, inspLeftX+22, y, colorTextMuted)
			drawTextRight(dst, fmt.Sprintf("%d / %s", c.Current, c.Condition.Requirement()),
				fontSmall, inspLeftX+inspLeftW, y, tint)
			y += 17
		}
		y += 4
	}
}
