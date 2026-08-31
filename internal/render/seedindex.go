package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/plant"
	"atomicfarming/internal/sim"
)

// The seed index is the picker for choosing between lines of one species.
//
// It shows each line as a rendered plant rather than a hex code, because that
// is the difference between choosing a seed and choosing a string. The sprite
// cache the farm and the lab already use makes the previews free.

const (
	indexW     = 820
	indexH     = 560
	indexX     = (screenW - indexW) / 2
	indexY     = (screenH - indexH) / 2
	indexHeadH = 56
	indexFootH = 34
	indexRowH  = 84
	indexPad   = 18
	indexRows  = (indexH - indexHeadH - indexFootH) / indexRowH
	indexBtnW  = 84
	indexBtnH  = 26
	indexPrevW = 62
)

type indexRow struct {
	rect    rect
	sow     rect
	discard rect
	stack   int
}

type indexLayout struct {
	rows  []indexRow
	total int
}

// seedIndexLayout computes the visible rows. Draw and click both use it, so a
// button cannot render in one place and respond in another.
func (g *Game) seedIndexLayout() indexLayout {
	var l indexLayout
	stacks := g.state.Inventory.StacksOfKind(g.uiState.SeedIndexKind)
	l.total = len(stacks)

	from := g.uiState.SeedIndexScroll
	if from > len(stacks) {
		from = len(stacks)
	}
	y := indexY + indexHeadH
	for i := from; i < len(stacks) && len(l.rows) < indexRows; i++ {
		r := rect{indexX + indexPad, y, indexW - 2*indexPad, indexRowH - 6}
		btnX := r.x + r.w - indexBtnW - 10
		l.rows = append(l.rows, indexRow{
			rect:    r,
			sow:     rect{btnX, r.y + 10, indexBtnW, indexBtnH},
			discard: rect{btnX, r.y + 10 + indexBtnH + 6, indexBtnW, indexBtnH},
			stack:   stacks[i],
		})
		y += indexRowH
	}
	return l
}

func (g *Game) handleSeedIndexClick(mx, my int) {
	// A click outside the modal dismisses it.
	if !(rect{indexX, indexY, indexW, indexH}).contains(mx, my) {
		g.uiState.CloseSeedIndex()
		return
	}
	for _, row := range g.seedIndexLayout().rows {
		switch {
		case row.sow.contains(mx, my):
			g.selectSeed(row.stack)
			g.uiState.CloseSeedIndex()
			return
		case row.discard.contains(mx, my):
			g.discardSeed(row.stack)
			return
		}
	}
}

func (g *Game) drawSeedIndex(dst *ebiten.Image) {
	fillRect(dst, 0, 0, screenW, screenH, colorOverlay)
	fillRect(dst, indexX, indexY, indexW, indexH, colorLabPanel)
	strokeRect(dst, indexX, indexY, indexW, indexH, 1, colorDivider)

	l := g.seedIndexLayout()
	title := "SEED INDEX — " + sim.CropDisplayName(g.uiState.SeedIndexKind)
	drawText(dst, title, fontTitle, indexX+indexPad, indexY+16, colorText)
	drawTextRight(dst, fmt.Sprintf("%d lines", l.total), fontBody,
		indexX+indexW-indexPad, indexY+22, colorTextMuted)

	for _, row := range l.rows {
		g.drawIndexRow(dst, row)
	}

	footer := "click a line to sow it, or discard one you are done with   ·   Esc to close"
	if l.total > indexRows {
		footer = fmt.Sprintf("showing %d–%d of %d   ·   scroll to see more   ·   Esc to close",
			g.uiState.SeedIndexScroll+1, g.uiState.SeedIndexScroll+len(l.rows), l.total)
	}
	drawText(dst, footer, fontSmall, indexX+indexPad, indexY+indexH-indexFootH+10, colorTextMuted)
}

func (g *Game) drawIndexRow(dst *ebiten.Image, row indexRow) {
	stack := g.state.Inventory.Stacks[row.stack]
	pheno := sim.SeedPhenotype(stack)
	selected := g.uiState.IsSeedSelected(stack)

	bg := colorRowBG
	if selected {
		bg = colorRowSelected
	}
	fillRect(dst, row.rect.x, row.rect.y, row.rect.w, row.rect.h, bg)

	// The preview: what this line actually grows into. DrawPlant rather than
	// DrawPlantFitted, because fitting the whole canvas would shrink the plant
	// to fill margins that are mostly empty — a thumbnail wants the plant at
	// full height with the empty sides clipped away.
	fillRect(dst, row.rect.x+6, row.rect.y+6, indexPrevW, row.rect.h-12, colorSky)
	g.sprites.DrawPlant(dst, pheno, 1, row.rect.x+6, row.rect.y+6, indexPrevW, row.rect.h-12)

	textX := row.rect.x + indexPrevW + 20
	name := sim.SeedStrainName(stack)
	label := colorText
	if strain, ok := sim.IdentifyStrain(stack.Kind, stack.Genome, pheno); ok {
		label = rarityColor(strain.Rarity)
	}
	drawText(dst, name, fontBody, textX, row.rect.y+10, label)
	drawText(dst, stack.Genome.Label(), fontSmall, textX, row.rect.y+30, colorTextMuted)

	// The three stats that decide whether a line is worth keeping.
	stats := fmt.Sprintf("Density %d    Growth %d    Yield %d",
		pheno.Get(plant.GeneDensity), pheno.Get(plant.GeneGrowthRate), pheno.Get(plant.GeneYieldAmount))
	drawText(dst, stats, fontSmall, textX, row.rect.y+50, colorTextMuted)

	drawTextRight(dst, fmt.Sprintf("x%d", stack.Count), fontBody,
		row.sow.x-16, row.rect.y+10, colorTextMuted)

	drawButton(dst, row.sow, "Sow", colorRowBuyable, colorText)
	drawButton(dst, row.discard, "Discard", colorRowBG, colorWarning)
}

func drawButton(dst *ebiten.Image, r rect, label string, bg, fg color.Color) {
	fillRect(dst, r.x, r.y, r.w, r.h, bg)
	strokeRect(dst, r.x, r.y, r.w, r.h, 1, colorDivider)
	drawTextCentered(dst, label, fontSmall, r.x+r.w/2, r.y+6, fg)
}
