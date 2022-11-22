package utils

import (
	"testing"

	utilpointer "k8s.io/utils/pointer"
)

func TestComputeObjectHash(t *testing.T) {
	type shadow struct {
		x int
		y *int
		z *shadow
	}

	for index, testCase := range []struct {
		left  interface{}
		right interface{}
		equal bool
	}{
		{
			left:  1,
			right: 2,
			equal: false,
		},
		{
			left:  1,
			right: utilpointer.Int(2),
			equal: false,
		},
		{
			left:  1,
			right: utilpointer.Int(1),
			equal: false,
		},
		{
			left:  1,
			right: 1,
			equal: true,
		},
		{
			left:  utilpointer.Int(1),
			right: utilpointer.Int(1),
			equal: true,
		},
		{
			left: shadow{
				x: 1,
				y: utilpointer.Int(1),
			},
			right: shadow{
				x: 1,
				y: utilpointer.Int(1),
			},
			equal: true,
		},
		{
			left: shadow{
				x: 1,
				z: &shadow{
					x: 1,
				},
			},
			right: shadow{
				x: 1,
				z: &shadow{
					x: 1,
				},
			},
			equal: true,
		},
	} {
		leftHash := ComputeObjectHash(testCase.left)
		rightHash := ComputeObjectHash(testCase.right)
		result := leftHash == rightHash
		if result != testCase.equal {
			t.Errorf("Hash result unexpected %d %v(%s) -> %v(%s), should %s be equal", index, testCase.left, leftHash, testCase.right, rightHash, func() string {
				if testCase.equal {
					return ""
				}
				return "not"
			}())
		}
	}
}
