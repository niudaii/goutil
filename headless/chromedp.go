package headless

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"time"
)

type ChromedpOptions struct {
	Headless   bool   `json:"headless"`
	Proxy      string `json:"proxy"`
	ChromePath string `json:"chromePath"`
	MaxRuntime int    `json:"maxRuntime"`
}

func NewChromedp(options *ChromedpOptions) (ctx context.Context, cancel context.CancelFunc) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", options.Headless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.DisableGPU,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-images", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"), // 禁用 blink 特征，绕过了加速乐检测
	)
	if options.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(options.Proxy))
	}
	if options.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(options.ChromePath))
	}
	ctx, _ = chromedp.NewExecAllocator(
		context.Background(),
		opts...,
	)
	ctx, cancel = chromedp.NewContext(
		ctx,
	)
	ctx, cancel = context.WithTimeout(ctx, time.Duration(options.MaxRuntime)*time.Second)
	return
}

func GetHeaderString(headers map[string]interface{}) (headerString string) {
	for k, v := range headers {
		headerString += fmt.Sprintf("%v: %v\n", k, v)
	}
	return headerString
}

func GetHeaderMap(headers map[string]interface{}) (headerMap map[string][]string) {
	headerMap = make(map[string][]string)
	for k, v := range headers {
		headerMap[k] = append(headerMap[k], fmt.Sprintf("%v", v))
	}
	return headerMap
}

func GetHeaderMapSingleValue(headers map[string]interface{}) (headerMap map[string]string) {
	headerMap = make(map[string]string)
	for k, v := range headers {
		headerMap[k] = fmt.Sprintf("%v", v)
	}
	return headerMap
}
