package common

import (
	"bytes"
	"fmt"
	"go/format"
)

type EmitterCtx struct {
	buf bytes.Buffer
}

func (c *EmitterCtx) Emit(format string, a ...interface{}) {
	fmt.Fprintf(&c.buf, format, a...)
}

func (c *EmitterCtx) Finalize() []byte {
	result, err := format.Source(c.buf.Bytes())
	if err != nil {
		panic(err)
	}

	return result
}
