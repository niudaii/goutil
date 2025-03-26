package httputil

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/beevik/etree"
)

// ParseQueryParams 解析 GET 请求的 URL 查询参数
func ParseQueryParams(urlStr string) (map[string]string, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	queryParams := make(map[string]string)
	for key, values := range parsedUrl.Query() {
		queryParams[key] = values[0] // 取第一个值
	}
	return queryParams, nil
}

// ParsePostData 解析 POST 请求的 PostData
func ParsePostData(postData, contentType string) (map[string]interface{}, error) {
	switch {
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return parseFormPostData(postData)
	case strings.Contains(contentType, "application/json"):
		return parseJSONPostData(postData)
	case strings.Contains(contentType, "/xml"):
		return parseXMLPostData(postData)
	case strings.Contains(contentType, "multipart/form-data"):
		return parseMultipartFormData(postData)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func parseFormPostData(data string) (result map[string]interface{}, err error) {
	parsedData, err := url.ParseQuery(data)
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	for key, values := range parsedData {
		result[key] = values[0] // 取第一个值
	}
	return
}

func parseJSONPostData(data string) (result map[string]interface{}, err error) {
	result = make(map[string]interface{})
	err = json.Unmarshal([]byte(data), &result)
	return
}

func parseXMLPostData(data string) (result map[string]interface{}, err error) {
	result = make(map[string]interface{})
	doc := etree.NewDocument()
	err = doc.ReadFromString(data)
	if err != nil {
		return
	}
	childs := doc.ChildElements()
	for _, child := range childs {
		result[child.Tag] = child.Text()
	}
	return
}

func parseMultipartFormData(data string) (result map[string]interface{}, err error) {
	result = make(map[string]interface{})
	boundaryIndex := strings.Index(data, "\r\n")
	if boundaryIndex == -1 {
		err = fmt.Errorf("no boundary found")
		return
	}
	boundary := data[:boundaryIndex]
	if boundary == "" {
		err = fmt.Errorf("boundary cannot be empty")
		return
	}
	// 使用边界分割表单字符串
	parts := strings.Split(data, boundary)
	for _, part := range parts {
		// 忽略空字符串和结束符
		if part == "" || strings.Contains(part, "--") {
			continue
		}
		// 分割头部和内容
		sections := strings.SplitN(part, "\r\n\r\n", 2)
		if len(sections) < 2 {
			err = fmt.Errorf("invalid part format")
			return
		}
		header := sections[0]
		value := strings.TrimSpace(sections[1])
		// 提取 name 值
		nameStart := strings.Index(header, `name="`)
		if nameStart == -1 {
			err = fmt.Errorf("name not found in header")
			return
		}
		nameStart += len(`name="`)
		nameEnd := strings.Index(header[nameStart:], `"`)
		if nameEnd == -1 {
			err = fmt.Errorf("name not properly quoted")
			return
		}
		name := header[nameStart : nameStart+nameEnd]
		// 存储结果
		result[name] = value
	}
	return
}

func GetMethod(request string) string {
	if strings.HasPrefix(request, "POST") {
		return "POST"
	}
	return "GET"
}

func ParseCustomHeaders(customHeaders []string) map[string]string {
	headerMap := make(map[string]string)
	for _, v := range customHeaders {
		if headerParts := strings.SplitN(v, ":", 2); len(headerParts) >= 2 {
			headerMap[strings.Trim(headerParts[0], " ")] = strings.Trim(headerParts[1], " ")
		}
	}
	return headerMap
}

func ParseHeaderMap(headerMap map[string]string) []string {
	headers := make([]string, 0)
	for k, v := range headerMap {
		headers = append(headers, k+": "+v)
	}
	return headers
}

func ClearPath(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}
	// 去掉查询参数
	parsedURL.RawQuery = ""
	return parsedURL.String()
}

func GetPath(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}
	return parsedURL.Path
}