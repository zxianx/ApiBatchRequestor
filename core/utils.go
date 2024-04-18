package core

import (
    "fmt"
    "net/url"
)

func appendParamsToURL(appendParams map[string]interface{}, getUrl string) (string, error) {
    // 解析原始URL
    u, err := url.Parse(getUrl)
    if err != nil {
        return "", err
    }

    // 获取原始URL中的查询参数
    queryParams, err := url.ParseQuery(u.RawQuery)
    if err != nil {
        return "", err
    }

    // 将appendParams中的参数添加到查询参数中
    for key, value := range appendParams {
        // 如果value为数组类型，则将每个元素作为参数添加到查询参数中
        switch value := value.(type) {
        case string:
            queryParams.Add(key, value)
        case int, int64, bool, float64, float32, uint:
            queryParams.Add(key, fmt.Sprint(value))
        default:
            return "", fmt.Errorf("unsupported parameter type: %T", value)
        }
    }

    // 将补充后的查询参数重新设置到URL中
    u.RawQuery = queryParams.Encode()

    // 返回新的URL字符串
    return u.String(), nil
}
