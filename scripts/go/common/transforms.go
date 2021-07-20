package common

import "strings"

func GoAnameForInsn(mnemonic string) string {
	// e.g. slli.w => ASLLIW
	tmp := strings.ReplaceAll(mnemonic, ".", "")
	tmp = strings.ReplaceAll(tmp, "_", "")
	tmp = strings.ToUpper(tmp)
	return "A" + tmp
}
