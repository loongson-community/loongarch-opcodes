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

	gox.SetDebug(true)
	pkg := gox.NewPackage("", "loong", nil)
	prepareScope(pkg)
	for _, f := range formats {
		emitValidatorForFormat(pkg, f)
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

var (
	tyInt    = types.Universe.Lookup("int").Type().(*types.Basic)
	tyUint32 = types.Universe.Lookup("uint32").Type().(*types.Basic)
	tyError  = types.Universe.Lookup("error").Type()
)

func prepareScope(pkg *gox.Package) {
	// func wantIntReg(uint32) error
	pkg.NewFunc(
		nil,
		"wantIntReg",
		gox.NewTuple(pkg.NewParam("", tyUint32)),
		gox.NewTuple(pkg.NewParam("", tyError)),
		false,
	)

	// func wantFPReg(uint32) error
	pkg.NewFunc(
		nil,
		"wantFPReg",
		gox.NewTuple(pkg.NewParam("", tyUint32)),
		gox.NewTuple(pkg.NewParam("", tyError)),
		false,
	)

	// func wantFCCReg(uint32) error
	pkg.NewFunc(
		nil,
		"wantFCCReg",
		gox.NewTuple(pkg.NewParam("", tyUint32)),
		gox.NewTuple(pkg.NewParam("", tyError)),
		false,
	)

	// func wantSignedImm(uint32, int) error
	pkg.NewFunc(
		nil,
		"wantSignedImm",
		gox.NewTuple(pkg.NewParam("", tyUint32), pkg.NewParam("", tyInt)),
		gox.NewTuple(pkg.NewParam("", tyError)),
		false,
	)

	// func wantUnsignedImm(uint32, int) error
	pkg.NewFunc(
		nil,
		"wantUnsignedImm",
		gox.NewTuple(pkg.NewParam("", tyUint32), pkg.NewParam("", tyInt)),
		gox.NewTuple(pkg.NewParam("", tyError)),
		false,
	)
}

func emitValidatorForFormat(pkg *gox.Package, f *common.InsnFormat) {
	formatName := f.CanonicalRepr()
	funcName := "validate" + formatName

	returnParam := pkg.NewParam("", tyError)
	returnTuple := gox.NewTuple(returnParam)

	params := make([]*gox.Param, len(f.Args))
	for i, a := range f.Args {
		params[i] = pkg.NewParam(strings.ToLower(a.CanonicalRepr()), tyUint32)
	}
	paramsTuple := gox.NewTuple(params...)

	// func validateXXX(params...) error {
	bldr := pkg.NewFunc(nil, funcName, paramsTuple, returnTuple, false).BodyStart(pkg)

	ctxRef := func(name string) gox.Ref {
		_, o := pkg.CB().Scope().LookupParent(name, token.NoPos)
		return o
	}

	// things to emit:
	//
	// for every arg X:
	//     if err := want<arg type>("argX", argX); err != nil {
	//         return err
	//     }
	for argIdx, a := range f.Args {
		argParam := params[argIdx]

		bldr = bldr.If()
		bldr = bldr.DefineVarStart("err")

		switch a.Kind {
		case common.ArgKindIntReg:
			// wantIntReg(argX)
			bldr = bldr.Val(pkg.Ref("wantIntReg")).
				Val(argParam).
				Call(1)

		case common.ArgKindFPReg:
			// wantFPReg(argX)
			bldr = bldr.Val(pkg.Ref("wantFPReg")).
				Val(argParam).
				Call(1)

		case common.ArgKindFCCReg:
			// wantFCCReg(argX)
			bldr = bldr.Val(pkg.Ref("wantFCCReg")).
				Val(argParam).
				Call(1)

		case common.ArgKindSignedImm,
			common.ArgKindUnsignedImm:
			// want[Un]signedImm(argX, width)
			var wantFuncName string
			if a.Kind == common.ArgKindSignedImm {
				wantFuncName = "wantSignedImm"
			} else {
				wantFuncName = "wantUnsignedImm"
			}

			bldr = bldr.Val(pkg.Ref(wantFuncName)).
				Val(argParam).
				Val(int(a.TotalWidth())).
				Call(2)
		}

		bldr = bldr.EndInit(1)

		errRef := ctxRef("err")

		bldr = bldr.Val(errRef).Val(nil).BinaryOp(token.EQL) // XXX Val(nil) doesn't work
		bldr = bldr.Then()
		bldr = bldr.Val(errRef).Return(1).EndStmt()
		bldr = bldr.End()
	}

	// return nil
	bldr = bldr.Val(nil).Return(1).EndStmt()

	// }
	_ = bldr.End()
}

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
