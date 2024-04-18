package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strconv"
)

const (
    port = 8081
)

type Response struct {
    ParamLen int    `json:"paramLen"`
    Param    string `json:"param"`
}

func main() {
    // 注册处理函数
    http.HandleFunc("/getDemo", getDemoHandler)
    http.HandleFunc("/postDemo", postDemoHandler)
    // GET/POST demoHandler 返回接口请求原始参数(param)及参数长度(paramLen)
    // eg. get  localhost:8081/getDemo?a=1&b=abc        =>    {"paramLen":9,"param":"a=1&b=abc"}
    // eg. post  localhost:8081/postDemo    {"a":1}     =>  {"paramLen":7,"param":"{\"a\":1}"}

    // 启动HTTP服务器
    err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
    if err != nil {
        fmt.Println("HTTP server failed to start:", err)
    }
}

func getDemoHandler(w http.ResponseWriter, r *http.Request) {
    // 获取请求参数
    params := r.URL.RawQuery

    // 获取原请求参数的长度
    paramLen := len(params)

    // 将参数转换为JSON格式
    response := Response{
        ParamLen: paramLen,
        Param:    fmt.Sprintf("%v", params),
    }

    // 将响应转换为JSON字符串
    jsonResponse, err := json.Marshal(response)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 设置响应头
    w.Header().Set("Content-Type", "application/json")

    // 写入响应
    w.Write(jsonResponse)
}

func postDemoHandler(w http.ResponseWriter, r *http.Request) {
    // 读取请求体
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer r.Body.Close()

    // 将参数转换为JSON格式
    response := Response{
        ParamLen: len(body),
        Param:    string(body),
    }

    // 将响应转换为JSON字符串
    jsonResponse, err := json.Marshal(response)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 设置响应头
    w.Header().Set("Content-Type", "application/json")

    // 写入响应
    w.Write(jsonResponse)
}
