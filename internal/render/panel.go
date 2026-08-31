package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/sim"
)

// Panel geometry. Row rectangles are computed once by panelLayout and used by
// both the draw and the click paths, so the two cannot drift apart — a button
// that draws in one place and responds in another is a nasty class of bug.
const (
	panelPadX    = 22
	panelInnerW  = panelW - 2*panelPadX
	offerRowH    = 40
	seedRowH     = 26
	sectionGap   = 14
	maxSeedRows  = 7
	noticeHeight = 26
)

type rect struct{ x, y, w, h int }

func (r rect) contains(px, py int) bool {
	return px >= r.x && px < r.x+r.w && py >= r.y && py < r.y+r.h
}

type offerRow struct {
	rect rect
	id   sim.SeedOfferID
}

type unlockRow struct {
	rect rect
	id   sim.UnlockID
}

type seedRow struct {
	rect  rect
	index int
}

type panelLayout struct {
	offers    []offerRow
	unlocks   []unlockRow
	seeds     []seedRow
	seedsY    int
	unlocksY  int
	plotY     int
	hiddenBad int
}

// panelLayout computes every clickable row for the current state.
func (g *Game) panelLayout() panelLayout {
	var l panelLayout
	x := panelX + panelPadX
	y := panelY + 54

	for _, id := range sim.SeedShopOrder {
		if !sim.SeedOfferAvailable(g.state, id) {
			continue
		}
		l.offers = append(l.offers, offerRow{rect{x, y, panelInnerW, offerRowH - 4}, id})
		y += offerRowH
	}

	// Unlocks are listed only while unowned: once bought they become a shop
	// entry, and a permanently greyed row is just noise.
	var pending []sim.UnlockID
	for _, id := range sim.UnlockCatalogOrder {
		if !sim.IsUnlocked(g.state, id) {
			pending = append(pending, id)
		}
	}
	if len(pending) > 0 {
		y += sectionGap
		l.unlocksY = y
		y += 22
		for _, id := range pending {
			l.unlocks = append(l.unlocks, unlockRow{rect{x, y, panelInnerW, offerRowH - 4}, id})
			y += offerRowH
		}
	}

	y += sectionGap
	l.seedsY = y
	y += 22
	stacks := g.state.Inventory.Stacks
	for i := range stacks {
		if len(l.seeds) >= maxSeedRows {
			l.hiddenBad = len(stacks) - len(l.seeds)
			break
		}
		l.seeds = append(l.seeds, seedRow{rect{x, y, panelInnerW, seedRowH - 3}, i})
		y += seedRowH
	}

	y += sectionGap
	l.plotY = y
	return l
}

// handlePanelClick routes a click in the right-hand column.
func (g *Game) handlePanelClick(mx, my int) {
	l := g.panelLayout()
	for _, row := range l.offers {
		if row.rect.contains(mx, my) {
			g.buySeed(row.id)
			return
		}
	}
	for _, row := range l.unlocks {
		if row.rect.contains(mx, my) {
			g.buyUnlock(row.id)
			return
		}
	}
	for _, row := range l.seeds {
		if row.rect.contains(mx, my) {
			g.selectSeed(row.index)
			return
		}
	}
}

func (g *Game) drawPanel(dst *ebiten.Image) {
	fillRect(dst, panelX, panelY, panelW, panelH, colorPanelBG)
	fillRect(dst, panelX, panelY, 1, panelH, colorDivider)

	l := g.panelLayout()
	x := panelX + panelPadX

	drawText(dst, "SHOP", fontBody, x, panelY+26, colorTextMuted)
	drawTextRight(dst, fmt.Sprintf("%d seeds", g.state.Inventory.Total()),
		fontBody, panelX+panelW-panelPadX, panelY+26, colorTextMuted)

	for _, row := range l.offers {
		offer := sim.SeedShop[row.id]
		cost := sim.SeedCost(g.state, row.id)
		g.drawBuyRow(dst, row.rect, offer.Name, offer.Description, cost, sim.CanBuySeed(g.state, row.id))
	}

	if len(l.unlocks) > 0 {
		drawText(dst, "UPGRADES", fontBody, x, l.unlocksY, colorTextMuted)
		for _, row := range l.unlocks {
			u := sim.UnlockCatalog[row.id]
			g.drawBuyRow(dst, row.rect, u.Name, u.Description, u.Cost, sim.CanPurchaseUnlock(g.state, row.id))
		}
	}

	drawText(dst, "SEEDS", fontBody, x, l.seedsY, colorTextMuted)
	if len(l.seeds) == 0 {
		drawText(dst, "None — buy one above", fontSmall, x, l.seedsY+24, colorTextMuted)
	}
	for _, row := range l.seeds {
		g.drawSeedRow(dst, row)
	}
	if l.hiddenBad > 0 {
		drawTextRight(dst, fmt.Sprintf("+%d more", l.hiddenBad), fontSmall,
			panelX+panelW-panelPadX, l.seedsY, colorTextMuted)
	}

	g.drawPlotDetail(dst, l.plotY)
	g.drawNotice(dst)
}

