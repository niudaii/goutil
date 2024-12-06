package fileutil

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func IsDir(path string) bool {
	file, err := os.Stat(path)
	if err != nil {
		return false
	}
	return file.IsDir()
}

func CreateFile(path string) bool {
	file, err := os.Create(path)
	if err != nil {
		return false
	}

	defer file.Close()
	return true
}

func CreateDir(absPath string) error {
	return os.MkdirAll(absPath, os.ModePerm)
}

func RemoveFile(path string) error {
	return os.Remove(path)
}

func ReadFile(path string) ([]byte, error) {
	bytes, err := os.ReadFile(path)
	return bytes, err
}

func ReadFileToString(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ReadFileByLine 按行读取文件
func ReadFileByLine(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make([]string, 0)
	buf := bufio.NewReader(f)

	for {
		line, _, err := buf.ReadLine()
		l := string(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if l == "" {
			continue
		}
		result = append(result, l)
	}

	return result, nil
}

// WriteJSON 将对象 json 序列化后写入文件
func WriteJSON(filename string, data interface{}) (err error) {
	var bytes []byte
	bytes, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	err = WriteFile(filename, string(bytes))
	return
}

// WriteCSV 将二维字符串数组写入文件
func WriteCSV(filename string, data [][]string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用 GBK 编码来写入 CSV 文件
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入 CSV 数据
	for _, row := range data {
		if err = writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile 将字符串写入文件
func WriteFile(filename string, data string) (err error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(data)
	if err != nil {
		return
	}
	err = writer.Flush()
	return
}

// AppendFile 将字符串追加写入文件
func AppendFile(filename string, data string) (err error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(data)
	if err != nil {
		return
	}
	err = writer.Flush()
	return
}

// WritePath 自动创建目录并将字符串写入文件
func WritePath(path string, data string) (err error) {
	b := IsExist(path)
	var f *os.File
	if b {
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	} else {
		err = CreateDir(filepath.Dir(path))
		if err != nil {
			return
		}
		f, err = os.Create(path)
	}
	if err != nil {
		return
	}
	defer f.Close()
	_, err = f.WriteString(data)
	return
}

// GetAllFile 递归获取指定目录下所有文件
func GetAllFile(dirPath string) (results []string, err error) {
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			results = append(results, path)
		}
		return nil
	})
	return
}
