package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/loongson-community/loongarch-opcodes/scripts/go/common"
)

func main() {
	inputs := os.Args[1:]

	descs, err := common.ReadInsnDescs(inputs)
	if err != nil {
		panic(err)
	}

	descs = filterFPInsns(descs)

	formats := gatherFormats(descs)
	scs := gatherDistinctSlotCombinations(formats)

	sort.Slice(descs, func(i int, j int) bool {
		return descs[i].Word < descs[j].Word
	})

	sort.Slice(formats, func(i int, j int) bool {
		return formats[i].CanonicalRepr() < formats[j].CanonicalRepr()
	})

	ectx := common.EmitterCtx{
		DontGofmt: true,
	}

	ectx.Emit("/* Code generated by genqemutcgdefs from loongson-community/loongarch-opcodes; DO NOT EDIT. */\n")

	emitOpcEnum(&ectx, descs)

	emitSlotEncoders(&ectx, scs)

	for _, f := range formats {
		emitFmtEncoderFn(&ectx, f)
	}

	for _, d := range descs {
		emitTCGEmitterForInsn(&ectx, d)
	}

	result := ectx.Finalize()
	os.Stdout.Write(result)
}

////////////////////////////////////////////////////////////////////////////

func filterFPInsns(descs []*common.InsnDescription) []*common.InsnDescription {
	var result []*common.InsnDescription
	for _, d := range descs {
		isFPInsn := false
		for _, a := range d.Format.Args {
			switch a.Kind {
			case common.ArgKindFPReg, common.ArgKindFCCReg:
				isFPInsn = true
				break
			}
		}

		if isFPInsn {
			// QEMU TCG doesn't emit FP instructions for now, so don't
			// generate these to reduce code size.
			continue
		}

		result = append(result, d)
	}

	return result
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

const (
	slotD = 0
	slotJ = 5
	slotK = 10
	slotA = 15
	slotM = 16
)

func gatherDistinctSlotCombinations(fmts []*common.InsnFormat) []string {
	slotCombinationsSet := make(map[string]struct{})
	for _, f := range fmts {
		// skip EMPTY
		if len(f.Args) == 0 {
			continue
		}
		slotCombinationsSet[slotCombinationForFmt(f)] = struct{}{}
	}

	result := make([]string, 0, len(slotCombinationsSet))
	for sc := range slotCombinationsSet {
		result = append(result, sc)
	}
	sort.Strings(result)

	return result
}

// slot combination looks like "DJKM"
func slotCombinationForFmt(f *common.InsnFormat) string {

	var slots []int
	for _, a := range f.Args {
		for _, s := range a.Slots {
			slots = append(slots, int(s.Offset))
		}
	}
	sort.Ints(slots)

	var sb strings.Builder
	for _, s := range slots {
		switch s {
		case slotD:
			sb.WriteRune('D')
		case slotJ:
			sb.WriteRune('J')
		case slotK:
			sb.WriteRune('K')
		case slotA:
			sb.WriteRune('A')
		case slotM:
			sb.WriteRune('M')
		default:
			panic("should never happen")
		}
	}

	return sb.String()
}

func slotOffsetFromRune(s rune) int {
	switch s {
	case 'D', 'd':
		return slotD
	case 'J', 'j':
		return slotJ
	case 'K', 'k':
		return slotK
	case 'A', 'a':
		return slotA
	case 'M', 'm':
		return slotM
	default:
		panic("should never happen")
	}
}

////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////

// e.g. "amadd_db.w" -> "AMADD_DB_W"
func insnMnemonicToUpperCase(x string) string {
	return strings.ToUpper(strings.ReplaceAll(x, ".", "_"))
}

func insnMnemonicToEnumVariantName(x string) string {
	return fmt.Sprintf("OPC_%s", insnMnemonicToUpperCase(x))
}

func emitOpcEnum(ectx *common.EmitterCtx, descs []*common.InsnDescription) {
	ectx.Emit("\ntypedef enum {\n")

	for _, d := range descs {
		enumVariantName := insnMnemonicToEnumVariantName(d.Mnemonic)

		ectx.Emit(
			"    %s = 0x%08x,\n",
			enumVariantName,
			d.Word,
		)
	}

	ectx.Emit("} LoongArchInsn;\n")
}

func insnFieldNameForRegArg(a *common.Arg) string {
	return strings.ToLower(a.CanonicalRepr())
}

type fieldDesc struct {
	name string
	typ  string
}

func fieldDescsForArgs(args []*common.Arg) []fieldDesc {
	result := make([]fieldDesc, len(args))
	for i, a := range args {
		fieldName := insnFieldNameForRegArg(a)

		var typ string
		switch a.Kind {
		case common.ArgKindIntReg, common.ArgKindFPReg, common.ArgKindFCCReg:
			typ = "TCGReg"
		case common.ArgKindSignedImm:
			typ = "int32_t"
		case common.ArgKindUnsignedImm:
			typ = "uint32_t"
		}

		result[i] = fieldDesc{name: fieldName, typ: typ}
	}

	return result
}

func emitSlotEncoders(ectx *common.EmitterCtx, scs []string) {
	for _, sc := range scs {
		emitSlotEncoderFn(ectx, sc)
	}
}

func slotEncoderFnNameForSc(sc string) string {
	plural := ""
	if len(sc) > 1 {
		plural = "s"
	}

	return fmt.Sprintf("encode_%s_slot%s", strings.ToLower(sc), plural)
}

func emitSlotEncoderFn(ectx *common.EmitterCtx, sc string) {
	funcName := slotEncoderFnNameForSc(sc)
	scLower := strings.ToLower(sc)

	ectx.Emit("\nstatic int32_t %s(LoongArchInsn opc", funcName)
	for _, s := range scLower {
		ectx.Emit(", uint32_t %c", s)
	}
	ectx.Emit(")\n{\n")

	ectx.Emit("    return opc")

	for _, s := range scLower {
		offset := slotOffsetFromRune(s)

		ectx.Emit(" | %c", s)
		if offset > 0 {
			ectx.Emit(" << %d", offset)
		}
	}

	ectx.Emit(";\n}\n")
}

func fmtEncoderFnNameForInsnFormat(f *common.InsnFormat) string {
	return fmt.Sprintf("encode_%s_insn", strings.ToLower(f.CanonicalRepr()))
}

func emitFmtEncoderFn(ectx *common.EmitterCtx, f *common.InsnFormat) {
	// EMPTY doesn't need encoder after all
	if len(f.Args) == 0 {
		return
	}

	argFieldDescs := fieldDescsForArgs(f.Args)

	ectx.Emit("\nstatic int32_t %s(LoongArchInsn opc", fmtEncoderFnNameForInsnFormat(f))
	for i := range f.Args {
		ectx.Emit(", %s %s", argFieldDescs[i].typ, argFieldDescs[i].name)
	}
	ectx.Emit(")\n{\n")

	for i, a := range f.Args {
		varName := argFieldDescs[i].name
		ectx.Emit("    %s ", varName)

		switch a.Kind {
		case common.ArgKindIntReg, common.ArgKindFPReg:
			ectx.Emit("&= 0x1f")
		case common.ArgKindFCCReg:
			ectx.Emit("&= 0x7")
		case common.ArgKindSignedImm, common.ArgKindUnsignedImm:
			widthMask := (1 << a.TotalWidth()) - 1
			ectx.Emit("&= 0x%x", widthMask)
		default:
			panic("unreachable")
		}

		ectx.Emit(";\n")
	}

	// collect slot expressions
	slotExprs := make(map[uint]string)
	for argIdx, a := range f.Args {
		argVarName := argFieldDescs[argIdx].name

		if len(a.Slots) == 1 {
			slotExprs[a.Slots[0].Offset] = argVarName
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
			// emit (d5 expr above)
			//
			// slot k16: remainingBits = 0
			// thus k16 = (sd5k16 >> 0) & 0b1111111111111111
			//          = sd5k16 & 0b1111111111111111
			// emit (k16 expr above)
			remainingBits := int(a.TotalWidth())
			for _, s := range a.Slots {
				remainingBits -= int(s.Width)
				mask := int((1 << s.Width) - 1)

				var sb strings.Builder
				sb.WriteString(argVarName)

				if remainingBits > 0 {
					sb.WriteString(">>")
					sb.WriteString(strconv.Itoa(remainingBits))
				}

				sb.WriteString("&0x")
				sb.WriteString(strconv.FormatUint(uint64(mask), 16))

				slotExprs[s.Offset] = sb.String()
			}
		}
	}

	sc := slotCombinationForFmt(f)
	encFnName := slotEncoderFnNameForSc(sc)
	ectx.Emit("    return %s(enc.bits", encFnName)

	for _, s := range sc {
		offset := uint(slotOffsetFromRune(s))
		slotExpr, ok := slotExprs[offset]
		if !ok {
			panic("should never happen")
		}
		ectx.Emit(", %s", slotExpr)
	}

	ectx.Emit(");\n}\n")
}

// transform InsnDescription to syntax example, e.g. "addi.d d, j, sk12"
func insnSyntaxDescForInsn(d *common.InsnDescription) string {
	if len(d.Format.Args) == 0 {
		// special-case EMPTY
		return d.Mnemonic
	}

	var sb strings.Builder

	sb.WriteString(d.Mnemonic)
	for i, a := range d.Format.Args {
		if i == 0 {
			sb.WriteRune(' ')
		} else {
			sb.WriteString(", ")
		}

		sb.WriteString(strings.ToLower(a.CanonicalRepr()))
	}

	return sb.String()
}

func emitTCGEmitterForInsn(ectx *common.EmitterCtx, d *common.InsnDescription) {
	opc := insnMnemonicToEnumVariantName(d.Mnemonic)
	opcLower := strings.ToLower(opc)
	argFieldDescs := fieldDescsForArgs(d.Format.Args)

	// docstring line
	ectx.Emit("\n/* Emits the `%s` instruction. */\n", insnSyntaxDescForInsn(d))

	// function header
	declFirstLinePrefix := fmt.Sprintf("static void tcg_out_%s(", opcLower)

	ectx.Emit("%sTCGContext *s", declFirstLinePrefix)
	if len(d.Format.Args) == 0 {
		// special-case EMPTY
		ectx.Emit(")\n{\n")
		ectx.Emit("    tcg_out32(s, %s);\n", opc)
		ectx.Emit("}\n")
		return
	}

	for _, fd := range argFieldDescs {
		ectx.Emit(", %s %s", fd.typ, fd.name)
	}
	ectx.Emit(")\n{\n")

	// body and tail
	fmtEncoderFnName := fmtEncoderFnNameForInsnFormat(d.Format)

	ectx.Emit("    tcg_out32(s, %s(%s", fmtEncoderFnName, opc)
	for _, fd := range argFieldDescs {
		ectx.Emit(", %s", fd.name)
	}
	ectx.Emit("));\n")

	ectx.Emit("}\n")
}
