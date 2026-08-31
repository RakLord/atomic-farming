package bignum

import (
	"strconv"
	"strings"
)

type DisplayMode string

const (
	DisplayScientific DisplayMode = "scientific"
	DisplayShort      DisplayMode = "short"
)

var shortSuffixes = [...]string{"", "K", "M", "B", "T", "Qa", "Qi", "Sx", "Sp", "Oc", "No", "Dc"}

// Format returns a player-facing string. It intentionally does not share the
// same contract as String(), which stays canonical for save/load.
func (d Decimal) Format(mode DisplayMode, places int) string {
	switch mode {
	case DisplayShort:
		return d.formatShort(places)
	default:
		return d.formatScientific(places)
	}
}

func (d Decimal) formatScientific(places int) string {
	if d.IsZero() {
		return "0"
	}
	absExp := d.Abs().exponent
	if absExp <= 5 {
		return d.formatPlain(places)
	}
	return formatSignedMantissa(d.sign, d.mantissa, places) + "e" + strconv.FormatInt(d.exponent, 10)
}

func (d Decimal) formatShort(places int) string {
	if d.IsZero() {
		return "0"
	}
	absExp := d.Abs().exponent
	if absExp < 3 {
		return d.formatPlain(places)
	}
	group := int(absExp / 3)
	if group >= len(shortSuffixes) {
		return d.formatScientific(places)
	}
	scaled := d.mantissa * pow10Small(int(absExp%3))
	return formatSignedMantissa(d.sign, scaled, places) + shortSuffixes[group]
}

func (d Decimal) formatPlain(places int) string {
	if d.IsZero() {
		return "0"
	}
	value := d.Float64()
	text := strconv.FormatFloat(value, 'f', places, 64)
	text = trimFraction(text)
	return addCommas(text)
}

func formatSignedMantissa(sign int8, mantissa float64, places int) string {
	text := strconv.FormatFloat(mantissa, 'f', places, 64)
	text = trimFraction(text)
	if sign < 0 {
		return "-" + text
	}
	return text
}

func trimFraction(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "-0" {
		return "0"
	}
	return s
}

func addCommas(s string) string {
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	if len(intPart) <= 3 {
		if len(parts) == 2 {
			return sign + intPart + "." + parts[1]
		}
		return sign + intPart
	}

	buf := make([]byte, 0, len(intPart)+len(intPart)/3)
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, byte(r))
	}
	if len(parts) == 2 {
		return sign + string(buf) + "." + parts[1]
	}
	return sign + string(buf)
}

func pow10Small(exp int) float64 {
	v := 1.0
	for range exp {
		v *= 10
	}
	return v
}
