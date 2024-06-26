package cryptor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBase64StdEncode(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "YWRtaW4=", Base64StdEncode("admin"))
}

func TestBase64StdDecode(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "admin", Base64StdDecode("YWRtaW4="))
}
