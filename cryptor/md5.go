package cryptor

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func Md5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func Md5File(filename string) (string, error) {
	if fileInfo, err := os.Stat(filename); err != nil {
		return "", err
	} else if fileInfo.IsDir() {
		return "", nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()

	chunkSize := 65536
	for buf, reader := make([]byte, chunkSize), bufio.NewReader(file); ; {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		hash.Write(buf[:n])
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	return checksum, nil
}
