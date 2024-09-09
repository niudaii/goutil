package cryptor

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	uuid "github.com/satori/go.uuid"
	"io"
)

func Padding(plainText []byte, blockSize int) []byte {
	//计算要填充的长度
	n := blockSize - len(plainText)%blockSize
	//对原来的明文填充 n 个 n
	temp := bytes.Repeat([]byte{byte(n)}, n)
	plainText = append(plainText, temp...)
	return plainText
}

func ShiroAesCbcEncrypt(key []byte, Content []byte) string {
	block, _ := aes.NewCipher(key)
	Content = Padding(Content, block.BlockSize())
	iv := uuid.NewV4().Bytes()                     //指定初始向量 vi，长度和 block 的块尺寸一致
	blockMode := cipher.NewCBCEncrypter(block, iv) //指定 CBC 分组模式，返回一个 BlockMode 接口对象
	cipherText := make([]byte, len(Content))
	blockMode.CryptBlocks(cipherText, Content) //加密数据
	return base64.StdEncoding.EncodeToString(append(iv[:], cipherText[:]...))
}

func ShiroAesGcmEncrypt(data, key []byte) (encrypted string, err error) {
	block, _ := aes.NewCipher(key)
	nonce := make([]byte, 16)
	io.ReadFull(rand.Reader, nonce)
	aesgcm, _ := cipher.NewGCMWithNonceSize(block, 16)
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}
