package httpreq

import (
	"github.com/imroc/req/v3"
	"github.com/niudaii/goutil/constants"
)

func GetHeaderString(resp *req.Response) (headerString string) {
	// req
	headerString = "request header =>\n"
	headerString += resp.Request.HeaderToString()
	cookieString := "Cookie: "
	for _, cookie := range resp.Response.Request.Cookies() {
		cookieString += cookie.String() + "; "
	}
	headerString += cookieString
	headerString += "\n\n"
	// resp
	headerString += "response header =>\n"
	headerMap := map[string]string{}
	for k := range resp.Header {
		if k != constants.SetCookieHeader {
			headerMap[k] = resp.Header.Get(k)
		}
	}
	for _, ck := range resp.Cookies() {
		headerMap[constants.SetCookieHeader] += ck.String() + ";"
	}
	for k, v := range headerMap {
		headerString += k + ": " + v + "\n"
	}
	return headerString
}

func GetHeaderMap(resp *req.Response) (headerMap map[string][]string) {
	headerMap = make(map[string][]string)
	for k := range resp.Header {
		if k != constants.SetCookieHeader {
			headerMap[k] = append(headerMap[k], resp.Header.Get(k))
		}
	}
	for _, ck := range resp.Cookies() {
		headerMap[constants.SetCookieHeader] = append(headerMap[constants.SetCookieHeader], ck.String())
	}
	return headerMap
}

func GetCert(resp *req.Response) (cert string) {
	if resp.TLS != nil {
		cert = resp.TLS.PeerCertificates[0].Subject.CommonName
		// fmt.Println(jsonutil.MustPretty(resp.TLS.PeerCertificates[0]))
	}
	return cert
}