func (g *Game) drawBuyRow(dst *ebiten.Image, r rect, name, desc string, cost bignum.Decimal, affordable bool) {
	bg, label := colorRowBG, colorTextMuted
	if affordable {
		bg, label = colorRowBuyable, colorText
	}
	fillRect(dst, r.x, r.y, r.w, r.h, bg)
	if r.contains(ebiten.CursorPosition()) && affordable {
		strokeRect(dst, r.x, r.y, r.w, r.h, 1, colorPlotPick)
	}

	drawText(dst, name, fontBody, r.x+10, r.y+5, label)
	if desc != "" {
		drawText(dst, truncate(desc, 46), fontSmall, r.x+10, r.y+21, colorTextMuted)
	}
	price := "$" + cost.Format(bignum.DisplayShort, 2)
	priceColor := colorTextMuted
	if affordable {
		priceColor = colorCash
	}
	drawTextRight(dst, price, fontBody, r.x+r.w-10, r.y+11, priceColor)
}

func (g *Game) drawSeedRow(dst *ebiten.Image, row seedRow) {
	stack := g.state.Inventory.Stacks[row.index]
	selected := g.uiState.IsSeedSelected(stack)

	bg := colorRowBG
	if selected {
		bg = colorRowSelected
	}
	fillRect(dst, row.rect.x, row.rect.y, row.rect.w, row.rect.h, bg)

	name := sim.SeedStrainName(stack)
	label := colorText
	if strain, ok := sim.IdentifyStrain(stack.Kind, stack.Genome, sim.SeedPhenotype(stack)); ok {
		label = rarityColor(strain.Rarity)
	}
	drawText(dst, name, fontBody, row.rect.x+10, row.rect.y+4, label)
	// Unnamed stacks are all called after their species, so show the strain
	// code too — otherwise a barn full of drifting seeds is six rows of "Stem".
	if name == sim.CropDisplayName(stack.Kind) {
		w, _ := textWidth(name, fontBody)
		drawText(dst, stack.Genome.Label(), fontSmall, row.rect.x+18+w, row.rect.y+6, colorTextMuted)
	}
	drawTextRight(dst, fmt.Sprintf("x%d", stack.Count), fontBody,
		row.rect.x+row.rect.w-10, row.rect.y+4, colorTextMuted)
}

func (g *Game) drawPlotDetail(dst *ebiten.Image, y int) {
	x := panelX + panelPadX
	fillRect(dst, x, y, panelInnerW, 1, colorDivider)
	y += 16

	if !g.uiState.HasSelection {
		drawText(dst, "Click a plot to sow or gather", fontSmall, x, y, colorTextMuted)
		return
	}

	pos := g.uiState.Selected
	plot, ok := g.state.Grid.At(pos)
	if !ok {
		return
	}
	drawText(dst, fmt.Sprintf("Plot (%d, %d)", pos.X, pos.Y), fontBody, x, y, colorTextMuted)
	y += 22

	if plot.Crop == nil {
		drawText(dst, "Empty", fontBody, x, y, colorText)
		return
	}

	name := g.state.PlotStrainName(pos)
	colour := colorText
	if strain, found := sim.IdentifyStrain(plot.Crop.Kind(), plot.Genome, plot.Phenotype); found {
		colour = rarityColor(strain.Rarity)
	}
	drawText(dst, name, fontBody, x, y, colour)
	y += 20

	maturity := plot.Growth.Maturity(plot.Crop.Stages())
	status := fmt.Sprintf("growing — %d%%", int(maturity*100))
	if plot.Growth.Ready {
		status = "ready to harvest"
	}
	drawText(dst, status, fontSmall, x, y, colorTextMuted)
	y += 18
	drawText(dst, "Strain "+plot.Genome.Label(), fontSmall, x, y, colorTextMuted)
}

func (g *Game) drawNotice(dst *ebiten.Image) {
	if g.uiState.Notice == "" {
		return
	}
	y := panelY + panelH - noticeHeight
	fillRect(dst, panelX, y, panelW, noticeHeight, colorRowSelected)
	drawText(dst, truncate(g.uiState.Notice, 58), fontSmall, panelX+panelPadX, y+7, colorText)
}

// plural renders a count with the right noun form.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func plantedCount(g *sim.Grid) int {
	n := 0
	for _, p := range g.Plots {
		if !p.IsEmpty() {
			n++
		}
	}
	return n
}
