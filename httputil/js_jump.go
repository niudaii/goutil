package httputil

import (
	"net"
	"regexp"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/niudaii/goutil/validator"
)

var (
	reg1 = regexp.MustCompile(`(?i)<meta.*?http-equiv=.*?refresh.*?url=(.*?)/?>`)

	reg2 = regexp.MustCompile(`(?i)[window\.]?location[\.href]?.*?=.*?["'](.*?)["']`)

	reg3 = regexp.MustCompile(`(?i)[window\.]?location\.replace\(['"](.*?)['"]\)`)
)

var (
	regHost = regexp.MustCompile(`(?i)https?://(.*?)/`)
)

func JsJump(resp *req.Response) (jumpUrl string) {
	if len(resp.String()) > 2000 {
		return
	}
	res := regexJsJump(resp)
	if res != "" {
		res = strings.TrimSpace(res)
		res = strings.ReplaceAll(res, "\"", "")
		res = strings.ReplaceAll(res, "'", "")
		res = strings.ReplaceAll(res, "./", "/")
		if strings.HasPrefix(res, "http") {
			matches := regHost.FindAllStringSubmatch(res, -1)
			if len(matches) > 0 {
				var ip net.IP
				if strings.Contains(matches[0][1], ":") {
					ip = net.ParseIP(strings.Split(matches[0][1], ":")[0])
				} else {
					ip = net.ParseIP(matches[0][1])
				}
				if validator.IsInnerIp(ip.String()) {
					baseURL := resp.Request.URL.Host
					res = strings.ReplaceAll(res, matches[0][1], baseURL)
				}
			}
			jumpUrl = res
		} else if strings.HasPrefix(res, "/") {
			// 前缀存在 / 时拼接绝对目录
			baseURL := resp.Request.URL.Scheme + "://" + resp.Request.URL.Host
			jumpUrl = baseURL + res
		} else {
			// 前缀不存在 / 时拼接相对目录
			baseURL := resp.Request.URL.Scheme + "://" + resp.Request.URL.Host + resp.Request.URL.Path
			if !strings.HasSuffix(baseURL, "/") {
				baseURL += "/"
			}
			baseURL = strings.ReplaceAll(baseURL, "./", "")
			jumpUrl = baseURL + res
		}
	}
	return
}

func regexJsJump(resp *req.Response) string {
	// 去除注释
	body := resp.String()
	body = removeComments(body)
	matches := reg1.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	matches = reg3.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	matches = reg2.FindAllStringSubmatch(body, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	return ""
}

func removeComments(str string) string {
	// 匹配 HTML 注释
	var htmlCommentsRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)
	// 先删除 HTML 注释，再删除 JavaScript 注释
	result := htmlCommentsRegex.ReplaceAllString(str, "")
	return result
}
