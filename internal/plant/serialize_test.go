package plant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenomeStringRoundTrips(t *testing.T) {
	for seed := uint64(0); seed < 64; seed++ {
		g := RandomGenome(seed)
		got, err := ParseGenome(g.String())
		if err != nil {
			t.Fatalf("seed %d: ParseGenome: %v", seed, err)
		}
		if got != g {
			t.Fatalf("seed %d: round trip changed the genome", seed)
		}
	}
}

// TestParseFillsMissingGenesFromDefaults is what makes the append-only gene
// catalog safe: a genome saved before a gene existed must still load.
func TestParseFillsMissingGenesFromDefaults(t *testing.T) {
	full := RandomGenome(9)
	encoded := full.String()
	colon := strings.IndexByte(encoded, ':')

	// Truncate to the first three genes, as an older build would have written.
	short := encoded[:colon+1] + encoded[colon+1:colon+1+3*4]

	got, err := ParseGenome(short)
	if err != nil {
		t.Fatalf("ParseGenome: %v", err)
	}
	for i := 0; i < 3; i++ {
		if got[i] != full[i] {
			t.Errorf("gene %d was not preserved: %+v, want %+v", i, got[i], full[i])
		}
	}
	for i := 3; i < GeneCount; i++ {
		d := GeneCatalog[i].Default
		if got[i] != (GenePair{A: d, B: d}) {
			t.Errorf("gene %d was not filled from its default: %+v", i, got[i])
		}
	}
}

// TestParseIgnoresUnknownTrailingGenes keeps a save touched by a newer build
// loadable rather than fatal.
func TestParseIgnoresUnknownTrailingGenes(t *testing.T) {
	g := RandomGenome(4)
	extended := g.String() + "abcd"

	got, err := ParseGenome(extended)
	if err != nil {
		t.Fatalf("ParseGenome with extra genes: %v", err)
	}
	if got != g {
		t.Error("known genes were not preserved when trailing genes were present")
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	cases := map[string]string{
		"no version prefix":  "deadbeef",
		"unreadable version": "x:dead",
		"future version":     "99:dead",
		"ragged body":        "1:abc",
		"non-hex body":       "1:zzzz",
	}
	for name, s := range cases {
		if _, err := ParseGenome(s); err == nil {
			t.Errorf("%s: ParseGenome(%q) returned no error", name, s)
		}
	}
}

func TestFingerprintIsStableAndDiscriminating(t *testing.T) {
	g := RandomGenome(17)
	first := g.Fingerprint()
	if again := g.Fingerprint(); again != first {
		t.Error("Fingerprint is not stable for a fixed genome")
	}

	// A one-step change must produce a different fingerprint, or the render
	// cache would serve a stale sprite for a mutated plant.
	mutated := g
	mutated[GeneStemHeight].A++
	if mutated.Fingerprint() == first {
		t.Error("a one-step allele change did not change the fingerprint")
	}

	seen := map[uint64]bool{}
	for seed := uint64(0); seed < 512; seed++ {
		f := RandomGenome(seed).Fingerprint()
		if seen[f] {
			t.Fatalf("fingerprint collision at seed %d", seed)
		}
		seen[f] = true
	}
}

func TestLabelIsShortAndStable(t *testing.T) {
	g := RandomGenome(3)
	label := g.Label()
	if len(label) != 9 || label[4] != '-' {
		t.Errorf("Label() = %q, want the form XXXX-XXXX", label)
	}
	if again := g.Label(); again != label {
		t.Error("Label is not stable")
	}
}

func TestGenomeJSONUsesTheCompactString(t *testing.T) {
	g := RandomGenome(21)
	blob, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.HasPrefix(string(blob), `"1:`) {
		t.Errorf("genome marshaled as %s, want a versioned string", blob)
	}

	var got Genome
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != g {
		t.Error("JSON round trip changed the genome")
	}

	if err := json.Unmarshal([]byte(`"garbage"`), &got); err == nil {
		t.Error("Unmarshal accepted a malformed genome string")
	}
}
