package fileutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
)

// ColumnMapping 定义了 CSV 列号到结构体字段名的映射
type ColumnMapping struct {
	ColumnIndex int
	FieldName   string
}

// ParseCSVToStructs 解析 CSV 文件，并将数据填充到指定的结构体切片中。
func ParseCSVToStructs(filePath string, out interface{}, mappings []ColumnMapping) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comment = '#'

	// 获取 out 的类型信息，确保它是指向切片的指针
	sliceVal := reflect.ValueOf(out)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("out argument must be a slice pointer")
	}

	sliceElemType := sliceVal.Elem().Type().Elem()

	// 读取 CSV 数据行，并填充到切片中
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		newElem := reflect.New(sliceElemType).Elem()
		for _, mapping := range mappings {
			if mapping.ColumnIndex >= 0 && mapping.ColumnIndex < len(record) {
				fieldValue := record[mapping.ColumnIndex]
				fieldVal := newElem.FieldByName(mapping.FieldName)
				if fieldVal.IsValid() && fieldVal.CanSet() {
					switch fieldVal.Kind() {
					case reflect.String:
						fieldVal.SetString(fieldValue)
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						intVal, _ := strconv.ParseInt(fieldValue, 10, 64)
						fieldVal.SetInt(intVal)
					case reflect.Float32, reflect.Float64:
						floatVal, _ := strconv.ParseFloat(fieldValue, 64)
						fieldVal.SetFloat(floatVal)
					}
				}
			}
		}
		sliceVal.Elem().Set(reflect.Append(sliceVal.Elem(), newElem))
	}

	return nil
}

func WriteCSV(filename string, columns []string, data [][]string) (err error) {

}
