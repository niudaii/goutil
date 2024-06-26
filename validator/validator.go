package validator

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	// URL
	urlMatcher = regexp.MustCompile(`^((ftp|http|https?):\/\/)?(\S+(:\S*)?@)?((([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(([a-zA-Z0-9]+([-\.][a-zA-Z0-9]+)*)|((www\.)?))?(([a-z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-z\x{00a1}-\x{ffff}]{2,}))?))(:(\d{1,5}))?((\/|\?|#)[^\s]*)?$`)
	// 域名
	domainMatcher = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	// 邮箱
	emailMatcher = regexp.MustCompile(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`)
	// 中文
	chineseMatcher = regexp.MustCompile("[\u4e00-\u9fa5]")
)

func IsIp(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	return ip != nil
}

func IsIpV4(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	return strings.Contains(ipstr, ".")
}

func IsIpV6(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	return strings.Contains(ipstr, ":")
}

func IsUrl(str string) bool {
	if str == "" || len(str) >= 2083 || len(str) <= 3 || strings.HasPrefix(str, ".") {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}

	return urlMatcher.MatchString(str)
}

func IsDomain(dns string) bool {
	return domainMatcher.MatchString(dns)
}

func IsEmail(email string) bool {
	return emailMatcher.MatchString(email)
}

func ContainChinese(s string) bool {
	return chineseMatcher.MatchString(s)
}

func IsEmptyString(str string) bool {
	return len(str) == 0
}

func IsInnerIp(ip string) bool {
	ipAddress := net.ParseIP(ip)
	return ipAddress.IsPrivate()
}

func CheckMatch(regexps []*regexp.Regexp, input string) bool {
	if len(regexps) == 0 {
		return false
	}
	for _, regex := range regexps {
		if regex.MatchString(input) {
			return true
		}
	}
	return false
}
