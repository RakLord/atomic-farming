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

// Discard removes an entire line from the inventory.
//
// The unit a player wants to bin is a lineage, not a single seed — nobody
// throws away one seed of a strain they have decided against.
func (inv *Inventory) Discard(kind CropKind, g plant.Genome) bool {
	i := inv.indexOf(kind, g)
	if i < 0 {
		return false
	}
	inv.Stacks = append(inv.Stacks[:i], inv.Stacks[i+1:]...)
	return true
}

// SeedGroup is one row of the seed list: either a species' unnamed holdings
// collapsed together, or a single named strain.
//
// Grouping is a view over the stored inventory, not a change to it, so the
// save format is untouched.
type SeedGroup struct {
	Kind   CropKind
	Name   string
	Strain NamedStrain
	Named  bool
	// Stacks indexes into Inventory.Stacks.
	Stacks []int
	Count  int
}

// GroupSeeds buckets the inventory for display.
//
// Unnamed seeds of a species collapse into one row — they are the pile nobody
// wants to scroll — while every named strain gets its own, because those are
// exactly the ones worth choosing between.
//
// Groups come out in first-seen order over the stacks. The map here is only a
// lookup; iterating it would reshuffle the list between frames, the same trap
// IdentifyStrain avoids.
func (s *GameState) GroupSeeds() []SeedGroup {
	type key struct {
		kind   CropKind
		strain StrainID
	}
	at := map[key]int{}
	var groups []SeedGroup

	for i, stack := range s.Inventory.Stacks {
		strain, named := IdentifyStrain(stack.Kind, stack.Genome, SeedPhenotype(stack))
		k := key{kind: stack.Kind}
		name := CropDisplayName(stack.Kind)
		if named {
			k.strain, name = strain.ID, strain.Name
		}

		gi, seen := at[k]
		if !seen {
			groups = append(groups, SeedGroup{Kind: stack.Kind, Name: name, Strain: strain, Named: named})
			gi = len(groups) - 1
			at[k] = gi
		}
		groups[gi].Stacks = append(groups[gi].Stacks, i)
		groups[gi].Count += stack.Count
	}
	return groups
}

// DefaultStack is the line a group sows when the player has not picked one:
// the most numerous, ties going to the earliest.
//
// Predictable beats clever here — the bulk line is almost always the one being
// cultivated, and a player who wants a specific line opens the index.
func (g SeedGroup) DefaultStack(inv *Inventory) int {
	best, bestCount := -1, 0
	for _, i := range g.Stacks {
		if i < 0 || i >= len(inv.Stacks) {
			continue
		}
		if c := inv.Stacks[i].Count; c > bestCount {
			best, bestCount = i, c
		}
	}
	return best
}

// StacksOfKind lists the inventory positions holding seeds of one species, in
// inventory order. It is what the seed index shows.
func (inv *Inventory) StacksOfKind(kind CropKind) []int {
	var out []int
	for i, stack := range inv.Stacks {
		if stack.Kind == kind {
			out = append(out, i)
		}
	}
	return out
}
