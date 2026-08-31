package bignum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const maxAddExponentGap int64 = 18

var negativePowers10 = [...]float64{
	1,
	0.1,
	0.01,
	0.001,
	0.0001,
	0.00001,
	0.000001,
	0.0000001,
	0.00000001,
	0.000000001,
	0.0000000001,
	0.00000000001,
	0.000000000001,
	0.0000000000001,
	0.00000000000001,
	0.000000000000001,
	0.0000000000000001,
	0.00000000000000001,
	0.000000000000000001,
}

// Decimal is a normalized scientific-decimal value optimized for the kinds of
// large-number math common in incremental games.
type Decimal struct {
	sign     int8
	mantissa float64
	exponent int64
}

func Zero() Decimal { return Decimal{} }

func One() Decimal { return FromInt(1) }

func FromInt(v int) Decimal { return FromInt64(int64(v)) }

func FromInt64(v int64) Decimal {
	if v == 0 {
		return Zero()
	}
	if v < 0 {
		return normalize(-1, float64(-v), 0)
	}
	return normalize(1, float64(v), 0)
}

func FromFloat64(v float64) Decimal {
	if v == 0 {
		return Zero()
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		panic(fmt.Sprintf("bignum: invalid float64 %v", v))
	}
	if v < 0 {
		return normalize(-1, -v, 0)
	}
	return normalize(1, v, 0)
}

func Parse(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Zero(), fmt.Errorf("bignum: empty value")
	}

	sign := int8(1)
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		sign = -1
		s = s[1:]
	}
	if s == "" {
		return Zero(), fmt.Errorf("bignum: missing digits")
	}

	exponent := int64(0)
	if idx := strings.IndexAny(s, "eE"); idx >= 0 {
		expPart := s[idx+1:]
		parsed, err := strconv.ParseInt(expPart, 10, 64)
		if err != nil {
			return Zero(), fmt.Errorf("bignum: invalid exponent %q", expPart)
		}
		exponent = parsed
		s = s[:idx]
	}

	if s == "" {
		return Zero(), fmt.Errorf("bignum: missing significand")
	}

	dot := strings.IndexByte(s, '.')
	digitsBeforeDot := len(s)
	if dot >= 0 {
		digitsBeforeDot = dot
		s = s[:dot] + s[dot+1:]
	}
	if s == "" {
		return Zero(), fmt.Errorf("bignum: missing digits")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return Zero(), fmt.Errorf("bignum: invalid digit %q", r)
		}
	}

	trimmed := strings.TrimLeft(s, "0")
	if trimmed == "" {
		return Zero(), nil
	}
	leadingZeros := len(s) - len(trimmed)
	if leadingZeros < digitsBeforeDot {
		exponent += int64(digitsBeforeDot - leadingZeros - 1)
	} else {
		exponent -= int64(leadingZeros - digitsBeforeDot + 1)
	}

	const maxMantissaDigits = 17
	if len(trimmed) > maxMantissaDigits {
		trimmed = trimmed[:maxMantissaDigits]
	}
	mantissaText := trimmed[:1]
	if len(trimmed) > 1 {
		mantissaText += "." + trimmed[1:]
	}
	mantissa, err := strconv.ParseFloat(mantissaText, 64)
	if err != nil {
		return Zero(), fmt.Errorf("bignum: invalid significand %q", mantissaText)
	}
	return normalize(sign, mantissa, exponent), nil
}

func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Decimal) String() string {
	if d.IsZero() {
		return "0"
	}
	mantissa := d.mantissa
	if d.sign < 0 {
		mantissa = -mantissa
	}
	return strconv.FormatFloat(mantissa, 'f', -1, 64) + "e" + strconv.FormatInt(d.exponent, 10)
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*d = Zero()
		return nil
	}

	var raw string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
	} else {
		raw = string(data)
	}

	parsed, err := Parse(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Decimal) Sign() int {
	return int(d.sign)
}

func (d Decimal) IsZero() bool {
	return d.sign == 0
}

func (d Decimal) Eq(other Decimal) bool {
	return d.Cmp(other) == 0
}

func (d Decimal) LT(other Decimal) bool {
	return d.Cmp(other) < 0
}

func (d Decimal) LTE(other Decimal) bool {
	return d.Cmp(other) <= 0
}

func (d Decimal) GT(other Decimal) bool {
	return d.Cmp(other) > 0
}

func (d Decimal) GTE(other Decimal) bool {
	return d.Cmp(other) >= 0
}

