package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestInventoryMergesIdenticalGenomes(t *testing.T) {
	var inv Inventory
	g := plant.RandomGenome(1)

	inv.Add(kindTestCrop, g, 2)
	inv.Add(kindTestCrop, g, 3)

	if len(inv.Stacks) != 1 {
		t.Fatalf("got %d stacks, want 1 — identical seeds did not merge", len(inv.Stacks))
	}
	if inv.Stacks[0].Count != 5 {
		t.Errorf("count = %d, want 5", inv.Stacks[0].Count)
	}
	if inv.Total() != 5 {
		t.Errorf("Total = %d, want 5", inv.Total())
	}
}

func TestInventorySeparatesDifferentGenomes(t *testing.T) {
	var inv Inventory
	a, b := plant.RandomGenome(1), plant.RandomGenome(2)

	inv.Add(kindTestCrop, a, 1)
	inv.Add(kindTestCrop, b, 1)

	if len(inv.Stacks) != 2 {
		t.Fatalf("got %d stacks, want 2 — distinct strains were merged", len(inv.Stacks))
	}
	// Insertion order must hold, or the inventory list would reshuffle itself
	// between frames.
	if inv.Stacks[0].Genome != a || inv.Stacks[1].Genome != b {
		t.Error("stacks are not in insertion order")
	}
}

func TestInventoryTakeRemovesEmptiedStacks(t *testing.T) {
	var inv Inventory
	g := plant.RandomGenome(3)
	inv.Add(kindTestCrop, g, 2)

	if !inv.Take(kindTestCrop, g) || !inv.Take(kindTestCrop, g) {
		t.Fatal("Take failed while seeds remained")
	}
	if inv.Take(kindTestCrop, g) {
		t.Error("Take succeeded on an empty inventory")
	}
	if len(inv.Stacks) != 0 {
		t.Errorf("%d stacks remain, want the emptied stack dropped", len(inv.Stacks))
	}
}

func TestInventoryIgnoresNonsenseCounts(t *testing.T) {
	var inv Inventory
	inv.Add(kindTestCrop, plant.RandomGenome(4), 0)
	inv.Add(kindTestCrop, plant.RandomGenome(5), -3)
	if len(inv.Stacks) != 0 {
		t.Errorf("got %d stacks, want none", len(inv.Stacks))
	}
}

func TestInventoryTakeAtRemovesOneSeed(t *testing.T) {
	var inv Inventory
	g := plant.RandomGenome(6)
	inv.Add(kindTestCrop, g, 3)

	stack, ok := inv.TakeAt(0)
	if !ok {
		t.Fatal("TakeAt failed")
	}
	if stack.Count != 1 || stack.Genome != g {
		t.Errorf("TakeAt returned %+v, want a single seed of the stack's genome", stack)
	}
	if inv.Total() != 2 {
		t.Errorf("Total = %d, want 2", inv.Total())
	}
	if _, ok := inv.TakeAt(9); ok {
		t.Error("TakeAt succeeded on an out-of-range index")
	}
}

func TestInventoryPruneDropsCorruptStacks(t *testing.T) {
	inv := Inventory{Stacks: []SeedStack{
		{Kind: kindTestCrop, Count: 2},
		{Kind: kindTestCrop, Count: 0},
		{Kind: "", Count: 5},
		{Kind: kindTestCrop, Count: -1},
	}}
	inv.prune()
	if len(inv.Stacks) != 1 || inv.Stacks[0].Count != 2 {
		t.Errorf("prune left %+v, want only the one valid stack", inv.Stacks)
	}
}
