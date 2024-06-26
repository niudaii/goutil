package structutil

import (
	"fmt"
	"testing"
)

func TestToMap(t *testing.T) {
	a := struct {
		A string
		b string
	}{
		"123",
		"456",
	}
	fmt.Println(ToMap(a))
}