func (d Decimal) Cmp(other Decimal) int {
	if d.sign != other.sign {
		if d.sign < other.sign {
			return -1
		}
		return 1
	}
	if d.sign == 0 {
		return 0
	}

	cmp := absCmp(d, other)
	if d.sign < 0 {
		cmp = -cmp
	}
	return cmp
}

func (d Decimal) Abs() Decimal {
	if d.sign < 0 {
		d.sign = 1
	}
	return d
}

func (d Decimal) Neg() Decimal {
	d.sign = -d.sign
	return d
}

func (d Decimal) Add(other Decimal) Decimal {
	if d.IsZero() {
		return other
	}
	if other.IsZero() {
		return d
	}

	if d.sign == other.sign {
		large, small := absOrdered(d, other)
		diff := large.exponent - small.exponent
		if diff > maxAddExponentGap {
			return large
		}
		return normalize(large.sign, large.mantissa+small.mantissa*negativePowers10[diff], large.exponent)
	}

	absCmp := absCmp(d, other)
	if absCmp == 0 {
		return Zero()
	}

	large, small := absOrdered(d, other)
	diff := large.exponent - small.exponent
	if diff > maxAddExponentGap {
		return large
	}
	return normalize(large.sign, large.mantissa-small.mantissa*negativePowers10[diff], large.exponent)
}

func (d Decimal) AddInt(v int) Decimal {
	return d.Add(FromInt(v))
}

func (d Decimal) Sub(other Decimal) Decimal {
	return d.Add(other.Neg())
}

func (d Decimal) SubInt(v int) Decimal {
	return d.Sub(FromInt(v))
}

func (d Decimal) Mul(other Decimal) Decimal {
	if d.IsZero() || other.IsZero() {
		return Zero()
	}
	sign := d.sign * other.sign
	return normalize(sign, d.mantissa*other.mantissa, d.exponent+other.exponent)
}

func (d Decimal) MulInt(v int) Decimal {
	return d.Mul(FromInt(v))
}

func (d Decimal) Div(other Decimal) Decimal {
	if other.IsZero() {
		panic("bignum: division by zero")
	}
	if d.IsZero() {
		return Zero()
	}
	sign := d.sign * other.sign
	return normalize(sign, d.mantissa/other.mantissa, d.exponent-other.exponent)
}

func (d Decimal) DivInt(v int) Decimal {
	return d.Div(FromInt(v))
}

func (d Decimal) Log10() float64 {
	if d.IsZero() {
		return math.Inf(-1)
	}
	return float64(d.exponent) + math.Log10(d.mantissa)
}

func (d Decimal) Float64() float64 {
	if d.IsZero() {
		return 0
	}
	v := d.mantissa * math.Pow10(int(d.exponent))
	if d.sign < 0 {
		v = -v
	}
	return v
}

// Ceil returns the smallest integer Decimal >= d. For non-negative d this
// rounds upward (1.3 -> 2). For negative d, ceiling truncates toward zero
// (-0.3 -> 0, -1.7 -> -1). Values with exponent >= 16 already exceed
// float64 significand precision and are treated as integers.
func (d Decimal) Ceil() Decimal {
	if d.IsZero() {
		return d
	}
	if d.exponent >= 16 {
		return d
	}
	return FromFloat64(math.Ceil(d.Float64()))
}

func Max(a, b Decimal) Decimal {
	if a.GTE(b) {
		return a
	}
	return b
}

func Min(a, b Decimal) Decimal {
	if a.LTE(b) {
		return a
	}
	return b
}

func normalize(sign int8, mantissa float64, exponent int64) Decimal {
	if mantissa == 0 {
		return Zero()
	}
	if mantissa < 0 {
		sign = -sign
		mantissa = -mantissa
	}

	shift := int64(math.Floor(math.Log10(mantissa)))
	mantissa /= math.Pow10(int(shift))
	exponent += shift

	if mantissa >= 10 {
		mantissa /= 10
		exponent++
	}
	if mantissa < 1 {
		mantissa *= 10
		exponent--
	}
	if mantissa == 0 {
		return Zero()
	}

	return Decimal{sign: sign, mantissa: mantissa, exponent: exponent}
}

func absCmp(a, b Decimal) int {
	if a.exponent != b.exponent {
		if a.exponent < b.exponent {
			return -1
		}
		return 1
	}
	if a.mantissa < b.mantissa {
		return -1
	}
	if a.mantissa > b.mantissa {
		return 1
	}
	return 0
}

func absOrdered(a, b Decimal) (Decimal, Decimal) {
	if absCmp(a, b) >= 0 {
		return a, b
	}
	return b, a
}
