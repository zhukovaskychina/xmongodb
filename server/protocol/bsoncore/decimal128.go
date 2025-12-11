// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"errors"
	"fmt"
	"math"
	"math/big"
)

// Decimal128 represents a BSON Decimal128 value.
type Decimal128 struct {
	h, l uint64
}

// NewDecimal128 creates a Decimal128 using the provide high and low uint64s.
func NewDecimal128(h, l uint64) Decimal128 {
	return Decimal128{h: h, l: l}
}

// GetBytes returns the underlying bytes of the BSON decimal128 as two uint64 values.
func (d Decimal128) GetBytes() (h, l uint64) {
	return d.h, d.l
}

// String returns the string representation of the decimal value.
func (d Decimal128) String() string {
	// Simplified implementation - just return hex representation for now
	return fmt.Sprintf("Decimal128(%016x%016x)", d.h, d.l)
}

// IsNaN returns if the decimal is NaN.
func (d Decimal128) IsNaN() bool {
	return (d.h&0x7c00000000000000 == 0x7c00000000000000)
}

// IsInf returns if the decimal is ±Inf.
func (d Decimal128) IsInf() int {
	if d.h&0x7c00000000000000 != 0x7800000000000000 {
		return 0
	}
	if d.h&0x8000000000000000 == 0x8000000000000000 {
		return -1
	}
	return 1
}

// ParseDecimal128 parses a string representation of a decimal128 value.
func ParseDecimal128(s string) (Decimal128, error) {
	// Simplified implementation
	return Decimal128{}, errors.New("ParseDecimal128 not fully implemented")
}

// ParseDecimal128FromBigInt creates a Decimal128 from a big.Int.
func ParseDecimal128FromBigInt(i *big.Int) (Decimal128, bool) {
	// Simplified implementation
	return Decimal128{}, false
}

// BigInt converts the Decimal128 to a big.Int.
func (d Decimal128) BigInt() (*big.Int, int, bool) {
	// Simplified implementation
	return big.NewInt(0), 0, false
}

// Decimal128NaN represents NaN for Decimal128.
var Decimal128NaN = Decimal128{h: 0x7c00000000000000, l: 0}

// Decimal128PosInf represents +Inf for Decimal128.
var Decimal128PosInf = Decimal128{h: 0x7800000000000000, l: 0}

// Decimal128NegInf represents -Inf for Decimal128.
var Decimal128NegInf = Decimal128{h: 0xf800000000000000, l: 0}

// ParseDecimal128 parses the given string and returns a Decimal128.
func (d Decimal128) AsFloat64() (float64, bool) {
	// Simplified conversion
	if d.IsNaN() {
		return math.NaN(), true
	}
	if inf := d.IsInf(); inf != 0 {
		if inf > 0 {
			return math.Inf(1), true
		}
		return math.Inf(-1), true
	}
	return 0, false
}
