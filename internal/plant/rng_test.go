package plant

import "testing"

// TestHash64MatchesSplitmix64Vectors pins the hash to known splitmix64
// output. This is the strongest guard in the package: save-visible outcomes
// derive from this stream, so a change here would silently rewrite every
// existing plant's rolls.
func TestHash64MatchesSplitmix64Vectors(t *testing.T) {
	vectors := []struct {
		in   uint64
		want uint64
	}{
		{0, 0xe220a8397b1dcdaf},
		{1, 0x910a2dec89025cc1},
		{2, 0x975835de1c9756ce},
		{42, 0xbdd732262feb6e95},
		{1 << 63, 0x481ec0a212a9f3db},
	}
	for _, v := range vectors {
		if got := Hash64(v.in); got != v.want {
			t.Errorf("Hash64(%d) = 0x%016x, want 0x%016x", v.in, got, v.want)
		}
	}
}

func TestRollStaysInRangeAndIsDeterministic(t *testing.T) {
	for n := uint64(1); n <= 64; n++ {
		for salt := uint64(0); salt < 32; salt++ {
			got := Roll(7, PurposeBreed, salt, n)
			if got >= n {
				t.Fatalf("Roll(.., n=%d) = %d, out of range", n, got)
			}
			if again := Roll(7, PurposeBreed, salt, n); again != got {
				t.Fatalf("Roll is not deterministic: %d then %d", got, again)
			}
		}
	}
	if got := Roll(1, PurposeBreed, 0, 0); got != 0 {
		t.Errorf("Roll with n=0 = %d, want 0", got)
	}
}

func TestPurposeTagsDecorrelateRolls(t *testing.T) {
	// The same seed and salt under different purposes must not produce the
	// same stream, or a death roll would predict a harvest roll.
	const trials = 256
	agree := 0
	for i := uint64(0); i < trials; i++ {
		a := Roll(99, PurposeDeath, i, 100)
		b := Roll(99, PurposeHarvest, i, 100)
		if a == b {
			agree++
		}
	}
	// Chance agreement is ~1%; anything near total agreement means the
	// purpose tag is not reaching the hash.
	if agree > trials/10 {
		t.Errorf("%d/%d rolls agreed across purposes; purpose tag is not decorrelating", agree, trials)
	}
}

func TestChanceHonoursBoundaries(t *testing.T) {
	if Chance(1, PurposeDeath, 0, 0) {
		t.Error("Chance at 0 bp fired")
	}
	if Chance(1, PurposeDeath, 0, -5) {
		t.Error("Chance at negative bp fired")
	}
	if !Chance(1, PurposeDeath, 0, BasisPoints) {
		t.Error("Chance at 10000 bp did not fire")
	}
	if !Chance(1, PurposeDeath, 0, BasisPoints+5) {
		t.Error("Chance above 10000 bp did not fire")
	}
}

func TestChanceApproximatesItsStatedRate(t *testing.T) {
	const trials = 20000
	for _, bp := range []int{500, 2500, 5000, 9000} {
		hits := 0
		for i := uint64(0); i < trials; i++ {
			if Chance(12345, PurposeMutation, i, bp) {
				hits++
			}
		}
		want := trials * bp / BasisPoints
		if diff := hits - want; diff < -trials/50 || diff > trials/50 {
			t.Errorf("bp=%d: %d/%d hits, want about %d", bp, hits, trials, want)
		}
	}
}

func TestUnitFloatStaysInRange(t *testing.T) {
	for i := uint64(0); i < 4096; i++ {
		u := UnitFloat(3, PurposeJitter, i)
		if u < 0 || u >= 1 {
			t.Fatalf("UnitFloat = %v, want [0,1)", u)
		}
		if s := SignedUnit(3, PurposeJitter, i); s < -1 || s >= 1 {
			t.Fatalf("SignedUnit = %v, want [-1,1)", s)
		}
	}
}
