package cryptor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAesEcbDecrypt(t *testing.T) {
	data := "yuan@info"
	key := "90cea1szcefb2461"

	encrypted, _ := AesEcbEncrypt([]byte(data), []byte(key))
	decrypted, _ := AesEcbDecrypt(encrypted, []byte(key))
	fmt.Println(Base64StdEncode(string(encrypted)), string(decrypted))

	assert.Equal(t, data, string(decrypted))
}

func TestAesCbcDecrypt(t *testing.T) {
	data := "hello"
	key := "abcdefghijklmnop"

	encrypted := AesCbcEncrypt([]byte(data), []byte(key))
	decrypted := AesCbcDecrypt(encrypted, []byte(key))

	assert.Equal(t, data, string(decrypted))
}
