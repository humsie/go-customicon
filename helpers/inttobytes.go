package helpers

import (
	"math/big"
)

func Int32toBytes(value int32) []byte {
	length := 4
	b := big.NewInt(int64(value)).Bytes()
	if len(b) >= length {
		return b
	}
	return append(make([]byte, length-len(b)), b...)
}
