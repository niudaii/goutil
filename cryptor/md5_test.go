package cryptor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMd5String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "e10adc3949ba59abbe56e057f20f883e", Md5String("123456"))
}

func TestMd5File(t *testing.T) {
	t.Parallel()

	fileMd5, err := Md5File("./md5.go")
	assert.NotNil(t, fileMd5)
	assert.Nil(t, err)
}
