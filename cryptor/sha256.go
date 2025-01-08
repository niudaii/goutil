package cryptor

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256(plaintext []byte) string {
	hash := sha256.Sum256(plaintext)
	return hex.EncodeToString(hash[:])
}

func VerifySha256(hash string, password string) bool {
	passwdHash := Sha256([]byte(password))
	if hash == passwdHash {
		return true
	} else {
		return false
	}
}
