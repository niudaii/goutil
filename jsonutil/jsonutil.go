package jsonutil

import (
	"bytes"
	"encoding/json"
	"github.com/zp857/goutil/constants"
	"strings"
)

func MustPretty(v any) (out string) {
	var err error
	out, err = Pretty(v)
	if err != nil {
		panic(err)
	}
	return
}

func Pretty(v any) (out string, err error) {
	byteBuf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteBuf)
	encoder.SetEscapeHTML(false) // 不转义特殊字符
	encoder.SetIndent(constants.EmptyString, constants.DoubleSpace)
	err = encoder.Encode(v)
	if err != nil {
		return
	}
	out = byteBuf.String()
	out = strings.TrimSpace(out)
	return
}

func IsJSON(s string) bool {
	if s == "" {
		return false
	}
	return json.Valid([]byte(s))
}
