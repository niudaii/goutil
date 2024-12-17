package httpreq

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/niudaii/goutil/constants"
)

type Options struct {
	Proxy   string   `yaml:"proxy" json:"proxy"`
	Timeout int      `yaml:"timeout" json:"timeout"`
	Headers []string `yaml:"headers" json:"headers"`
	DumpAll bool     `yaml:"dumpAll" json:"dumpAll"`
	Retry   int      `yaml:"retry" json:"retry"`
}

const (
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36"
	defaultTimeout   = 10 * time.Second
)

func NewReqClient(options *Options) *req.Client {
	reqClient := req.C().EnableDumpEachRequest()
	reqClient.SetCommonHeaders(map[string]string{
		constants.UAHeader: defaultUserAgent,
	})
	reqClient.EnableInsecureSkipVerify()
	reqClient.GetTLSClientConfig().InsecureSkipVerify = true
	reqClient.GetTLSClientConfig().MinVersion = tls.VersionTLS10
	reqClient.SetRedirectPolicy(req.AlwaysCopyHeaderRedirectPolicy(constants.CookieHeader))
	if options.Proxy != "" {
		reqClient.SetProxyURL(options.Proxy)
	}
	if options.Timeout > 0 {
		reqClient.SetTimeout(time.Duration(options.Timeout) * time.Second)
	} else {
		reqClient.SetTimeout(defaultTimeout)
	}
	var key, value string
	for _, header := range options.Headers {
		tokens := strings.SplitN(header, ":", 2)
		if len(tokens) < 2 {
			continue
		}
		key = strings.TrimSpace(tokens[0])
		value = strings.TrimSpace(tokens[1])
		reqClient.SetCommonHeader(key, value)
	}
	if options.DumpAll {
		reqClient.EnableDumpAll()
	}
	if options.Retry > 0 {
		reqClient.SetCommonRetryCount(options.Retry)
	}
	return reqClient
}
