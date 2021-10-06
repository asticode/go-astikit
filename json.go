package astikit

import (
	"bytes"
	"encoding/json"
)

func JSONEqual(a, b interface{}) bool {
	ba, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ba, bb)
}
