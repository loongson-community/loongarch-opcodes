package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"

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

	tp := tabPrinter{
		tabstop: 8,
	}

	for _, d := range descs {
		// atomic instructions may be special cases in Go assembly
		if strings.HasPrefix(d.Mnemonic, "am") {
			continue
		}

		// test cases for jumps are to be manually written so skip those too
		switch d.Mnemonic {
		case "beqz", "bnez", "bceqz", "bcnez",
			"jirl", "b", "bl",
			"beq", "bne", "bgt", "ble", "bgtu", "bleu":
			continue
		}

		tc := generateTestCase(d)

		tp.oneTab()
		tp.printf("%s", tc.mnemonic)
		tp.tabUntil(32)

		// Go assembly has arguments in reverse order.
		for i := len(tc.args) - 1; i >= 0; i-- {
			tca := tc.args[i]

			sep := ""
			if i < len(tc.args)-1 {
				sep = ", "
			}

			tp.printf("%s%s", sep, tca.repr)
		}

		tp.tabUntil(64)
		fmt.Printf("// %s", formatExpectedInsnWord(tc.expectedInsnWord))
		tp.newline()
	}
}

////////////////////////////////////////////////////////////////////////////

type tabPrinter struct {
	tabstop    int
	currentCol int
}

func (t *tabPrinter) printf(format string, a ...interface{}) {
	// Can't handle multi-byte characters, but we don't care in our case.
	n, _ := fmt.Printf(format, a...)
	t.currentCol += n
}

func (t *tabPrinter) oneTab() {
	fmt.Printf("\t")
	t.currentCol += t.tabstop - t.currentCol%t.tabstop
}

func (t *tabPrinter) tabUntil(col int) {
	for t.currentCol < col {
		t.oneTab()
	}
}

func (t *tabPrinter) newline() {
	fmt.Printf("\n")
	t.currentCol = 0
}

////////////////////////////////////////////////////////////////////////////

func formatExpectedInsnWord(w uint32) string {
	return fmt.Sprintf(
		"%02x%02x%02x%02x",
		w&0xff,
		(w>>8)&0xff,
		(w>>16)&0xff,
		(w>>24)&0xff,
	)
}

type testcaseData struct {
	mnemonic         string
	args             []testcaseArg
	expectedInsnWord uint32
}

type testcaseArg struct {
	val  uint32
	repr string
}

func generateTestCase(d *common.InsnDescription) testcaseData {
	rng := rngFromInsnDescription(d)

	args := make([]testcaseArg, len(d.Format.Args))
	for i, a := range d.Format.Args {
		var generatedArg testcaseArg
		switch a.Kind {
		case common.ArgKindIntReg:
			// remove R0, R21 (reserved) and R31 (g)
			val := uint32(rng.Intn(29)) + 1
			if val >= 21 {
				val++
			}

			var repr string
			switch val {
			case 3:
				repr = "SP"
			default:
				repr = fmt.Sprintf("R%d", val)
			}

			generatedArg = testcaseArg{
				val:  val,
				repr: repr,
			}

		case common.ArgKindFPReg:
			// remove F0
			val := uint32(rng.Intn(31)) + 1
			generatedArg = testcaseArg{
				val:  val,
				repr: fmt.Sprintf("F%d", val),
			}

		case common.ArgKindFCCReg:
			// remove FCC0
			val := uint32(rng.Intn(7)) + 1
			generatedArg = testcaseArg{
				val:  val,
				repr: fmt.Sprintf("FCC%d", val),
			}

		case common.ArgKindSignedImm, common.ArgKindUnsignedImm:
			valueRange := int64(1) << a.TotalWidth()

			var lowerBound int64
			if a.Kind == common.ArgKindUnsignedImm {
				lowerBound = 0
			} else {
				lowerBound = -(1 << (a.TotalWidth() - 1))
			}

			// ensure non-zero value
			val := int64(0)
			for val == 0 {
				val = lowerBound + rng.Int63n(valueRange)
			}

			generatedArg = testcaseArg{
				val:  uint32(val),
				repr: fmt.Sprintf("$%d", val),
			}
		}

		args[i] = generatedArg
	}

	expectedInsnWord := d.Word
	for i, tca := range args {
		a := d.Format.Args[i]

		remainingBits := a.TotalWidth()
		for _, s := range a.Slots {
			remainingBits -= s.Width

			slotWidthMask := (uint32(1) << s.Width) - 1
			slotVal := (tca.val >> remainingBits) & slotWidthMask

			expectedInsnWord |= slotVal << s.Offset
		}
	}

	return testcaseData{
		mnemonic:         common.GoAnameForInsn(d.Mnemonic)[1:], // strip the "A" prefix
		args:             args,
		expectedInsnWord: expectedInsnWord,
	}
}

func rngFromInsnDescription(d *common.InsnDescription) *rand.Rand {
	// hash the mnemonic for random seed
	// the first few bytes are enough
	h := sha256.Sum256([]byte(d.Mnemonic))
	seed := int64(h[0])<<56 |
		int64(h[1])<<48 |
		int64(h[2])<<40 |
		int64(h[3])<<32 |
		int64(h[4])<<24 |
		int64(h[5])<<16 |
		int64(h[6])<<8 |
		int64(h[7])

	s := rand.NewSource(seed)
	return rand.New(s)
}
