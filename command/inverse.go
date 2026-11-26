package command

import (
	"fmt"
	"math/big"
	"reflect"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
)

// Inverse keys name the inverse of a projection encoder: the Matter value back
// to the SemReg SI value of the operation's argument.
const (
	InverseCentiCelsiusToKelvin = "centicelsius_to_kelvin"
	InverseMilliampereToAmpere  = "milliampere_to_ampere"
)

type inverse struct {
	argument semreg.DefinitionID
	unit     semreg.DefinitionID
	exponent int32
	offset   int64
}

var inverses = map[string]inverse{
	InverseCentiCelsiusToKelvin: {argument: "thermal.setpoint.temperature_setpoint", unit: "unit.kelvin", exponent: -2, offset: 27315},
	InverseMilliampereToAmpere:  {argument: "evse.limit.allocated_current", unit: "unit.ampere", exponent: -3},
}

func (i inverse) field(matter int64) semreg.TypedField {
	coefficient := new(big.Int).Add(big.NewInt(matter), big.NewInt(i.offset))
	return semreg.TypedField{ID: i.argument, Value: semreg.Value{
		Kind: semreg.ValueQuantity,
		Quantity: &semreg.Quantity{
			Number: semreg.Decimal{Coefficient: coefficient.String(), Exponent10: i.exponent},
			Unit:   i.unit,
		},
	}}
}

func integerOf(v any) (int64, error) {
	rv := reflect.ValueOf(v)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return 0, fmt.Errorf("value is null")
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return 0, fmt.Errorf("value is null")
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > 1<<63-1 {
			return 0, fmt.Errorf("value %d does not fit a signed 64-bit integer", u)
		}
		return int64(u), nil
	case reflect.Struct:
		return singleInteger(rv)
	}
	return 0, fmt.Errorf("value of kind %s is not an integer", rv.Kind())
}

func singleInteger(rv reflect.Value) (int64, error) {
	var found reflect.Value
	for i := 0; i < rv.NumField(); i++ {
		if !rv.Type().Field(i).IsExported() {
			continue
		}
		if found.IsValid() {
			return 0, fmt.Errorf("request carries more than one field")
		}
		found = rv.Field(i)
	}
	if !found.IsValid() {
		return 0, fmt.Errorf("request carries no field")
	}
	switch found.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return integerOf(found.Interface())
	}
	return 0, fmt.Errorf("request field of kind %s is not an integer", found.Kind())
}
