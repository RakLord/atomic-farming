package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"atomicfarming/internal/bignum"
	"atomicfarming/internal/sim"
)

// Panel geometry. Row rectangles are computed once by panelLayout and used by
// both the draw and the click paths, so the two cannot drift apart — a button
// that draws in one place and responds in another is a nasty class of bug.
const (
	panelPadX   = 22
	panelInnerW = panelW - 2*panelPadX
	offerRowH   = 40
	seedRowH    = 30
	// Row buttons: the seed index, and the auto-pick toggle.
	seedBtnSize  = 20
	seedBtnGap   = 4
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
	index rect // opens the seed picker
	auto  rect // toggles auto-pick for this species
	group int  // index into panelLayout.groups
	// buttons is false for a group with a single line, where a picker and a
	// pick-for-me toggle would both be meaningless.
	buttons bool
}

type panelLayout struct {
	offers   []offerRow
	unlocks  []unlockRow
	seeds    []seedRow
	groups   []sim.SeedGroup
	seedsY   int
	unlocksY int
	plotY    int
	hidden   int
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
	// Rows are groups, not stacks: a species' unnamed lines collapse into one
	// row so ordinary drift never fills the panel, while named strains stay
	// visible in their own right.
	l.groups = g.state.GroupSeeds()
	for i := range l.groups {
		if len(l.seeds) >= maxSeedRows {
			l.hidden = len(l.groups) - len(l.seeds)
			break
		}
		r := rect{x, y, panelInnerW, seedRowH - 4}
		btnY := r.y + (r.h-seedBtnSize)/2
		autoX := r.x + r.w - 6 - seedBtnSize
		indexX := autoX - seedBtnGap - seedBtnSize
		l.seeds = append(l.seeds, seedRow{
			rect:    r,
			index:   rect{indexX, btnY, seedBtnSize, seedBtnSize},
			auto:    rect{autoX, btnY, seedBtnSize, seedBtnSize},
			group:   i,
			buttons: len(l.groups[i].Stacks) > 1,
		})
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
		group := l.groups[row.group]
		// Buttons are tested before the row body, or a click on one would also
		// fall through and queue a seed.
		if row.buttons {
			switch {
			case row.index.contains(mx, my):
				g.openSeedIndex(group.Kind)
				return
			case row.auto.contains(mx, my):
				g.toggleSeedAutoSelect(group.Kind)
				return
			}
		}
		if row.rect.contains(mx, my) {
			g.clickSeedGroup(group)
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
		g.drawSeedRow(dst, l.groups[row.group], row)
	}
	// Tooltips last, so they sit over the rows beneath them.
	g.drawSeedRowTooltip(dst, l)
	if l.hidden > 0 {
		drawTextRight(dst, fmt.Sprintf("+%d more", l.hidden), fontSmall,
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

func (g *Game) drawSeedRow(dst *ebiten.Image, group sim.SeedGroup, row seedRow) {
	r := row.rect
	selected := false
	for _, i := range group.Stacks {
		if i < len(g.state.Inventory.Stacks) && g.uiState.IsSeedSelected(g.state.Inventory.Stacks[i]) {
			selected = true
			break
		}
	}

	bg := colorRowBG
	if selected {
		bg = colorRowSelected
	}
	fillRect(dst, r.x, r.y, r.w, r.h, bg)

	label := colorText
	if group.Named {
		label = rarityColor(group.Strain.Rarity)
	}
	drawText(dst, group.Name, fontBody, r.x+10, r.y+6, label)

	if len(group.Stacks) > 1 {
		w, _ := textWidth(group.Name, fontBody)
		drawText(dst, fmt.Sprintf("%d lines", len(group.Stacks)), fontSmall,
			r.x+18+w, r.y+8, colorTextMuted)
	}

	countRight := r.x + r.w - 10
	if row.buttons {
		countRight = row.index.x - 10
		auto := g.state.AutoSelectSeeds(group.Kind)
		mx, my := ebiten.CursorPosition()

		drawIconButton(dst, row.index, false, row.index.contains(mx, my))
		drawListIcon(dst, row.index, colorText)

		drawIconButton(dst, row.auto, auto, row.auto.contains(mx, my))
		drawAutoIcon(dst, row.auto, auto)
	}
	drawTextRight(dst, fmt.Sprintf("x%d", group.Count), fontBody, countRight, r.y+6, colorTextMuted)
}

// seedRowTooltip is the label for whichever row button sits under (mx, my).
//
// The decision is split from the drawing so it can be tested: a cursor cannot
// be moved in a headless test, and a tooltip that names the wrong button — or
// a button whose hit box has drifted from where it is drawn — would otherwise
// only be caught by someone noticing.
func (g *Game) seedRowTooltip(l panelLayout, mx, my int) (label string, anchor rect, ok bool) {
	for _, row := range l.seeds {
		if !row.buttons {
			continue
		}
		switch {
		case row.index.contains(mx, my):
			return "See every line of this crop", row.index, true
		case row.auto.contains(mx, my):
			if g.state.AutoSelectSeeds(l.groups[row.group].Kind) {
				return "Auto-pick on — click to choose each time", row.auto, true
			}
			return "Auto-pick off — click to always sow your bulk line", row.auto, true
		}
	}
	return "", rect{}, false
}

// drawSeedRowTooltip labels whichever row button the cursor is over. Two
// unlabelled icons are exactly the kind of thing a player should not have to
// click to find out about.
func (g *Game) drawSeedRowTooltip(dst *ebiten.Image, l panelLayout) {
	mx, my := ebiten.CursorPosition()
	label, anchor, ok := g.seedRowTooltip(l, mx, my)
	if !ok {
		return
	}

	w, _ := textWidth(label, fontSmall)
	box := rect{anchor.x + anchor.w - w - 16, anchor.y - 24, w + 16, 20}
	if box.x < panelX+4 {
		box.x = panelX + 4
	}
	fillRect(dst, box.x, box.y, box.w, box.h, colorTooltip)
	strokeRect(dst, box.x, box.y, box.w, box.h, 1, colorDivider)
	drawText(dst, label, fontSmall, box.x+8, box.y+4, colorText)
}

func drawIconButton(dst *ebiten.Image, r rect, active, hovered bool) {
	bg := colorRowBG
	if active {
		bg = colorToggleOn
	}
	fillRect(dst, r.x, r.y, r.w, r.h, bg)
	edge := colorDivider
	if hovered {
		edge = colorPlotPick
	}
	strokeRect(dst, r.x, r.y, r.w, r.h, 1, edge)
}

// drawListIcon is the seed index: a bulleted list of the lines you hold.
func drawListIcon(dst *ebiten.Image, r rect, c color.Color) {
	for i := 0; i < 3; i++ {
		y := r.y + 5 + i*5
		fillRect(dst, r.x+4, y, 2, 2, c)
		fillRect(dst, r.x+8, y, r.w-13, 2, c)
	}
}

// drawAutoIcon is the auto-pick toggle: a tick, lit when the row sows without
// asking.
func drawAutoIcon(dst *ebiten.Image, r rect, on bool) {
	c := colorIconOff
	if on {
		c = colorCash
	}
	cx, cy := r.x+r.w/2, r.y+r.h/2
	strokeLine(dst, cx-5, cy, cx-2, cy+4, 2, c)
	strokeLine(dst, cx-2, cy+4, cx+5, cy-4, 2, c)
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
