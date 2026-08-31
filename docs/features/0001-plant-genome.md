# Feature 0001 — The plant genome

Every plant carries a genome of **45 genes**, each holding **two alleles** (0–255). What you see is the *expressed* value; what a plant passes on is its alleles. The two differ, and that difference is the breeding game.

Architecture and rationale live in `docs/adr/0009-plant-genome.md`. This document is the reference for what each gene does.

## Reading a gene

| Term | Meaning |
|---|---|
| Allele | One of the two copies of a gene. |
| Expression | How the pair collapses into the value the plant shows. |
| Co-dominant | The midpoint of the two alleles. Smooth; used for numeric traits. |
| Dominant | The higher allele wins. Used for categorical archetypes. |
| Recessive | The lower allele wins, so a high value hides until both alleles carry it. |
| Phenotype | The expressed genome, after the species' gene ranges are applied. |
| Strain | A genome, named by its short fingerprint (`4F2A-91BC`). |

## Stem — the plant's frame

| Gene | Expression | Effect |
|---|---|---|
| Stem Archetype | Dominant | Upright, Branching, Vining, Rosette, or Bulb. Sets the whole silhouette. |
| Stem Height | Co-dominant | How tall the mature plant grows. Everything else is sized from it. |
| Stem Thickness | Co-dominant | Width at the base. |
| Stem Curve | Co-dominant | Lean and bow. The midpoint is straight. |
| Stem Taper | Co-dominant | How much the stem narrows toward the tip. |
| Node Count | Co-dominant | Attachment points, and so how many leaves and branches. |
| Branch Angle | Co-dominant | How far branches swing from the stem (Branching only). |
| Branch Length | Co-dominant | How far branches reach (Branching only). |
| Stem Hue / Saturation / Luminance | Co-dominant | The stem's colour, which foliage is derived from. |

## Foliage

| Gene | Expression | Effect |
|---|---|---|
| Leaf Archetype | Dominant | Oval, Lance, Lobed, Needle, Heart, or Fan. |
| Leaf Size | Co-dominant | Blade size, relative to stem height. |
| Leaf Droop | Co-dominant | How far blades splay from the stem, upswept to drooping. |
| Leaves Per Node | Co-dominant | 1–3 blades at each node. |
| Foliage Hue Shift | Co-dominant | Leaf hue as an offset from the stem's. |
| Foliage Luminance Shift | Co-dominant | Leaf lightness as an offset from the stem's. |

Foliage colour is a *shift* rather than its own triple so the plant always reads as one organism.

## Flower

| Gene | Expression | Effect |
|---|---|---|
| Flower Archetype | Dominant | None, Bell, Star, Disc, Cluster, Spike, or Trumpet. |
| Flower Size | Co-dominant | Head size and the size of its eye. |
| Petal Count | Co-dominant | 3–12 petals. |
| Petal Length / Width | Co-dominant | Petal proportions. |
| Petal Curl | Co-dominant | Whether petals hook inward or flare outward. |
| Flower Hue / Saturation / Luminance | Co-dominant | Petal colour, independent of the stem's. |

## Fruit

| Gene | Expression | Effect |
|---|---|---|
| Fruit Archetype | Dominant | None, Berry, Pod, Grain Head, Tuber, or Capsule. |
| Fruit Size | Co-dominant | Body size, and how dark the fruit reads. |
| Fruit Count | Co-dominant | 1–5 fruiting sites. |
| Fruit Hue | Co-dominant | Fruit colour. |

## Noise

| Gene | Expression | Effect |
|---|---|---|
| Jitter | Co-dominant | How much asymmetry the plant carries. Zero is mechanically perfect. |
| Symmetry | Co-dominant | Biases the noise pattern between bilateral and radial. |

## Vigour *(declared; read from Phase 1)*

| Gene | Expression | Intended effect |
|---|---|---|
| Growth Rate | Co-dominant | Ticks needed per growth stage. |
| Vitality | Dominant | Resistance to the per-stage death roll. Hardiness dominates. |
| Lifespan | Co-dominant | Harvests a perennial yields before dying. |
| Water Need | Co-dominant | Reserved for watering. |
| Nutrient Drain | Co-dominant | Reserved for soil and fertiliser. |

## Yield *(declared; read from Phase 1)*

| Gene | Expression | Intended effect |
|---|---|---|
| Yield Amount | Co-dominant | Units produced per harvest. |
| Yield Quality | Co-dominant | Value multiplier per unit. |
| Harvest Chance | Co-dominant | Probability a harvest succeeds. |
| **Density** | Co-dominant | Mass per unit. **The stat the whole game climbs toward** — the prestige goal is matter dense enough to collapse the universe. |
| Regrowth | Co-dominant | 0 is an annual; higher regrows faster after harvest. |

## Meta *(declared; read from Phase 1)*

| Gene | Expression | Intended effect |
|---|---|---|
| Mutability | **Recessive** | The plant's own mutation rate. Recessive, so high mutability hides in the pool and takes deliberate inbreeding to express. |
| Fertility | Co-dominant | Chance a cross succeeds. |
| Affinity | Co-dominant | Bonus from neighbouring plots. |

## Breeding

A cross draws one allele from each parent for every gene (meiosis), then applies mutation. Mutation is per-allele, shifts by at most 3 steps, and runs at a rate derived from the parents' Mutability plus any global upgrade — at most 0.4% per allele from genetics alone.

Because a child's alleles are always literally its parents', a rare extreme allele passes down intact even while it is not expressed. That is what lets a breeding line keep improving instead of converging on the average.

## Looking at plants

```bash
go run ./cmd/game                     # press L for the genetics lab
go run ./cmd/plantsheet -mode population
go run ./cmd/plantsheet -mode growth
go run ./cmd/plantsheet -mode mutations
go run ./cmd/plantsheet -mode archetypes
```
