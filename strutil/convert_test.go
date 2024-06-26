package strutil

import (
	"fmt"
	"testing"
)

func TestMustToMap(t *testing.T) {
	a := ""
	fmt.Println(MustToMap(a))
}
