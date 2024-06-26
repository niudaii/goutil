package networkx

import (
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"time"
)

func NewDialer(proxyAddr string, timeout time.Duration) (dialer proxy.Dialer, err error) {
	if proxyAddr == "" {
		dialer = &net.Dialer{
			Timeout: timeout,
		}
	} else {
		dialer, err = proxyDialerFromURL(proxyAddr, timeout)
	}
	return
}

func proxyDialerFromURL(proxyAddr string, timeout time.Duration) (dialer proxy.Dialer, err error) {
	dialer = &net.Dialer{
		Timeout: timeout,
	}
	var proxyURL *url.URL
	proxyURL, err = url.Parse(proxyAddr)
	if err != nil {
		return
	}
	if proxyURL.Scheme == "socks5" {
		dialer, err = proxy.FromURL(proxyURL, dialer)
	} else {
		err = fmt.Errorf("unsupported proxy scheme: %v", proxyURL.Scheme)
	}
	return
}
