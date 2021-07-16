package main

import (
	"go/token"
	"go/types"
	"os"
	"sort"
	"strings"

	"github.com/goplus/gox"

	"github.com/loongson-community/loongarch-opcodes/scripts/go/common"
)

func main() {
	inputs := os.Args[1:]

	formats, err := gatherFormats(inputs)
	if err != nil {
		panic(err)
	}

	sort.Slice(formats, func(i int, j int) bool {
		return formats[i].CanonicalRepr() < formats[j].CanonicalRepr()
	})

	pkg := gox.NewPackage("", "loong", nil)
	for _, f := range formats {
		emitEncoderForFormat(pkg, f)
	}

	err = gox.WriteTo(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
}

func gatherFormats(paths []string) ([]*common.InsnFormat, error) {
	formatsSet := make(map[string]*common.InsnFormat)
	for _, path := range paths {
		descs, err := common.ReadInsnDescriptionFile(path)
		if err != nil {
			return nil, err
		}

		for _, d := range descs {
			canonicalFormatName := d.Format.CanonicalRepr()
			if _, ok := formatsSet[canonicalFormatName]; !ok {
				formatsSet[canonicalFormatName] = d.Format
			}
		}
	}

	result := make([]*common.InsnFormat, 0, len(formatsSet))
	for _, f := range formatsSet {
		result = append(result, f)
	}

	return result, nil
}

var tyUint32 = types.Universe.Lookup("uint32").Type().(*types.Basic)

func emitEncoderForFormat(pkg *gox.Package, f *common.InsnFormat) {
	formatName := f.CanonicalRepr()
	funcName := "encode" + formatName

	returnParam := pkg.NewParam("", tyUint32)
	returnTuple := gox.NewTuple(returnParam)

	params := make([]*gox.Param, len(f.Args)+1)
	params[0] = pkg.NewParam("bits", tyUint32)
	for i, a := range f.Args {
		params[i+1] = pkg.NewParam(strings.ToLower(a.CanonicalRepr()), tyUint32)
	}
	paramsTuple := gox.NewTuple(params...)

	// func encodeXXX(bits uint32, params...) uint32 {
	bldr := pkg.NewFunc(nil, funcName, paramsTuple, returnTuple, false).BodyStart(pkg)

	// things to emit:
	//
	// for every arg X:
	//     if only one slot:
	//         bits |= argX << slot offset
	//
	//     else for every slot in arg:
	//         slot value = (extract from argX)
	//         bits |= slot value << slot offset
	//
	// seems gox doesn't have in-place assignments, so some duplication is necessary
	for argIdx, a := range f.Args {
		bitsParam := params[0]
		argParam := params[argIdx+1]

		if len(a.Slots) == 1 {
			offset := int(a.Slots[0].Offset)
			// bits = bits | (argX << offset)
			// lvalue
			bldr = bldr.VarRef(bitsParam)

			// rvalue expr
			// stack: bits argX
			bldr = bldr.Val(bitsParam).
				Val(argParam)

			if offset != 0 {
				// stack: bits (argX << offset)
				bldr = bldr.Val(offset).BinaryOp(token.SHL)
			}

			// stack: (bits | processedArgX)
			bldr = bldr.BinaryOp(token.OR)

			// assign
			bldr = bldr.Assign(1)
		} else {
			// remainingBits is shift amount to extract the current slot from arg
			//
			// take example of Sd5k16:
			//
			// Sd5k16 = (MSB) DDDDDKKKKKKKKKKKKKKKK (LSB)
			//
			// initially remainingBits = 5+16
			//
			// consume from left to right:
			//
			// slot d5: remainingBits = 16
			// thus d5 = (sd5k16 >> 16) & 0b11111
			// emit bits |= (d5 expr above)
			//
			// slot k16: remainingBits = 0
			// thus k16 = (sd5k16 >> 0) & 0b1111111111111111
			//          = sd5k16 & 0b1111111111111111
			// emit bits |= (k16 expr above)
			remainingBits := int(a.TotalWidth())
			for _, s := range a.Slots {
				remainingBits -= int(s.Width)
				mask := int((1 << s.Width) - 1)

				// lvalue
				bldr = bldr.VarRef(bitsParam)

				// rvalue expr
				// stack: bits argX
				bldr = bldr.Val(bitsParam).Val(argParam)

				if remainingBits > 0 {
					// stack: bits (argX >> remainingBits)
					bldr = bldr.Val(remainingBits).BinaryOp(token.SHR)
				}

				// stack: bits ((tmp above) & mask)
				bldr = bldr.Val(mask).BinaryOp(token.AND)

				// stack: (bits | above)
				bldr = bldr.BinaryOp(token.OR)

				// assign
				bldr = bldr.Assign(1)
			}
		}
	}

	// return bits
	bldr = bldr.Val(params[0]).Return(1).EndStmt()

	// }
	_ = bldr.End()
}
