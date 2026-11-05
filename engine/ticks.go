package engine

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/shopspring/decimal"
)

type ticks int64

// Prices are also Redis sorted-set scores, which must represent each tick
// exactly. Keep the existing per-order input range for volumes as well.
const maxOrderTicks = (1 << 53) - 1

var errInvalidTicks = errors.New("value is not an exact integer tick count in range")

func (value ticks) String() string {
	return strconv.FormatInt(int64(value), 10)
}

// parseTicks also accepts legacy numbers written with a decimal point or exponent.
func parseTicks(raw string) (ticks, error) {
	if integer, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return ticks(integer), nil
	}
	value, err := parseDepthTicks(raw)
	if err != nil {
		return 0, err
	}
	if !value.IsInt64() {
		return 0, fmt.Errorf("%w: %s", errInvalidTicks, raw)
	}
	return ticks(value.Int64()), nil
}

// A level can contain many orders. Its total can exceed the range of one
// order or of int64, so keep the aggregate as a wider exact integer.
func parseDepthTicks(raw string) (*big.Int, error) {
	if integer, ok := new(big.Int).SetString(raw, 10); ok {
		return integer, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil || value.Exponent() < -64 || value.Exponent() > 64 || !value.Equal(value.Truncate(0)) {
		return nil, fmt.Errorf("%w: %s", errInvalidTicks, raw)
	}
	return value.BigInt(), nil
}

func (value *ticks) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 || (raw[0] != '-' && (raw[0] < '0' || raw[0] > '9')) {
		return fmt.Errorf("expected numeric tick count")
	}
	integer, err := parseTicks(string(raw))
	if err != nil {
		return err
	}
	*value = integer
	return nil
}

func toOrderTicks(value float64, accuracy int) (ticks, error) {
	if !(value > 0) || math.IsInf(value, 0) {
		return 0, errInvalidTicks
	}
	scaled := decimal.NewFromFloat(value).Shift(int32(accuracy))
	if !scaled.Equal(scaled.Truncate(0)) || scaled.GreaterThan(decimal.NewFromInt(maxOrderTicks)) {
		return 0, errInvalidTicks
	}
	return ticks(scaled.IntPart()), nil
}
