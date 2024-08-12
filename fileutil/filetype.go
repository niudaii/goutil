package fileutil

import "github.com/gabriel-vasile/mimetype"

func GetFileType(filePath string) string {
	mtype, err := mimetype.DetectFile(filePath)
	if err != nil {
		return ""
	}
	return mtype.String()
}
