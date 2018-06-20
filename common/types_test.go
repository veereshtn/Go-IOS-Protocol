package common

import (
	"math"
	"testing"
)

var intCases = []int{-1, 0, 1, math.MaxInt32}
var byteCases = [][]byte{{255, 255, 255, 255}, {0, 0, 0, 0}, {0, 0, 0, 1}, {127, 255, 255, 255}}

func TestIntToBytes(t *testing.T) {
	for k, v := range intCases {
		bs := IntToBytes(v)
		assert.Equal(t, byteCases[k], bs)
	}
}
