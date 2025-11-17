package core

import (
	"fmt"
	"net/http"
	"strings"
)

type ReqHeader struct {
	header   http.Header
	paramPos map[string]int
}

// ParseReqHeader 将字符串头转换为结构体
// headerStr 格式: "header1:abc\theader2=$3\theader3=value"
func ParseReqHeader(headerStr string) (*ReqHeader, error) {
	if headerStr == "" {
		return nil, nil
	}
	result := &ReqHeader{
		header: make(http.Header),
		// paramPos 初始为nil，只有发现动态头时才创建map
	}

	// 按制表符分割不同的头
	headers := strings.Split(headerStr, "\t")
	hasDynamicHeader := false

	for _, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}

		// 分割键值对
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s", header)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("header key cannot be empty")
		}

		// 检查是否为动态头（包含$符号的参数）
		if strings.HasPrefix(value, "$") {
			// 提取参数位置，例如 $3 中的 3
			var paramNum int
			_, err := fmt.Sscanf(value, "$%d", &paramNum)
			if err == nil && paramNum > 0 {
				// 如果是第一个动态头，初始化paramPos
				if !hasDynamicHeader {
					result.paramPos = make(map[string]int)
					hasDynamicHeader = true
				}
				result.paramPos[key] = paramNum
			}
			// 在header中存储模板值（即使是动态头也存储）
			result.header.Add(key, value)
		} else {
			// 普通头
			result.header.Add(key, value)
		}
	}

	return result, nil
}

// CompleteReqHeader 补全请求头（原地修改）
// reqHeader: 需要补全的请求头（会被直接修改）
// reqHeaderStruct: 包含模板和动态头信息的结构体
// paramsStr: 参数字符串，如 "value1,value2,value3"
// paramSeparator: 参数分隔符，如 ",", "|", " " 等
func CompleteReqHeader(reqHeader http.Header, reqHeaderStruct *ReqHeader, paramsStr, paramSeparator string) {
	// 如果没有提供 ReqHeaderStruct，直接返回
	if reqHeaderStruct == nil {
		return
	}

	// 添加或更新模板头
	for key, values := range reqHeaderStruct.header {
		// 删除原有的该头信息，然后用模板值覆盖
		reqHeader.Del(key)
		for _, value := range values {
			reqHeader.Add(key, value)
		}
	}

	// 处理动态头
	if reqHeaderStruct.paramPos != nil && paramsStr != "" {
		// 解析参数
		params := strings.Split(paramsStr, paramSeparator)

		for key, pos := range reqHeaderStruct.paramPos {
			if pos == 0 {
				reqHeader.Set(key, paramsStr)
				continue
			}
			if pos > 0 && pos <= len(params) {
				// 替换动态头的值为实际参数
				reqHeader.Set(key, params[pos-1])
			}
			// 如果参数不足，保持原来的模板值（如 $3）
		}
	}
}

func HeadersToDebugString(headers http.Header) string {
	if headers == nil || len(headers) == 0 {
		return ""
	}

	var parts []string
	for key, values := range headers {
		// 处理同一个key有多个值的情况
		valueStr := strings.Join(values, ", ")
		parts = append(parts, fmt.Sprintf("%s:%s", key, valueStr))
	}

	return strings.Join(parts, "\t")
}
