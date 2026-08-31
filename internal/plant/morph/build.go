package morph

import (
	"math"
	"sort"

	"atomicfarming/internal/plant"
)

// Growth windows, in units of overall maturity t. Every part of the plant
// fades in over its own window, which is what makes one Build function cover
// every growth stage: there are no discrete sprites, only a plant evaluated
// at a different t.
const (
	stemGrowEnd  = 0.70
	nodeFirst    = 0.05
	nodeLast     = 0.78
	flowerOpen   = 0.58
	flowerFull   = 0.88
	fruitSet     = 0.80
	fruitRipe    = 1.00
	nodeFadeSpan = 0.12
)

// outlineWidth is the edge drawn around foliage, petals and fruit, in
// normalised units.
//
// Flat fills of similar colour merge where they overlap: a rosette came out as
// one solid dome rather than a whorl of blades. A darker edge on every organ
// separates them, which is what makes the silhouette readable.
const outlineWidth = 0.006

// addOutlined emits a filled shape with a darker edge at the same depth. The
// blueprint is sorted stably, so the edge always paints immediately over its
// own fill and never over the shape in front.
func (b *Blueprint) addOutlined(path []Segment, fill HSL, z int) {
	b.add(Shape{Kind: ShapeFill, Path: path, Color: fill, Z: z})
	b.add(Shape{Kind: ShapeStroke, Path: path, Color: shade(fill, 0.11), Width: outlineWidth, Z: z})
}

// stemProfile is how a stem archetype bends the shared construction. Keeping
// archetypes as parameters rather than separate build paths means a change of
// archetype moves the plant rather than replacing it.
type stemProfile struct {
	heightScale float64
	curveScale  float64
	nodeScale   float64
	// leafScale compensates for the frame each archetype offers. Foliage is
	// sized relative to stem height, so a rosette — which is almost all leaf
	// and almost no stem — would otherwise come out minuscule.
	leafScale float64
	branching bool
	rosette   bool
	bulb      bool
}

func profileFor(a plant.StemArchetype) stemProfile {
	switch a {
	case plant.StemBranching:
		return stemProfile{heightScale: 0.95, curveScale: 0.6, nodeScale: 1.2, leafScale: 0.85, branching: true}
	case plant.StemVining:
		return stemProfile{heightScale: 0.85, curveScale: 2.4, nodeScale: 1.6, leafScale: 1.05}
	case plant.StemRosette:
		return stemProfile{heightScale: 0.55, curveScale: 0.3, nodeScale: 1.5, leafScale: 1.55, rosette: true}
	case plant.StemBulb:
		return stemProfile{heightScale: 0.80, curveScale: 0.4, nodeScale: 0.7, leafScale: 1.25, bulb: true}
	default: // plant.StemUpright
		return stemProfile{heightScale: 1.0, curveScale: 0.55, nodeScale: 1.0, leafScale: 1.0}
	}
}

// Build turns an expressed phenotype into geometry at maturity t, where t
// runs 0 (just sown) to 1 (fully ripe).
//
// Build is pure: the same phenotype and t always produce the same Blueprint,
// with no hidden state and no RNG stream.
func Build(p plant.Phenotype, t float64) Blueprint {
	// Fit is measured once on the mature plant and applied at every stage.
	// Re-fitting per stage made a plant visibly shrink — by up to 16% — the
	// moment a flower opened or fruit set, because the new organ widened it
	// and the whole plant was compressed to compensate.
	b := build(p, t)
	scaleBlueprint(&b, fitScale(build(p, 1)))
	return b
}

func build(p plant.Phenotype, t float64) Blueprint {
	t = clamp01(t)
	var b Blueprint

	prof := profileFor(p.StemArchetype())
	grown := easeOut(ramp(t, 0, stemGrowEnd))
	if grown <= 0 {
		return b
	}

	full := matureHeight(p, prof)
	sp, widthAt := buildStem(p, prof, full, grown)
	b.add(Shape{
		Kind:  ShapeFill,
		Path:  ribbon(sp, widthAt),
		Color: stemColor(p),
		Z:     ZStem,
	})

	nodes, planned := placeNodes(p, prof, sp, t)
	addFoliage(&b, p, prof, nodes, planned, full)
	if prof.branching {
		addBranches(&b, p, sp, nodes)
	}
	addFlowers(&b, p, prof, sp, nodes, t)
	addFruit(&b, p, sp, nodes, t)

	// Stable paint order. sort.SliceStable keeps same-Z shapes in emission
	// order, so the blueprint stays deterministic.
	sort.SliceStable(b.Shapes, func(i, j int) bool { return b.Shapes[i].Z < b.Shapes[j].Z })
	return b
}

// matureHeight is the stem height the plant grows to, independent of how far
// along it currently is. Foliage and flowers are sized from it so a plant's
// proportions hold at every stage of growth.
func matureHeight(p plant.Phenotype, prof stemProfile) float64 {
	return lerp(0.55, 0.90, p.Unit(plant.GeneStemHeight)) * prof.heightScale
}

