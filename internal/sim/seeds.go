package sim

import "atomicfarming/internal/plant"

// SeedStack is a quantity of identical seeds: same species, same genome.
//
// A seed carries a genome, not just a species. That is what lets a strain the
// player bred or discovered be replanted at all — without it, a rare find
// could never be propagated.
type SeedStack struct {
	Kind   CropKind     `json:"kind"`
	Genome plant.Genome `json:"genome"`
	Count  int          `json:"count"`
}

// Inventory holds the player's seeds.
//
// Stacks are a slice in insertion order rather than a map keyed by genome: map
// iteration order is random in Go, and the inventory list must render the same
// way every frame.
type Inventory struct {
	Stacks []SeedStack `json:"stacks,omitempty"`
}

// plant.Genome is a comparable array, so stack identity is a plain equality
// check on (kind, genome).
func (inv *Inventory) indexOf(kind CropKind, g plant.Genome) int {
	for i := range inv.Stacks {
		if inv.Stacks[i].Kind == kind && inv.Stacks[i].Genome == g {
			return i
		}
	}
	return -1
}

// Add puts n seeds into the inventory, merging into an existing stack when the
// genome matches exactly. Non-positive counts are ignored.
func (inv *Inventory) Add(kind CropKind, g plant.Genome, n int) {
	if n <= 0 {
		return
	}
	if i := inv.indexOf(kind, g); i >= 0 {
		inv.Stacks[i].Count += n
		return
	}
	inv.Stacks = append(inv.Stacks, SeedStack{Kind: kind, Genome: g, Count: n})
}

// Take removes one seed, reporting whether there was one to take. An emptied
// stack is dropped so the inventory does not fill with zero rows.
func (inv *Inventory) Take(kind CropKind, g plant.Genome) bool {
	i := inv.indexOf(kind, g)
	if i < 0 || inv.Stacks[i].Count <= 0 {
		return false
	}
	inv.Stacks[i].Count--
	if inv.Stacks[i].Count == 0 {
		inv.Stacks = append(inv.Stacks[:i], inv.Stacks[i+1:]...)
	}
	return true
}

// TakeAt removes one seed from the stack at index i.
func (inv *Inventory) TakeAt(i int) (SeedStack, bool) {
	if i < 0 || i >= len(inv.Stacks) {
		return SeedStack{}, false
	}
	stack := inv.Stacks[i]
	inv.Take(stack.Kind, stack.Genome)
	stack.Count = 1
	return stack, true
}

// Total is how many seeds the player holds across every stack.
func (inv *Inventory) Total() int {
	n := 0
	for _, s := range inv.Stacks {
		n += s.Count
	}
	return n
}

// prune drops stacks with a non-positive count, repairing a hand-edited save.
func (inv *Inventory) prune() {
	kept := inv.Stacks[:0]
	for _, s := range inv.Stacks {
		if s.Count > 0 && s.Kind != "" {
			kept = append(kept, s)
		}
	}
	inv.Stacks = kept
}
