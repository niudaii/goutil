package httputil

import (
	"strings"

	"github.com/imroc/req/v3"
	"github.com/niudaii/goutil/constants"
)

var (
	toHttps = []string{
		"sent to HTTPS port",
		"This combination of host and port requires TLS",
		"Instead use the HTTPS scheme to",
		"This web server is running in SSL mode",
	}
)

func FirstGet(client *req.Client, url string) (resp *req.Response, err error) {
	request := client.R()
	var scheme string
	var flag bool
	if !strings.HasPrefix(url, "http") {
		scheme = constants.HTTP
		resp, err = request.Get(scheme + url)
		if err != nil {
			scheme = constants.HTTPS
			flag = true
		} else {
			for _, str := range toHttps {
				if strings.Contains(resp.String(), str) {
					scheme = constants.HTTPS
					flag = true
					break
				}
			}
		}
	} else if strings.HasPrefix(url, constants.HTTP) {
		resp, err = request.Get(url)
		if err != nil {
			scheme = constants.HTTPS
			url = url[7:]
			flag = true
		} else {
			for _, str := range toHttps {
				if strings.Contains(resp.String(), str) {
					scheme = constants.HTTPS
					url = url[7:]
					flag = true
				}
			}
		}
	} else {
		flag = true
	}
	if flag {
		resp, err = request.Get(scheme + url)
	}
	return
}
