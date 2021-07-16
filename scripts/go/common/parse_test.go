package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInsnDescriptionLine(t *testing.T) {
	testcases := []struct {
		x        string
		ok       bool
		expected *InsnDescription
	}{
		{
			x:  "12345678 foo EMPTY",
			ok: true,
			expected: &InsnDescription{
				Word:     0x12345678,
				Mnemonic: "foo",
				Format: &InsnFormat{
					Args: nil,
				},
			},
		},
		{
			x:  "20000000 ll.w                   DJSk14",
			ok: true,
			expected: &InsnDescription{
				Word:     0x20000000,
				Mnemonic: "ll.w",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{{Offset: 10, Width: 14}}},
					},
				},
			},
		},
		{
			x:  "2ac00000 preld                  JUd5Sk12",
			ok: true,
			expected: &InsnDescription{
				Word:     0x2ac00000,
				Mnemonic: "preld",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindUnsignedImm, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{{Offset: 10, Width: 12}}},
					},
				},
			},
		},
		{
			x:  "40000000 beqz                   JSd5k16",
			ok: true,
			expected: &InsnDescription{
				Word:     0x40000000,
				Mnemonic: "beqz",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{
							{Offset: 0, Width: 5},
							{Offset: 10, Width: 16},
						}},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		actual, err := ParseInsnDescriptionLine(tc.x)
		if tc.ok {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		} else {
			assert.Error(t, err)
			assert.Nil(t, actual)
		}
	}
}