func buildStem(p plant.Phenotype, prof stemProfile, full, grown float64) (spine, func(float64) float64) {
	// The mature stem is built first and then scaled down by growth, rather
	// than being rebuilt at the current height. Uniform scaling preserves
	// angles, so a growing plant keeps its silhouette and its leaves hold
	// their heading. Deriving curvature from the current height instead made
	// the stem straighten and re-bend as it grew, swinging the foliage around
	// and visibly shrinking the plant by up to 12% mid-growth.
	curve := p.Signed(plant.GeneStemCurve) * prof.curveScale
	// A gentle phenotype-driven lean, so two plants of the same height are
	// not identical silhouettes.
	curve += wobble(p, 0, 0) * p.Unit(plant.GeneJitter) * 0.35

	sp := spine{
		P0: Point{0.5, 0},
		C1: Point{0.5 + curve*0.05*full, full * 0.34},
		C2: Point{0.5 + curve*0.24*full, full * 0.74},
		P3: Point{0.5 + curve*0.28*full, full},
	}.scaledAboutBase(grown)

	base := lerp(0.012, 0.070, p.Unit(plant.GeneStemThickness))
	// Taper is the fraction of base width remaining at the tip.
	taper := lerp(0.12, 0.95, p.Unit(plant.GeneStemTaper))
	bulge := 0.0
	if prof.bulb {
		bulge = 1.8
	}

	widthAt := func(u float64) float64 {
		w := base * lerp(1, taper, u)
		if bulge > 0 {
			w *= 1 + bulge*math.Exp(-u*7)
		}
		return w
	}
	return sp, widthAt
}

// node is one attachment point along the stem.
type node struct {
	index  int
	u      float64 // position along the spine
	pos    Point
	angle  float64 // the stem's heading here
	appear float64 // 0..1, how far this node has emerged
}

// placeNodes returns the nodes that have emerged by t, and the total the
// plant will eventually carry. Layout must be derived from the planned count,
// never the visible one: sizing a rosette's fan by how many leaves happen to
// have sprouted re-spaced the whole whorl every time one appeared, swinging
// established leaves and making the plant visibly shrink as it grew.
func placeNodes(p plant.Phenotype, prof stemProfile, sp spine, t float64) (nodes []node, planned int) {
	count := int(float64(p.Scaled(plant.GeneNodeCount, 1, 8)) * prof.nodeScale)
	if count < 1 {
		count = 1
	}
	if count > 12 {
		count = 12
	}

	nodes = make([]node, 0, count)
	for i := 0; i < count; i++ {
		frac := float64(i+1) / float64(count+1)
		u := frac
		if prof.rosette {
			// A rosette has no real stem: every leaf springs from the crown.
			u = 0.05 + frac*0.18
		}
		// Lower nodes emerge first, so the plant builds upward.
		start := lerp(nodeFirst, nodeLast, frac)
		appear := easeOut(ramp(t, start, start+nodeFadeSpan))
		if appear <= 0 {
			continue
		}
		nodes = append(nodes, node{
			index:  i,
			u:      u,
			pos:    sp.at(u),
			angle:  sp.angleAt(u),
			appear: appear,
		})
	}
	return nodes, count
}

func addFoliage(b *Blueprint, p plant.Phenotype, prof stemProfile, nodes []node, planned int, full float64) {
	arch := p.LeafArchetype()
	perNode := p.Scaled(plant.GeneLeafPerNode, 1, 3)
	// Leaves are sized relative to the frame carrying them. Sizing them
	// absolutely made every short plant a pile of foliage with the stem
	// buried somewhere inside it.
	frame := math.Max(full, 0.24)
	size := frame * lerp(0.20, 0.48, p.Unit(plant.GeneLeafSize)) * prof.leafScale
	width := lerp(0.28, 0.85, p.Unit(plant.GeneLeafSize)) * leafWidthFactor(arch)
	spread := lerp(0.45, 2.15, p.Unit(plant.GeneLeafDroop))

	leaf := leafPath(arch, width)
	front := foliageColor(p)
	back := shade(front, 0.17)

	for _, n := range nodes {
		for k := 0; k < perNode; k++ {
			side := 1.0
			if (n.index+k)%2 == 1 {
				side = -1
			}

			var rot float64
			if prof.rosette {
				// Fan evenly out of the crown rather than tracking a stem.
				// The fan spans the upper hemisphere only: a rosette seen from
				// the side splays left and right, and leaves aimed straight
				// down would draw through the soil.
				total := maxInt(planned*perNode, 1)
				rot = math.Pi * float64(n.index*perNode+k) / float64(total)
			} else {
				rot = n.angle + side*spread
			}
			rot += wobble(p, n.index*4+k, 1) * p.Unit(plant.GeneJitter) * 0.30
			rot = clampDroop(rot)

			z, col := ZFrontLeaf, front
			if (n.index+k)%2 == 1 {
				z, col = ZBackLeaf, back
			}

			// Vary blade length a little per leaf. Identical evenly spaced
			// leaves overlap into a solid arc — a rosette in particular came
			// out as a featureless dome rather than a plant.
			slot := n.index*perNode + k
			length := size * n.appear * (1 + 0.20*wobble(p, slot, 6))

			b.addOutlined(transform(leaf, xform{Origin: n.pos, Scale: length, Rot: rot}), col, z)
		}
	}
}

