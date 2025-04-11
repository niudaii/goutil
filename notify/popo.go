package notify

import (
	"time"

	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

type PopoRunner struct {
	webhookURL string
	sign       string
	client     *req.Client
	logger     *zap.SugaredLogger
}

func NewPopoRunner(webhookURL, sign string) *PopoRunner {
	return &PopoRunner{
		webhookURL: webhookURL,
		sign:       sign,
		client:     req.C(),
		logger:     zap.L().Sugar(),
	}
}

func (r *PopoRunner) Send(msg string) {
	data := map[string]any{
		"message": msg,
	}
	if r.sign != "" {
		data["signData"] = r.sign
		data["timestamp"] = time.Now().Unix()
	}
	request := r.client.R()
	request.SetHeader("Content-Type", "application/json")
	request.SetBodyJsonMarshal(data)
	_, err := request.Post(r.webhookURL)
	if err != nil {
		zap.L().Sugar().Error(err)
	}
}
