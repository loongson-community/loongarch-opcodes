package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/loongson-community/loongarch-opcodes/scripts/go/common"
)

func main() {
	inputs := os.Args[1:]

	descs, err := readInsnDescs(inputs)
	if err != nil {
		panic(err)
	}

	formats := gatherFormats(descs)

	sort.Slice(descs, func(i int, j int) bool {
		return descs[i].Word < descs[j].Word
	})

	sort.Slice(formats, func(i int, j int) bool {
		return formats[i].CanonicalRepr() < formats[j].CanonicalRepr()
	})

	fmt.Printf("package loong\n\n")
	fmt.Printf("import \"cmd/internal/obj\"\n\n")

	emitInsnFormatTypes(formats)

	for _, f := range formats {
		emitValidatorForFormat(f)
		emitEncoderForFormat(f)
	}

	emitInsnEncodings(descs)
}

func readInsnDescs(paths []string) ([]*common.InsnDescription, error) {
	var result []*common.InsnDescription
	for _, path := range paths {
		descs, err := common.ReadInsnDescriptionFile(path)
		if err != nil {
			return nil, err
		}
		result = append(result, descs...)
	}
	return result, nil
}

func gatherFormats(descs []*common.InsnDescription) []*common.InsnFormat {
	formatsSet := make(map[string]*common.InsnFormat)
	for _, d := range descs {
		canonicalFormatName := d.Format.CanonicalRepr()
		if _, ok := formatsSet[canonicalFormatName]; !ok {
			formatsSet[canonicalFormatName] = d.Format
		}
	}

	result := make([]*common.InsnFormat, 0, len(formatsSet))
	for _, f := range formatsSet {
		result = append(result, f)
	}

	return result
}

func emitInsnFormatTypes(fmts []*common.InsnFormat) {
	fmt.Printf("type insnFormat int\n\nconst (\n")
	fmt.Printf("\tinsnFormatUnknown insnEncoding = iota\n")

	for _, f := range fmts {
		fmt.Printf("\tinsnFormat%s\n", f.CanonicalRepr())
	}

	fmt.Printf(")\n\n")
}

func goOpcodeNameForInsn(mnemonic string) string {
	// e.g. slli.w => ASLLIW
	tmp := strings.ReplaceAll(mnemonic, ".", "")
	tmp = strings.ReplaceAll(tmp, "_", "")
	tmp = strings.ToUpper(tmp)
	return "A" + tmp
}

func emitInsnEncodings(descs []*common.InsnDescription) {
	fmt.Printf("type encoding struct {\n")
	fmt.Printf("\tbits uint32\n")
	fmt.Printf("\tfmt  insnFormat\n")
	fmt.Printf("}\n\n")
	fmt.Printf("var encodings = [ALAST & obj.AMask]encoding{\n")

	for _, d := range descs {
		goOpcodeName := goOpcodeNameForInsn(d.Mnemonic)
		formatName := "insnFormat" + d.Format.CanonicalRepr()

		fmt.Printf(
			"\t%s & obj.AMask: {bits: 0x%08x, fmt: %s},\n",
			goOpcodeName,
			d.Word,
			formatName,
		)
	}

	fmt.Printf("}\n")
}

func emitValidatorForFormat(f *common.InsnFormat) {
	formatName := f.CanonicalRepr()
	funcName := "validate" + formatName

	argNames := make([]string, len(f.Args))
	for i, a := range f.Args {
		argNames[i] = strings.ToLower(a.CanonicalRepr())
	}

	fmt.Printf("func %s(", funcName)
	for i, p := range argNames {
		var sep string
		if i > 0 {
			sep = ", "
		}
		fmt.Printf("%s%s uint32", sep, p)
	}
	fmt.Printf(") error {\n")

	// things to emit:
	//
	// for every arg X:
	//     if err := want<arg type>("argX", argX); err != nil {
	//         return err
	//     }
	for argIdx, a := range f.Args {
		argParamName := argNames[argIdx]

		fmt.Printf("\tif err := ")

		switch a.Kind {
		case common.ArgKindIntReg:
			fmt.Printf("wantIntReg(%s)", argParamName)

		case common.ArgKindFPReg:
			fmt.Printf("wantFPReg(%s)", argParamName)

		case common.ArgKindFCCReg:
			fmt.Printf("wantFCCReg(%s)", argParamName)

		case common.ArgKindSignedImm,
			common.ArgKindUnsignedImm:
			// want[Un]signedImm(argX, width)
			var wantFuncName string
			if a.Kind == common.ArgKindSignedImm {
				wantFuncName = "wantSignedImm"
			} else {
				wantFuncName = "wantUnsignedImm"
			}

			fmt.Printf("%s(%s, %d)", wantFuncName, argParamName, a.TotalWidth())
		}

		fmt.Printf("; err != nil {\n\t\treturn err\n\t}\n")
	}

	fmt.Printf("\treturn nil\n}\n\n")
}

func emitEncoderForFormat(f *common.InsnFormat) {
	formatName := f.CanonicalRepr()
	funcName := "encode" + formatName

	argNames := make([]string, len(f.Args))
	for i, a := range f.Args {
		argNames[i] = strings.ToLower(a.CanonicalRepr())
	}

	// func encodeXXX(bits uint32, params...) uint32 {
	fmt.Printf("func %s(bits uint32", funcName)
	for _, p := range argNames {
		fmt.Printf(", %s uint32", p)
	}
	fmt.Printf(") uint32 {\n")

	// things to emit:
	//
	// for every arg X:
	//     if only one slot:
	//         bits |= argX << slot offset
	//
	//     else for every slot in arg:
	//         slot value = (extract from argX)
	//         bits |= slot value << slot offset
	for argIdx, a := range f.Args {
		argParamName := argNames[argIdx]

		if len(a.Slots) == 1 {
			fmt.Printf("\tbits |= %s", argParamName)
			offset := int(a.Slots[0].Offset)
			if offset != 0 {
				fmt.Printf(" << %d", offset)
			}
			fmt.Printf("\n")
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

				fmt.Printf("\tbits |= %s", argParamName)

				if remainingBits > 0 {
					fmt.Printf(" >> %d", remainingBits)
				}

				fmt.Printf(" & %#x\n", mask)
			}
		}
	}

	fmt.Printf("\treturn bits\n}\n\n")
}
