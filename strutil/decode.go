package strutil

import "golang.org/x/text/encoding/simplifiedchinese"

const (
	UTF8 = Charset("UTF-8")
	GBK  = Charset("GBK")
)

type Charset string

func Byte2String(byte []byte, charset Charset) string {
	var str string
	switch charset {
	case GBK:
		var decodeBytes, _ = simplifiedchinese.GBK.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}
