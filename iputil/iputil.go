package iputil

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
)

func ExtractPortFromUrl(rawUrl string) (port string, err error) {
	isIpv6 := isIPv6(rawUrl)
	if isIpv6 {
		return getPortFromIPv6(rawUrl)
	}
	return getPortFromCommonUrl(rawUrl)
}

func isIPv6(rawUrl string) bool {
	// Simplify scheme trimming using a single regex
	trimUrl := regexp.MustCompile(`^https?://`).ReplaceAllString(rawUrl, "")
	host, _, err := net.SplitHostPort(trimUrl)
	if err != nil {
		host = trimUrl
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil // Check if it's not an IPv4 address
}

func getPortFromIPv6(url string) (string, error) {
	// Use a more precise regex to capture the port in IPv6 URLs
	re := regexp.MustCompile(`\[.*\]:(\d+)$`)
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", fmt.Errorf("port not found in URL")
	}
	return match[1], nil
}

func getPortFromCommonUrl(rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	port := parsedUrl.Port()
	if port == "" {
		if parsedUrl.Scheme == "https" {
			return "443", nil
		}
		return "80", nil
	}
	return port, nil
}