func addBranches(b *Blueprint, p plant.Phenotype, sp spine, nodes []node) {
	angle := lerp(0.30, 1.25, p.Unit(plant.GeneBranchAngle))
	length := lerp(0.08, 0.34, p.Unit(plant.GeneBranchLength))
	thickness := lerp(0.006, 0.030, p.Unit(plant.GeneStemThickness))
	col := shade(stemColor(p), 0.04)

	for _, n := range nodes {
		side := 1.0
		if n.index%2 == 1 {
			side = -1
		}
		rot := clampDroop(n.angle + side*angle + wobble(p, n.index, 2)*p.Unit(plant.GeneJitter)*0.25)
		l := length * n.appear
		if l <= 0 {
			continue
		}

		dx, dy := math.Cos(rot)*l, math.Sin(rot)*l
		br := spine{
			P0: n.pos,
			C1: Point{n.pos.X + dx*0.32, n.pos.Y + dy*0.42},
			C2: Point{n.pos.X + dx*0.70, n.pos.Y + dy*0.86},
			P3: Point{n.pos.X + dx, n.pos.Y + dy},
		}
		b.add(Shape{
			Kind:  ShapeFill,
			Path:  ribbon(br, func(u float64) float64 { return thickness * lerp(1, 0.35, u) }),
			Color: col,
			Z:     ZBranch,
		})
	}
}

// branchTip recomputes where a branch ends, so flowers and fruit can sit on it.
func branchTip(p plant.Phenotype, n node) Point {
	angle := lerp(0.30, 1.25, p.Unit(plant.GeneBranchAngle))
	length := lerp(0.08, 0.34, p.Unit(plant.GeneBranchLength))
	side := 1.0
	if n.index%2 == 1 {
		side = -1
	}
	rot := clampDroop(n.angle + side*angle + wobble(p, n.index, 2)*p.Unit(plant.GeneJitter)*0.25)
	l := length * n.appear
	return Point{X: n.pos.X + math.Cos(rot)*l, Y: n.pos.Y + math.Sin(rot)*l}
}

func addFlowers(b *Blueprint, p plant.Phenotype, prof stemProfile, sp spine, nodes []node, t float64) {
	arch := p.FlowerArchetype()
	if arch == plant.FlowerNone {
		return
	}
	open := ramp(t, flowerOpen, flowerFull)
	if open <= 0 {
		return
	}

	size := lerp(0.05, 0.22, p.Unit(plant.GeneFlowerSize))
	petal := flowerColor(p)
	centre := flowerCentreColor(p)
	tip := sp.at(1)

	if arch == plant.FlowerSpike {
		// A spike is many small heads stacked up the upper stem rather than
		// one head at the tip.
		const heads = 5
		for i := 0; i < heads; i++ {
			u := lerp(0.62, 1.0, float64(i)/float64(heads-1))
			// Upper florets open later, so the spike blooms from the bottom.
			headOpen := ramp(t, flowerOpen+float64(i)*0.03, flowerFull+float64(i)*0.02)
			b.addFlower(p, sp.at(u), size*0.55, headOpen, petal, centre, ZFlower)
		}
		return
	}

	b.addFlower(p, tip, size, open, petal, centre, ZFlower)

	if prof.branching {
		for _, n := range nodes {
			if n.appear < 0.9 {
				continue
			}
			b.addFlower(p, branchTip(p, n), size*0.7, open, petal, centre, ZFlower)
		}
	}
}

func addFruit(b *Blueprint, p plant.Phenotype, sp spine, nodes []node, t float64) {
	arch := p.FruitArchetype()
	if arch == plant.FruitNone {
		return
	}
	set := ramp(t, fruitSet, fruitRipe)
	if set <= 0 {
		return
	}

	size := lerp(0.03, 0.11, p.Unit(plant.GeneFruitSize)) * easeOut(set)
	sites := p.Scaled(plant.GeneFruitCount, 1, 5)
	perSite := fruitsPerCluster(arch)
	col := fruitColor(p)
	body := fruitPath(arch, 1)

	placed := 0
	// Fruit hangs from the upper nodes, where a real plant carries it.
	for i := len(nodes) - 1; i >= 0 && placed < sites; i-- {
		n := nodes[i]
		if n.appear < 0.85 {
			continue
		}
		for k := 0; k < perSite && placed < sites; k++ {
			rot := -math.Pi/2 + wobble(p, placed, 4)*0.55
			origin := Point{
				X: n.pos.X + wobble(p, placed, 5)*size*0.8,
				Y: n.pos.Y,
			}
			b.add(Shape{
				Kind:  ShapeFill,
				Path:  transform(body, xform{Origin: origin, Scale: size, Rot: rot}),
				Color: col,
				Z:     ZFruit,
			})
			placed++
		}
	}
}
