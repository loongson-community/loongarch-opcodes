package main

import (
	"os"
	"sort"

	"github.com/loongson-community/loongarch-opcodes/scripts/go/common"
)

func main() {
	inputs := os.Args[1:]

	descs, err := common.ReadInsnDescs(inputs)
	if err != nil {
		panic(err)
	}

	sort.Slice(descs, func(i int, j int) bool {
		return descs[i].Word < descs[j].Word
	})

	var ectx common.EmitterCtx

	ectx.Emit("package loong\n\n")
	ectx.Emit("// NOTE: Paste into cpu.go and adjust as necessary (add pseudo-ops, etc.)\n\n")

	emitAnames(&ectx, descs)

	result := ectx.Finalize()
	os.Stdout.Write(result)
}

func emitAnames(ectx *common.EmitterCtx, descs []*common.InsnDescription) {
	ectx.Emit(`// LoongArch instruction mnemonics.
//
// If you modify this table, you MUST run 'go generate' to regenerate anames.go!
const (
`)

	for i, d := range descs {
		aname := common.GoAnameForInsn(d.Mnemonic)

		suffix := ""
		if i == 0 {
			suffix = "= obj.ABaseLoong + obj.A_ARCHSPECIFIC + iota"
		}
		ectx.Emit("\t%s%s\n", aname, suffix)
	}

	ectx.Emit("\n// TODO: Edit to include pseudo-ops.\n\n")

	ectx.Emit("\n\t// End marker\n\tALAST\n")
	ectx.Emit(")\n\n")
}
