package plant

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// genomeFormatVersion prefixes an encoded genome. Bump it only if the
// encoding itself changes; appending genes does not, because a short genome
// is filled from catalog defaults on parse.
const genomeFormatVersion = 1

const hexDigits = "0123456789abcdef"

// String encodes the genome losslessly as "<version>:<hex>", two hex digits
// per allele, alleles in gene order.
//
// This is the shareable strain code. It is stable across builds as long as
// the gene catalog only ever grows.
func (g Genome) String() string {
	var b strings.Builder
	b.Grow(len(strconv.Itoa(genomeFormatVersion)) + 1 + GeneCount*4)
	b.WriteString(strconv.Itoa(genomeFormatVersion))
	b.WriteByte(':')
	for i := 0; i < GeneCount; i++ {
		writeHexByte(&b, uint8(g[i].A))
		writeHexByte(&b, uint8(g[i].B))
	}
	return b.String()
}

func writeHexByte(b *strings.Builder, v uint8) {
	b.WriteByte(hexDigits[v>>4])
	b.WriteByte(hexDigits[v&0x0f])
}

// ParseGenome decodes a genome string.
//
// A genome encoding fewer genes than the current catalog is filled from
// catalog defaults — that is what makes appending a gene safe for existing
// saves. A genome encoding more genes than this build knows about has the
// extras ignored rather than rejected, so a save touched by a newer build
// still loads.
func ParseGenome(s string) (Genome, error) {
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return Genome{}, fmt.Errorf("plant: genome %q has no version prefix", s)
	}
	version, err := strconv.Atoi(s[:colon])
	if err != nil {
		return Genome{}, fmt.Errorf("plant: genome %q has an unreadable version: %w", s, err)
	}
	if version != genomeFormatVersion {
		return Genome{}, fmt.Errorf("plant: unsupported genome format version %d", version)
	}
	body := s[colon+1:]
	if len(body)%4 != 0 {
		return Genome{}, fmt.Errorf("plant: genome body has %d hex digits, not a whole number of gene pairs", len(body))
	}

	g := DefaultGenome()
	encoded := len(body) / 4
	for i := 0; i < encoded && i < GeneCount; i++ {
		a, err := parseHexByte(body[i*4 : i*4+2])
		if err != nil {
			return Genome{}, fmt.Errorf("plant: gene %d allele A: %w", i, err)
		}
		b, err := parseHexByte(body[i*4+2 : i*4+4])
		if err != nil {
			return Genome{}, fmt.Errorf("plant: gene %d allele B: %w", i, err)
		}
		g[i] = GenePair{A: Allele(a), B: Allele(b)}
	}
	return g, nil
}

func parseHexByte(s string) (uint8, error) {
	v, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("%q is not a hex byte", s)
	}
	return uint8(v), nil
}

// Fingerprint is a stable 64-bit hash of the genome, for display and for
// keying render caches. Two genomes with the same fingerprint are the same
// genome for every practical purpose.
func (g Genome) Fingerprint() uint64 {
	var h uint64 = 0xcbf29ce484222325
	for i := 0; i < GeneCount; i++ {
		h = Hash64(h ^ uint64(g[i].A)<<8 ^ uint64(g[i].B))
	}
	return h
}

// Label is the short human-facing strain name, e.g. "4F2A-91BC".
func (g Genome) Label() string {
	f := g.Fingerprint()
	return fmt.Sprintf("%04X-%04X", uint16(f>>48), uint16(f>>32))
}

// MarshalJSON encodes the genome as its compact string, so a save stays
// readable and a plot costs one short string rather than an array of pairs.
func (g Genome) MarshalJSON() ([]byte, error) { return json.Marshal(g.String()) }

func (g *Genome) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseGenome(s)
	if err != nil {
		return err
	}
	*g = parsed
	return nil
}
