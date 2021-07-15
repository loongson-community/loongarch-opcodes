package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsnFormat(t *testing.T) {
	testcases := []struct {
		x                     InsnFormat
		expectedCanonicalRepr string
	}{
		{
			x: InsnFormat{
				Args: nil,
			},
			expectedCanonicalRepr: "EMPTY",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindReg,
						Slots: []*Slot{
							{Offset: 0, Width: 5},
						},
					},
				},
			},
			expectedCanonicalRepr: "D",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindFCCReg,
						Slots: []*Slot{
							{Offset: 5, Width: 3},
						},
					},
				},
			},
			expectedCanonicalRepr: "Cj",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindSignedImm,
						Slots: []*Slot{
							{Offset: 0, Width: 5},
						},
					},
				},
			},
			expectedCanonicalRepr: "Sd5",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindUnsignedImm,
						Slots: []*Slot{
							{Offset: 10, Width: 5},
						},
					},
				},
			},
			expectedCanonicalRepr: "Uk5",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindSignedImm,
						Slots: []*Slot{
							{Offset: 0, Width: 5},
							{Offset: 10, Width: 16},
						},
					},
				},
			},
			expectedCanonicalRepr: "Sd5k16",
		},
		{
			x: InsnFormat{
				Args: []*Arg{
					{
						Kind: ArgKindReg,
						Slots: []*Slot{
							{Offset: 0, Width: 5},
						},
					},
					{
						Kind: ArgKindReg,
						Slots: []*Slot{
							{Offset: 5, Width: 5},
						},
					},
					{
						Kind: ArgKindUnsignedImm,
						Slots: []*Slot{
							{Offset: 10, Width: 6},
						},
					},
					{
						Kind: ArgKindUnsignedImm,
						Slots: []*Slot{
							{Offset: 16, Width: 6},
						},
					},
				},
			},
			expectedCanonicalRepr: "DJUk6Um6",
		},
	}

	for _, tc := range testcases {
		err := tc.x.Validate()
		assert.NoError(t, err)

		actualRepr := tc.x.CanonicalRepr()
		assert.Equal(t, tc.expectedCanonicalRepr, actualRepr)

		roundtrip, err := ParseInsnFormat(actualRepr)
		assert.NoError(t, err)
		assert.Equal(t, &tc.x, roundtrip, "canonical repr should survive round-trip")
	}
}
