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

// ErrRange is returned when the integer is outside the target's representation.
var ErrRange = errors.New("electrical power measurement: milliampere value is out of range")

var (
	currentMinimum = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 62))
	currentMaximum = new(big.Int).Lsh(big.NewInt(1), 62)
	thousand       = big.NewInt(1000)
)

// AmpereToMilliampere is the encoder of the ActiveCurrent row: the amperes
// value coefficient x 10^exponent10 multiplied by 1000, exactly, with an
// integral result required and the target's signed range.
func AmpereToMilliampere(coefficient string, exponent10 int) (string, error) {
	n, ok := new(big.Int).SetString(coefficient, 10)
	if !ok {
		return "", ErrInput
	}
	n.Mul(n, thousand)
	if exponent10 >= 0 {
		n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent10)), nil))
	} else {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponent10)), nil)
		quotient, remainder := new(big.Int).QuoRem(n, divisor, new(big.Int))
		if remainder.Sign() != 0 {
			return "", ErrNotIntegral
		}
		n = quotient
	}
	if n.Cmp(currentMinimum) < 0 || n.Cmp(currentMaximum) > 0 {
		return "", ErrRange
	}
	return n.String(), nil
}
