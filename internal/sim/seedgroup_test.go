package sim

import (
	"testing"

	"atomicfarming/internal/plant"
)

func TestUnnamedSeedsCollapseIntoOneGroup(t *testing.T) {
	withStrains(t) // no strains registered: everything is unnamed
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}
	for i := uint64(0); i < 8; i++ {
		s.Inventory.Add(kindTestCrop, plant.RandomGenome(i), 1)
	}

	groups := s.GroupSeeds()
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1 — this is the pile the grouping exists to hide", len(groups))
	}
	if groups[0].Count != 8 {
		t.Errorf("group holds %d seeds, want 8", groups[0].Count)
	}
	if len(groups[0].Stacks) != 8 {
		t.Errorf("group covers %d lines, want 8", len(groups[0].Stacks))
	}
	if groups[0].Named {
		t.Error("an unnamed group is marked named")
	}
	if groups[0].Name != CropDisplayName(kindTestCrop) {
		t.Errorf("group name = %q, want the species name", groups[0].Name)
	}
}

// TestNamedStrainsGetTheirOwnGroup: the whole point of naming a strain is to
// be able to pick it out, so it must not be buried in the species pile.
func TestNamedStrainsGetTheirOwnGroup(t *testing.T) {
	withStrains(t, denseStrain("dense", 200, 10))
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}

	heavy := genomeWith(map[plant.GeneID]plant.Allele{plant.GeneDensity: 240})
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(1), 2)
	s.Inventory.Add(kindTestCrop, heavy, 3)
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(2), 1)

	groups := s.GroupSeeds()
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2 (unnamed pile + the named strain)", len(groups))
	}

	var named, plain *SeedGroup
	for i := range groups {
		if groups[i].Named {
			named = &groups[i]
		} else {
			plain = &groups[i]
		}
	}
	if named == nil || plain == nil {
		t.Fatal("expected one named and one unnamed group")
	}
	if named.Name != "dense" || named.Count != 3 {
		t.Errorf("named group = %q x%d, want dense x3", named.Name, named.Count)
	}
	if plain.Count != 3 {
		t.Errorf("unnamed group holds %d, want 3", plain.Count)
	}
}

func TestGroupOrderIsStable(t *testing.T) {
	withStrains(t)
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}
	for i := uint64(0); i < 6; i++ {
		s.Inventory.Add(kindTestCrop, plant.RandomGenome(i), 1)
	}

	first := s.GroupSeeds()
	for i := 0; i < 50; i++ {
		got := s.GroupSeeds()
		if len(got) != len(first) {
			t.Fatal("group count is unstable")
		}
		for j := range got {
			if got[j].Name != first[j].Name || got[j].Count != first[j].Count {
				t.Fatal("group order is unstable; the list would reshuffle between frames")
			}
		}
	}
}

func TestDefaultStackPicksTheBulkLine(t *testing.T) {
	withStrains(t)
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(1), 1)
	bulk := plant.RandomGenome(2)
	s.Inventory.Add(kindTestCrop, bulk, 9)
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(3), 2)

	groups := s.GroupSeeds()
	idx := groups[0].DefaultStack(&s.Inventory)
	if idx < 0 {
		t.Fatal("no default line")
	}
	if s.Inventory.Stacks[idx].Genome != bulk {
		t.Error("DefaultStack did not pick the most numerous line")
	}

	if empty := (SeedGroup{}).DefaultStack(&s.Inventory); empty != -1 {
		t.Errorf("an empty group gave %d, want -1", empty)
	}
}

func TestDiscardRemovesAWholeLine(t *testing.T) {
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}
	keep, bin := plant.RandomGenome(1), plant.RandomGenome(2)
	s.Inventory.Add(kindTestCrop, keep, 2)
	s.Inventory.Add(kindTestCrop, bin, 5)

	if !s.Inventory.Discard(kindTestCrop, bin) {
		t.Fatal("Discard reported failure")
	}
	if s.Inventory.Total() != 2 {
		t.Errorf("%d seeds remain, want 2 — a discard should take the whole line", s.Inventory.Total())
	}
	if len(s.Inventory.Stacks) != 1 || s.Inventory.Stacks[0].Genome != keep {
		t.Error("the wrong line was discarded")
	}
	if s.Inventory.Discard(kindTestCrop, bin) {
		t.Error("discarding an absent line reported success")
	}
}

func TestStacksOfKindListsOnlyThatSpecies(t *testing.T) {
	s := NewGameStateWithSeed(1)
	s.Inventory = Inventory{}
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(1), 1)
	s.Inventory.Add("other_species", plant.RandomGenome(2), 1)
	s.Inventory.Add(kindTestCrop, plant.RandomGenome(3), 1)

	got := s.Inventory.StacksOfKind(kindTestCrop)
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("StacksOfKind = %v, want [0 2] in inventory order", got)
	}
}
