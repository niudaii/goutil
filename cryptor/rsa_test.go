package cryptor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRsaEncrypt(t *testing.T) {
	t.Parallel()
	err := GenerateRsaKey(4096, "rsa_private.pem", "rsa_public.pem")
	if err != nil {
		t.FailNow()
	}
	data := []byte("hello world")
	encrypted := RsaEncrypt(data, "rsa_public.pem")
	decrypted := RsaDecrypt(encrypted, "rsa_private.pem")

	assert.Equal(t, string(data), string(decrypted))
}
