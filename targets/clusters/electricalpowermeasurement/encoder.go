package electricalpowermeasurement

import (
	"errors"
	"math/big"
)

// ErrInput is returned for a coefficient that is not a decimal integer.
var ErrInput = errors.New("electrical power measurement: input is not a decimal number")

// ErrNotIntegral is returned when the milliampere value is not an integer; the
// value is reported, never rounded.
var ErrNotIntegral = errors.New("electrical power measurement: milliampere value is not integral")

// ErrPercent is returned when a ratio has no exact percent100ths value inside
// the attribute's range; the figure is refused, never rounded.
var ErrPercent = errors.New("electrical power measurement: ratio has no exact percent100ths value")

// ErrRange is returned when the integer is outside the target's representation.
var ErrRange = errors.New("electrical power measurement: milliampere value is out of range")

var (
	currentMinimum = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 62))
	currentMaximum = new(big.Int).Lsh(big.NewInt(1), 62)
	thousand       = big.NewInt(1000)
	percentMaximum = big.NewInt(Percent100thsMaximum)
	percentScale   = big.NewInt(Percent100thsPerRatio)
)

// AmpereToMilliampere is the encoder of the ActiveCurrent row: the amperes
// value coefficient x 10^exponent10 multiplied by 1000, exactly, with an
// integral result required and the target's signed range.
func AmpereToMilliampere(coefficient string, exponent10 int) (string, error) {
	return ToMilli(coefficient, exponent10)
}

// ToMilli converts a value in a base unit to its thousandth, exactly, with an
// integral result and the signed range the measurement attributes state.
func ToMilli(coefficient string, exponent10 int) (string, error) {
	n, err := scaleExact(coefficient, exponent10, thousand)
	if err != nil {
		return "", err
	}
	if n.Cmp(currentMinimum) < 0 || n.Cmp(currentMaximum) > 0 {
		return "", ErrRange
	}
	return n.String(), nil
}

// RatioToPercent100ths converts a ratio to percent100ths, exactly. A result
// that is not an integer, is negative or exceeds the attribute's maximum is
// refused.
func RatioToPercent100ths(coefficient string, exponent10 int) (uint16, error) {
	n, err := scaleExact(coefficient, exponent10, percentScale)
	if err != nil {
		if errors.Is(err, ErrNotIntegral) {
			return 0, ErrPercent
		}
		return 0, err
	}
	if n.Sign() < 0 || n.Cmp(percentMaximum) > 0 {
		return 0, ErrPercent
	}
	return uint16(n.Uint64()), nil
}

func scaleExact(coefficient string, exponent10 int, factor *big.Int) (*big.Int, error) {
	n, ok := new(big.Int).SetString(coefficient, 10)
	if !ok {
		return nil, ErrInput
	}
	n.Mul(n, factor)
	if exponent10 >= 0 {
		return n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent10)), nil)), nil
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponent10)), nil)
	quotient, remainder := new(big.Int).QuoRem(n, divisor, new(big.Int))
	if remainder.Sign() != 0 {
		return nil, ErrNotIntegral
	}
	return quotient, nil
}
