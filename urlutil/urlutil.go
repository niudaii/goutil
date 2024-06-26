package urlutil

import (
	"fmt"
	"github.com/zp857/goutil/slice"
	"github.com/zp857/goutil/validator"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

var DefaultFilterExts = []string{".3g2", ".3gp", ".7z", ".apk", ".arj", ".avi", ".axd", ".bmp", ".csv", ".deb", ".dll", ".doc", ".drv", ".eot", ".exe", ".flv", ".gif", ".gifv", ".gz", ".h264", ".ico", ".iso", ".jar", ".jpeg", ".jpg", ".lock", ".m4a", ".m4v", ".map", ".mkv", ".mov", ".mp3", ".mp4", ".mpeg", ".mpg", ".msi", ".ogg", ".ogm", ".ogv", ".otf", ".pdf", ".pkg", ".png", ".ppt", ".psd", ".rar", ".rm", ".rpm", ".svg", ".swf", ".sys", ".tar.gz", ".tar", ".tif", ".tiff", ".ttf", ".vob", ".wav", ".webm", ".webp", ".wmv", ".woff", ".woff2", ".xcf", ".xls", ".xlsx", ".zip", ".xsd", ".dtd", ".cab",
	".css", ".js", ".vue",
}

func ExtensionFilter(rawURL string) bool {
	ext := GetFileExt(rawURL)
	if ext == "" {
		return true
	}
	if slice.Contain(DefaultFilterExts, ext) {
		return false
	}
	return true
}

func GetFileName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	file := filepath.Base(u.Path)
	return file
}

func GetFileWithoutExt(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	file := filepath.Base(u.Path)
	ext := filepath.Ext(file)
	file = strings.TrimSuffix(file, ext)
	return file
}

func GetFileExt(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	ext := filepath.Ext(u.Path)
	ext = strings.ToLower(ext)
	return ext
}

func GetUniqueURLs(rawURLs []string) (urls []string) {
	for _, rawURL := range rawURLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		u.RawQuery = ""
		urls = append(urls, u.String())
	}
	urls = slice.Unique(urls)
	return
}

func RemoveQueryParams(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// 清除查询参数
	parsedURL.RawQuery = ""
	return parsedURL.String(), nil
}

func GetBaseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%v://%v", u.Scheme, u.Host)
}

func GetHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

func GetHostname(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func GetPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Path
}

func IsRootURL(rawURL string) bool {
	path := GetPath(rawURL)
	if path == "" {
		return true
	}
	return false
}

func TrimURL(rawURL string) string {
	path := GetPath(rawURL)
	if path == "/" {
		return strings.TrimSuffix(rawURL, "/")
	}
	return rawURL
}

func GetDomainIP(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if validator.IsDomain(u.Hostname()) {
		var ipRecords []net.IP
		ipRecords, err = net.LookupIP(u.Hostname())
		if err != nil {
			return ""
		}
		return ipRecords[0].String()
	}
	return u.Hostname()
}
