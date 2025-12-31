package jsonutil

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/niudaii/goutil/constants"
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

// MustCompress 对应的 Panic 版本，方便在确定数据没问题时链式调用
func MustCompress(v any) string {
	out, err := Compress(v)
	if err != nil {
		panic(err)
	}
	return out
}

// Compress 压缩 JSON：去除空格和换行，同时禁止 HTML 转义
func Compress(v any) (string, error) {
	// 优化 1: 直接声明 Buffer 零值即可，不需要 bytes.NewBuffer([]byte{})
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)

	// 核心重点: 禁止转义 HTML 字符 (<, >, &)，这对 Prompt 中的 URL 和代码片段非常重要
	// 如果用 json.Marshal，这些字符会被转义成 \u003c 等，浪费 Token
	encoder.SetEscapeHTML(false)

	// 优化 2: Encoder 默认输出就是非缩进的，所以其实不需要显式调用 SetIndent("", "")
	// encoder.SetIndent("", "")

	err := encoder.Encode(v)
	if err != nil {
		return "", err
	}

	// 优化 3: json.Encoder.Encode 默认会在末尾追加一个 '\n'
	// 对于 Prompt 拼接来说，这个换行符是不需要的，必须 Trim 掉
	return strings.TrimSpace(buf.String()), nil
}
