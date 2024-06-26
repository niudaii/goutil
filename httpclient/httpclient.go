package httpclient

import (
	"github.com/imroc/req/v3"
	v1 "github.com/zp857/goutil/constants/v1"
	"github.com/zp857/goutil/httpreq"
	"github.com/zp857/goutil/jsonutil"
	"github.com/zp857/goutil/slice"
	"github.com/zp857/goutil/strutil"
	"go.uber.org/zap"
)

type Config struct {
	BaseUrl    string          `yaml:"baseUrl" json:"baseUrl"`
	ReqOptions httpreq.Options `yaml:"reqOptions" json:"reqOptions"`
	NoLogAPIs  []string        `yaml:"noLogAPIs" json:"noLogAPIs"`
}

type HTTPClient struct {
	config    Config
	reqClient *req.Client
	logger    *zap.SugaredLogger
}

func NewHTTPClient(config Config) *HTTPClient {
	reqClient := httpreq.NewReqClient(&config.ReqOptions)
	return &HTTPClient{
		config:    config,
		reqClient: reqClient,
		logger:    zap.L().Named(v1.HTTPClientLogger).Sugar(),
	}
}

func (c *HTTPClient) SendJSON(api string, data interface{}) {
	shouldLog := !slice.Contain(c.config.NoLogAPIs, api)
	if shouldLog {
		c.logger.Infof(v1.ClientRequest, api, jsonutil.MustPretty(data))
	}
	r := c.reqClient.R()
	r.SetBody(data)
	resp, err := r.Post(c.config.BaseUrl + api)
	if err != nil {
		c.logger.Errorf(v1.RequestError, err)
		return
	}
	if shouldLog {
		c.logger.Infof(v1.ClientResponse, resp.StatusCode, jsonutil.MustPretty(strutil.MustToMap(resp.String())))
	}
}
