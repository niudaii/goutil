package urlutil

import (
	"fmt"
	"testing"
)

func TestGetFileExt(t *testing.T) {
	t.Run("GetFileExt", func(t *testing.T) {
		ext := GetFileExt("https://10.1.2.130/ui/bower_components/vui-bootstrap/css/vui-bootstrap.min.css")
		fmt.Println(ext)
	})

	t.Run("GetPath", func(t *testing.T) {
		path := GetPath("https://10.1.2.130/ui/bower_components/vui-bootstrap/css/vui-bootstrap.min.css")
		fmt.Println(path)

		path = GetPath("https://10.1.2.130/")
		fmt.Println(path)

		path = GetPath("https://10.1.2.130")
		fmt.Println(path)
	})
}
