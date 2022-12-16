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
				Attribs: map[string]string{},
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
				Attribs: map[string]string{},
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
				Attribs: map[string]string{},
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
				Attribs: map[string]string{},
			},
		},
		{
			x:  "20000000 ll.w                   DJSk14     @orig_fmt=DJSk14ps2 @32 @atomics @primary @foo=amswap_db.d",
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
				OrigFormat: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{{Offset: 10, Width: 14}}, Post: PostprocessOp{Kind: PostprocessOpKindShl, Amount: 2}},
					},
				},
				Attribs: map[string]string{
					"32":      "true",
					"atomics": "true",
					"primary": "true",
					"foo":     "amswap_db.d",
				},
			},
		},
		{
			x:  "002c0000 sladd.d                DJKUa2          @orig_name=alsl.d @orig_fmt=DJKUa2pp1",
			ok: true,
			expected: &InsnDescription{
				Word:     0x002c0000,
				Mnemonic: "sladd.d",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 10, Width: 5}}},
						{Kind: ArgKindUnsignedImm, Slots: []*Slot{{Offset: 15, Width: 2}}},
					},
				},
				OrigFormat: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 10, Width: 5}}},
						{Kind: ArgKindUnsignedImm, Slots: []*Slot{{Offset: 15, Width: 2}}, Post: PostprocessOp{Kind: PostprocessOpKindAdd, Amount: 1}},
					},
				},
				Attribs: map[string]string{
					"orig_name": "alsl.d",
				},
			},
		},
		{
			x:  "31100000 vstelm.d               VdJSk8Un1",
			ok: true,
			expected: &InsnDescription{
				Word:     0x31100000,
				Mnemonic: "vstelm.d",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindVReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{{Offset: 10, Width: 8}}},
						{Kind: ArgKindUnsignedImm, Slots: []*Slot{{Offset: 18, Width: 1}}},
					},
				},
				Attribs: map[string]string{},
			},
		},
		{
			x:  "2c800000 xvld                   XdJSk12",
			ok: true,
			expected: &InsnDescription{
				Word:     0x2c800000,
				Mnemonic: "xvld",
				Format: &InsnFormat{
					Args: []*Arg{
						{Kind: ArgKindXReg, Slots: []*Slot{{Offset: 0, Width: 5}}},
						{Kind: ArgKindIntReg, Slots: []*Slot{{Offset: 5, Width: 5}}},
						{Kind: ArgKindSignedImm, Slots: []*Slot{{Offset: 10, Width: 12}}},
					},
				},
				Attribs: map[string]string{},
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
